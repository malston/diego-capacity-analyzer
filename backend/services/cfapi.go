// ABOUTME: Cloud Foundry API client for apps and isolation segments
// ABOUTME: Handles authentication, pagination, and data transformation

package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

type CFClient struct {
	apiURL        string
	username      string
	password      string
	token         string
	client        *http.Client
	logCache      *LogCacheClient
	skipSSLVerify bool
}

func NewCFClient(apiURL, username, password string, skipSSLValidation bool) *CFClient {
	return &CFClient{
		apiURL:        apiURL,
		username:      username,
		password:      password,
		skipSSLVerify: skipSSLValidation,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSLValidation},
			},
		},
	}
}

func (c *CFClient) Authenticate() error {
	// Get UAA URL from CF API info
	infoResp, err := c.client.Get(c.apiURL + "/v3/info")
	if err != nil {
		return fmt.Errorf("failed to get CF info: %w", err)
	}
	defer infoResp.Body.Close()

	var info struct {
		Links struct {
			Login struct {
				Href string `json:"href"`
			} `json:"login"`
		} `json:"links"`
	}

	if err := json.NewDecoder(infoResp.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to parse CF info: %w", err)
	}

	uaaURL := info.Links.Login.Href

	// If login URL not in info response, construct from API URL
	if uaaURL == "" {
		uaaURL = strings.Replace(c.apiURL, "://api.", "://login.", 1)
	}

	// Authenticate with UAA
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequest("POST", uaaURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("cf", "")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	c.token = tokenResp.AccessToken
	slog.Info("CF API authentication successful", "api_url", c.apiURL)

	// Initialize Log Cache client with the same token and SSL settings
	c.logCache = NewLogCacheClient(c.apiURL, c.token, c.skipSSLVerify)

	return nil
}

// doAuthenticatedRequest performs an HTTP request with the CF API token
func (c *CFClient) doAuthenticatedRequest(method, path string) (*http.Response, error) {
	if c.token == "" {
		return nil, fmt.Errorf("not authenticated: call Authenticate() first")
	}

	req, err := http.NewRequest(method, c.apiURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("CF API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// GetApps fetches all apps from CF API v3 with pagination
func (c *CFClient) GetApps() ([]models.App, error) {
	start := time.Now()
	var apps []models.App
	var pageCount int
	nextURL := "/v3/apps?per_page=100"

	for nextURL != "" {
		pageCount++
		resp, err := c.doAuthenticatedRequest("GET", nextURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result struct {
			Resources []struct {
				GUID          string `json:"guid"`
				Name          string `json:"name"`
				State         string `json:"state"`
				Relationships struct {
					Space struct {
						Data struct {
							GUID string `json:"guid"`
						} `json:"data"`
					} `json:"space"`
				} `json:"relationships"`
			} `json:"resources"`
			Pagination struct {
				Next struct {
					Href string `json:"href"`
				} `json:"next"`
			} `json:"pagination"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to parse apps response: %w", err)
		}

		// Fetch processes for each app to get memory, disk, and instance info
		for _, resource := range result.Resources {
			processes, err := c.getAppProcesses(resource.GUID)
			if err != nil {
				return nil, err
			}

			// Calculate totals across all processes
			var totalInstances, totalRequestedMB, totalRequestedDiskMB int
			for _, proc := range processes {
				totalInstances += proc.Instances
				totalRequestedMB += proc.Instances * proc.MemoryMB
				totalRequestedDiskMB += proc.Instances * proc.DiskMB
			}

			// Try to get actual memory from Log Cache
			totalActualMB := totalRequestedMB // Default to requested
			if c.logCache != nil && totalInstances > 0 {
				metrics, err := c.logCache.GetAppMemoryMetrics(resource.GUID)
				if err == nil && metrics.MemoryBytesAvg > 0 {
					// Convert bytes to MB
					totalActualMB = int(metrics.MemoryBytesAvg / (1024 * 1024))
					// Multiply by instances if we got per-instance average
					if metrics.InstanceCount > 0 && metrics.InstanceCount < totalInstances {
						totalActualMB = totalActualMB * totalInstances / metrics.InstanceCount
					}
				}
			}

			// Get isolation segment for the space
			isoSegName, err := c.getSpaceIsolationSegment(resource.Relationships.Space.Data.GUID)
			if err != nil || isoSegName == "" {
				// Apps without explicit isolation segment run on "default"
				isoSegName = "default"
			}

			apps = append(apps, models.App{
				Name:             resource.Name,
				GUID:             resource.GUID,
				Instances:        totalInstances,
				RequestedMB:      totalRequestedMB,
				ActualMB:         totalActualMB,
				RequestedDiskMB:  totalRequestedDiskMB,
				IsolationSegment: isoSegName,
			})
		}

		// Check for next page
		if result.Pagination.Next.Href != "" {
			// Extract path from full URL
			parsedURL, err := url.Parse(result.Pagination.Next.Href)
			if err != nil {
				return nil, fmt.Errorf("failed to parse next page URL: %w", err)
			}
			nextURL = parsedURL.Path + "?" + parsedURL.RawQuery
		} else {
			nextURL = ""
		}
	}

	slog.Info("CF API GetApps completed", "app_count", len(apps), "pages", pageCount, "duration_ms", time.Since(start).Milliseconds())
	return apps, nil
}

// getAppProcesses fetches process information for an app
func (c *CFClient) getAppProcesses(appGUID string) ([]struct {
	Type      string `json:"type"`
	Instances int    `json:"instances"`
	MemoryMB  int    `json:"memory_in_mb"`
	DiskMB    int    `json:"disk_in_mb"`
}, error) {
	resp, err := c.doAuthenticatedRequest("GET", "/v3/apps/"+appGUID+"/processes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Resources []struct {
			Type      string `json:"type"`
			Instances int    `json:"instances"`
			MemoryMB  int    `json:"memory_in_mb"`
			DiskMB    int    `json:"disk_in_mb"`
		} `json:"resources"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse processes response: %w", err)
	}

	return result.Resources, nil
}

// getSpaceIsolationSegment fetches the isolation segment name for a space
func (c *CFClient) getSpaceIsolationSegment(spaceGUID string) (string, error) {
	resp, err := c.doAuthenticatedRequest("GET", "/v3/spaces/"+spaceGUID+"/relationships/isolation_segment")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			GUID string `json:"guid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse space isolation segment response: %w", err)
	}

	// If no isolation segment, return empty
	if result.Data.GUID == "" {
		return "", nil
	}

	// Fetch the isolation segment name
	segResp, err := c.doAuthenticatedRequest("GET", "/v3/isolation_segments/"+result.Data.GUID)
	if err != nil {
		return "", err
	}
	defer segResp.Body.Close()

	var seg struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(segResp.Body).Decode(&seg); err != nil {
		return "", fmt.Errorf("failed to parse isolation segment response: %w", err)
	}

	return seg.Name, nil
}

// GetIsolationSegments fetches all isolation segments from CF API v3
func (c *CFClient) GetIsolationSegments() ([]models.IsolationSegment, error) {
	var segments []models.IsolationSegment
	nextURL := "/v3/isolation_segments?per_page=100"

	for nextURL != "" {
		resp, err := c.doAuthenticatedRequest("GET", nextURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result struct {
			Resources []struct {
				GUID string `json:"guid"`
				Name string `json:"name"`
			} `json:"resources"`
			Pagination struct {
				Next struct {
					Href string `json:"href"`
				} `json:"next"`
			} `json:"pagination"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to parse isolation segments response: %w", err)
		}

		for _, resource := range result.Resources {
			segments = append(segments, models.IsolationSegment{
				GUID: resource.GUID,
				Name: resource.Name,
			})
		}

		// Check for next page
		if result.Pagination.Next.Href != "" {
			parsedURL, err := url.Parse(result.Pagination.Next.Href)
			if err != nil {
				return nil, fmt.Errorf("failed to parse next page URL: %w", err)
			}
			nextURL = parsedURL.Path + "?" + parsedURL.RawQuery
		} else {
			nextURL = ""
		}
	}

	return segments, nil
}
