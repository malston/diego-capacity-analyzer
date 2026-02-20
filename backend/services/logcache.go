// ABOUTME: Log Cache API client for container metrics
// ABOUTME: Fetches actual memory usage per app from CF Log Cache

package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LogCacheClient struct {
	logCacheURL string
	token       string
	client      *http.Client
}

// LogCacheEnvelope represents a metric envelope from Log Cache
type LogCacheEnvelope struct {
	Timestamp string `json:"timestamp"`
	SourceID  string `json:"source_id"`
	Gauge     *struct {
		Metrics map[string]struct {
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
		} `json:"metrics"`
	} `json:"gauge"`
}

// LogCacheResponse represents the response from Log Cache read API
type LogCacheResponse struct {
	Envelopes struct {
		Batch []LogCacheEnvelope `json:"batch"`
	} `json:"envelopes"`
}

// AppMetrics contains memory metrics for an app
type AppMetrics struct {
	GUID           string
	MemoryBytesAvg int64
	MemoryBytesCur int64
	InstanceCount  int
}

// NewLogCacheClient creates a Log Cache client from a CF API URL
func NewLogCacheClient(cfAPIURL, token string, skipSSLValidation bool) *LogCacheClient {
	// Derive log-cache URL from CF API URL
	// api.sys.example.com -> log-cache.sys.example.com
	logCacheURL := strings.Replace(cfAPIURL, "://api.", "://log-cache.", 1)

	return &LogCacheClient{
		logCacheURL: logCacheURL,
		token:       token,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSLValidation},
			},
		},
	}
}

// SetToken updates the authentication token
func (l *LogCacheClient) SetToken(token string) {
	l.token = token
}

// GetAppMemoryMetrics fetches memory metrics for a specific app
func (l *LogCacheClient) GetAppMemoryMetrics(ctx context.Context, appGUID string) (*AppMetrics, error) {
	if l.token == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	// Fetch recent gauge envelopes (up to 100 most recent)
	endpoint := fmt.Sprintf("%s/api/v1/read/%s?envelope_types=GAUGE&limit=100",
		l.logCacheURL, appGUID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+l.token)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query log cache: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("log cache returned status %d: %s", resp.StatusCode, string(body))
	}

	var result LogCacheResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse log cache response: %w", err)
	}

	// Extract memory metrics from envelopes
	var totalMemory int64
	var count int
	instancesSeen := make(map[string]bool)

	for _, env := range result.Envelopes.Batch {
		if env.Gauge == nil {
			continue
		}

		// Look for memory or memory_bytes metric
		for name, metric := range env.Gauge.Metrics {
			if name == "memory" || name == "memory_bytes" {
				totalMemory += int64(metric.Value)
				count++
				instancesSeen[env.SourceID] = true
			}
		}
	}

	if count == 0 {
		slog.Debug("Log Cache returned no memory metrics", "app_guid", appGUID)
		return &AppMetrics{
			GUID:           appGUID,
			MemoryBytesAvg: 0,
			MemoryBytesCur: 0,
			InstanceCount:  0,
		}, nil
	}

	metrics := &AppMetrics{
		GUID:           appGUID,
		MemoryBytesAvg: totalMemory / int64(count),
		MemoryBytesCur: totalMemory / int64(len(instancesSeen)),
		InstanceCount:  len(instancesSeen),
	}
	slog.Debug("Log Cache metrics retrieved", "app_guid", appGUID, "avg_bytes", metrics.MemoryBytesAvg, "instances", metrics.InstanceCount)
	return metrics, nil
}

// GetAppMemoryPromQL uses PromQL endpoint for more precise queries
func (l *LogCacheClient) GetAppMemoryPromQL(ctx context.Context, appGUID string) (int64, error) {
	if l.token == "" {
		return 0, fmt.Errorf("not authenticated")
	}

	// Use PromQL to get average memory over last 5 minutes
	query := url.QueryEscape(fmt.Sprintf(`avg_over_time(memory{source_id="%s"}[5m])`, appGUID))
	endpoint := fmt.Sprintf("%s/api/v1/promql?query=%s", l.logCacheURL, query)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+l.token)

	resp, err := l.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to query log cache promql: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("log cache promql returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Result []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse promql response: %w", err)
	}

	if len(result.Data.Result) == 0 || len(result.Data.Result[0].Value) < 2 {
		return 0, nil
	}

	// Value is [timestamp, "value_string"]
	valueStr, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, nil
	}

	var value float64
	fmt.Sscanf(valueStr, "%f", &value)

	return int64(value), nil
}
