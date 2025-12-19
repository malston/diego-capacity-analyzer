// ABOUTME: Cloud Foundry API client for apps and isolation segments
// ABOUTME: Handles authentication, pagination, and data transformation

package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CFClient struct {
	apiURL   string
	username string
	password string
	token    string
	client   *http.Client
}

func NewCFClient(apiURL, username, password string) *CFClient {
	return &CFClient{
		apiURL:   apiURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
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
	return nil
}
