// ABOUTME: BOSH API client for Diego cell VM metrics
// ABOUTME: Queries BOSH Director for deployment VMs with vitals

package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/socks5-proxy"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

type BOSHClient struct {
	environment string
	clientID    string
	secret      string
	caCert      string
	deployment  string
	client      *http.Client
	token       string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex
}

func NewBOSHClient(environment, clientID, secret, caCert, deployment string) *BOSHClient {
	// Normalize environment URL - bosh cli omits protocol and sometimes port
	if environment != "" {
		// Add https:// if missing
		if !strings.HasPrefix(environment, "https://") && !strings.HasPrefix(environment, "http://") {
			environment = "https://" + environment
		}
		// Add default port :25555 if no port specified
		if u, err := url.Parse(environment); err == nil && u.Port() == "" {
			environment = environment + ":25555"
		}
	}

	tlsConfig := &tls.Config{}

	if caCert != "" {
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM([]byte(caCert)); ok {
			tlsConfig.RootCAs = certPool
		} else {
			slog.Warn("Failed to parse BOSH_CA_CERT, using InsecureSkipVerify")
			tlsConfig.InsecureSkipVerify = true
		}
	} else {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		TLSHandshakeTimeout: 30 * time.Second,
	}

	// Check for BOSH_ALL_PROXY environment variable
	if allProxy := os.Getenv("BOSH_ALL_PROXY"); allProxy != "" {
		dialContextFunc := createSOCKS5DialContextFunc(allProxy)
		if dialContextFunc != nil {
			transport.DialContext = dialContextFunc
		}
	}

	return &BOSHClient{
		environment: environment,
		clientID:    clientID,
		secret:      secret,
		caCert:      caCert,
		deployment:  deployment,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
	}
}

// SetHTTPClient allows overriding the HTTP client (useful for testing)
func (b *BOSHClient) SetHTTPClient(client *http.Client) {
	b.client = client
}

// getUAAEndpoint discovers the UAA endpoint from the BOSH Director info
func (b *BOSHClient) getUAAEndpoint() (string, error) {
	req, err := http.NewRequest("GET", b.environment+"/info", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create info request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get BOSH info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("BOSH info returned status %d: %s", resp.StatusCode, string(body))
	}

	var info struct {
		UserAuthentication struct {
			Type    string `json:"type"`
			Options struct {
				URL string `json:"url"`
			} `json:"options"`
		} `json:"user_authentication"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("failed to parse BOSH info: %w", err)
	}

	if info.UserAuthentication.Options.URL == "" {
		// Fall back to Director URL with port 8443
		parsed, err := url.Parse(b.environment)
		if err != nil {
			return "", fmt.Errorf("failed to parse environment URL: %w", err)
		}
		host := parsed.Hostname()
		return fmt.Sprintf("https://%s:8443", host), nil
	}

	return info.UserAuthentication.Options.URL, nil
}

// authenticate gets an OAuth token from BOSH's UAA
func (b *BOSHClient) authenticate() error {
	b.tokenMutex.RLock()
	if b.token != "" && time.Now().Before(b.tokenExpiry) {
		b.tokenMutex.RUnlock()
		return nil
	}
	b.tokenMutex.RUnlock()

	b.tokenMutex.Lock()
	defer b.tokenMutex.Unlock()

	// Double-check after acquiring write lock
	if b.token != "" && time.Now().Before(b.tokenExpiry) {
		return nil
	}

	uaaURL, err := b.getUAAEndpoint()
	if err != nil {
		return fmt.Errorf("failed to get UAA endpoint: %w", err)
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", uaaURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(b.clientID, b.secret)

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("UAA token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	b.token = tokenResp.AccessToken
	// Set expiry with 1 minute buffer
	b.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return nil
}

// sshKeyAllowedDirs defines directories where SSH keys are allowed to be read from.
// This prevents arbitrary file reads even when path traversal is blocked.
// Can be overridden via BOSH_SSH_KEY_ALLOWED_DIRS environment variable (colon-separated).
var sshKeyAllowedDirs = []string{
	"/var/vcap",    // BOSH-managed locations
	"/tmp",         // Temp directory
	"/var/tmp",     // Persistent temp
	"/var/folders", // macOS temp directories
}

// getSSHKeyAllowedDirs returns the list of allowed directories for SSH keys.
// Includes $HOME if set, plus default system directories.
func getSSHKeyAllowedDirs() []string {
	// Check for custom override via environment variable
	if custom := os.Getenv("BOSH_SSH_KEY_ALLOWED_DIRS"); custom != "" {
		return strings.Split(custom, ":")
	}

	dirs := make([]string, len(sshKeyAllowedDirs))
	copy(dirs, sshKeyAllowedDirs)

	// Add user's home directory if available
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, home)
	}

	return dirs
}

// isPathUnderAllowedDirs checks if absPath is under one of the allowed directories.
func isPathUnderAllowedDirs(absPath string, allowedDirs []string) bool {
	for _, dir := range allowedDirs {
		// Ensure dir ends with separator for proper prefix matching
		dirWithSep := dir
		if !strings.HasSuffix(dirWithSep, string(filepath.Separator)) {
			dirWithSep += string(filepath.Separator)
		}
		// Check if path starts with allowed dir (or equals it exactly)
		if strings.HasPrefix(absPath, dirWithSep) || absPath == dir {
			return true
		}
	}
	return false
}

// ValidateSSHKeyPath validates that an SSH key path is safe to read.
// It prevents path traversal attacks, restricts to allowed directories,
// and ensures the file exists and is readable.
func ValidateSSHKeyPath(path string) (string, error) {
	// Clean the path to remove . and .. components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts after cleaning
	// filepath.Clean resolves .. but we want to detect if they were present
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal detected in SSH key path")
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve SSH key path: %w", err)
	}

	// Verify path is within allowed directories (defense in depth)
	if !isPathUnderAllowedDirs(absPath, getSSHKeyAllowedDirs()) {
		return "", fmt.Errorf("SSH key path outside allowed directories")
	}

	// Verify the file exists and is a regular file
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("SSH key file does not exist")
		}
		return "", fmt.Errorf("cannot access SSH key file: %w", err)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("SSH key path is not a regular file")
	}

	return absPath, nil
}

// createSOCKS5DialContextFunc creates a dial function for SSH+SOCKS5 proxy connections.
// Supports format: ssh+socks5://user@host:port?private-key=/path/to/key
func createSOCKS5DialContextFunc(allProxy string) func(ctx context.Context, network, address string) (net.Conn, error) {
	// Strip ssh+ prefix if present
	allProxy = strings.TrimPrefix(allProxy, "ssh+")

	proxyURL, err := url.Parse(allProxy)
	if err != nil {
		slog.Error("Failed to parse BOSH_ALL_PROXY URL", "error", err)
		return nil
	}

	queryMap, err := url.ParseQuery(proxyURL.RawQuery)
	if err != nil {
		slog.Error("Failed to parse BOSH_ALL_PROXY query params", "error", err)
		return nil
	}

	username := ""
	if proxyURL.User != nil {
		username = proxyURL.User.Username()
	}

	proxySSHKeyPath := queryMap.Get("private-key")
	if proxySSHKeyPath == "" {
		slog.Error("BOSH_ALL_PROXY missing required 'private-key' query param")
		return nil
	}

	// Validate SSH key path to prevent path traversal attacks (Issue #70)
	validatedPath, err := ValidateSSHKeyPath(proxySSHKeyPath)
	if err != nil {
		slog.Error("Invalid SSH private key path", "error", err)
		return nil
	}

	proxySSHKey, err := os.ReadFile(validatedPath)
	if err != nil {
		slog.Error("Failed to read SSH private key", "error", err)
		return nil
	}

	// Create the socks5 proxy with host key callback
	socks5Proxy := proxy.NewSocks5Proxy(proxy.NewHostKey(), log.Default(), 1*time.Minute)

	var (
		dialer proxy.DialFunc
		mut    sync.RWMutex
	)

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		mut.RLock()
		haveDialer := dialer != nil
		mut.RUnlock()

		if haveDialer {
			return dialer(network, address)
		}

		mut.Lock()
		defer mut.Unlock()
		if dialer == nil {
			proxyDialer, err := socks5Proxy.Dialer(username, string(proxySSHKey), proxyURL.Host)
			if err != nil {
				return nil, fmt.Errorf("error creating SOCKS5 dialer: %w", err)
			}
			dialer = proxyDialer
		}
		return dialer(network, address)
	}
}

// boshTask represents a BOSH async task
type boshTask struct {
	ID          int    `json:"id"`
	State       string `json:"state"`
	Description string `json:"description"`
	Result      string `json:"result"`
}

// boshVM represents a VM from the BOSH VMs endpoint
type boshVM struct {
	JobName string `json:"job_name"`
	Index   int    `json:"index"`
	ID      string `json:"id"`
	Vitals  struct {
		Mem struct {
			KB      string `json:"kb"`
			Percent string `json:"percent"`
		} `json:"mem"`
		CPU struct {
			Sys  string `json:"sys"`
			User string `json:"user"`
			Wait string `json:"wait"`
		} `json:"cpu"`
		Disk struct {
			System struct {
				Percent string `json:"percent"`
			} `json:"system"`
		} `json:"disk"`
	} `json:"vitals"`
}

func (b *BOSHClient) GetDiegoCells() ([]models.DiegoCell, error) {
	// Authenticate with UAA first
	if err := b.authenticate(); err != nil {
		return nil, fmt.Errorf("failed to authenticate with BOSH: %w", err)
	}

	// Get list of deployments to query
	deployments, err := b.getDeployments()
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	slog.Info("Found deployments to query", "count", len(deployments), "deployments", deployments)

	var allCells []models.DiegoCell
	for _, deployment := range deployments {
		slog.Debug("Querying deployment", "deployment", deployment)
		cells, err := b.getCellsForDeployment(deployment)
		if err != nil {
			slog.Warn("Failed to get cells for deployment", "deployment", deployment, "error", err)
			continue
		}
		slog.Debug("Found cells in deployment", "deployment", deployment, "count", len(cells))
		allCells = append(allCells, cells...)
	}

	if len(allCells) == 0 {
		return nil, fmt.Errorf("no Diego cells found in any deployment")
	}

	return allCells, nil
}

// getDeployments returns list of CF and isolation segment deployments
func (b *BOSHClient) getDeployments() ([]string, error) {
	req, err := http.NewRequest("GET", b.environment+"/deployments", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.token)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BOSH API returned status %d: %s", resp.StatusCode, string(body))
	}

	var deploymentList []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deploymentList); err != nil {
		return nil, fmt.Errorf("failed to parse deployments: %w", err)
	}

	// Filter for CF and isolation segment deployments
	var result []string
	for _, d := range deploymentList {
		if strings.HasPrefix(d.Name, "cf-") || strings.HasPrefix(d.Name, "p-isolation-segment") {
			result = append(result, d.Name)
		}
	}

	allDeploymentNames := make([]string, len(deploymentList))
	for i, d := range deploymentList {
		allDeploymentNames[i] = d.Name
	}
	slog.Debug("All deployments from BOSH", "deployments", allDeploymentNames)

	return result, nil
}

// getCellsForDeployment fetches Diego cells for a specific deployment
func (b *BOSHClient) getCellsForDeployment(deployment string) ([]models.DiegoCell, error) {
	reqURL := fmt.Sprintf("%s/deployments/%s/vms?format=full", b.environment, deployment)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.token)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VMs: %w", err)
	}
	defer resp.Body.Close()

	// BOSH returns 302 redirect to task, or task object directly
	var taskID int
	if resp.StatusCode == http.StatusFound {
		// Get task ID from Location header
		location := resp.Header.Get("Location")
		// Location is like /tasks/123
		fmt.Sscanf(location, "/tasks/%d", &taskID)
	} else if resp.StatusCode == http.StatusOK {
		// Parse task from response body
		var task boshTask
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			return nil, fmt.Errorf("failed to parse task response: %w", err)
		}
		taskID = task.ID
	} else {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BOSH API returned status %d: %s", resp.StatusCode, string(body))
	}

	if taskID == 0 {
		return nil, fmt.Errorf("could not determine task ID from BOSH response")
	}

	// Poll task until done
	vms, err := b.waitForTaskAndGetOutput(taskID)
	if err != nil {
		return nil, err
	}

	// Determine isolation segment from deployment name
	// p-isolation-segment-* deployments have isolated cells
	isolationSegment := "default"
	if strings.HasPrefix(deployment, "p-isolation-segment") {
		// The segment name is typically configured in the tile
		// For now, use a generic name based on deployment
		isolationSegment = "isolated"
	}

	// Log all VM job names for debugging
	var jobNames []string
	for _, vm := range vms {
		jobNames = append(jobNames, vm.JobName)
	}
	slog.Info("VMs found in deployment", "deployment", deployment, "vm_count", len(vms), "job_names", jobNames)

	var cells []models.DiegoCell
	for _, vm := range vms {
		// Include diego_cell, compute, and any job name containing "diego_cell" (e.g., isolated_diego_cell, isolated_diego_cell_small_cell)
		if vm.JobName == "diego_cell" || vm.JobName == "compute" || strings.Contains(vm.JobName, "diego_cell") {
			memoryKB := parseIntOrZero(vm.Vitals.Mem.KB)
			memoryMB := memoryKB / 1024
			memPercent := parseIntOrZero(vm.Vitals.Mem.Percent)
			cpuSys := parseFloatOrZero(vm.Vitals.CPU.Sys)

			// mem.percent from BOSH vitals is VM-level memory usage
			usedMB := (memoryMB * memPercent) / 100

			// Use deployment-specific isolation segment
			cellSegment := isolationSegment
			if vm.JobName == "isolated_diego_cell" {
				cellSegment = "isolated" // isolated_diego_cell is always in an isolation segment
			}

			cells = append(cells, models.DiegoCell{
				ID:               vm.ID,
				Name:             fmt.Sprintf("%s/%d", vm.JobName, vm.Index),
				MemoryMB:         memoryMB,
				AllocatedMB:      usedMB,
				UsedMB:           usedMB,
				CPUPercent:       int(cpuSys),
				IsolationSegment: cellSegment,
			})
		}
	}

	return cells, nil
}

// waitForTaskAndGetOutput polls a BOSH task until done and returns VM data
func (b *BOSHClient) waitForTaskAndGetOutput(taskID int) ([]boshVM, error) {
	taskURL := fmt.Sprintf("%s/tasks/%d", b.environment, taskID)

	for i := 0; i < 60; i++ { // Max 60 attempts (2 minutes with 2s sleep)
		req, err := http.NewRequest("GET", taskURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create task request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+b.token)

		resp, err := b.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get task status: %w", err)
		}

		var task boshTask
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to parse task status: %w", err)
		}
		resp.Body.Close()

		switch task.State {
		case "done":
			// Get task output
			return b.getTaskOutput(taskID)
		case "error", "cancelled":
			return nil, fmt.Errorf("BOSH task failed: %s", task.Result)
		case "processing", "queued":
			time.Sleep(2 * time.Second)
		default:
			time.Sleep(2 * time.Second)
		}
	}

	return nil, fmt.Errorf("timeout waiting for BOSH task %d", taskID)
}

// getTaskOutput retrieves the output from a completed task
func (b *BOSHClient) getTaskOutput(taskID int) ([]boshVM, error) {
	outputURL := fmt.Sprintf("%s/tasks/%d/output?type=result", b.environment, taskID)

	req, err := http.NewRequest("GET", outputURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create output request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.token)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task output: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get task output (status %d): %s", resp.StatusCode, string(body))
	}

	// Task output is NDJSON (newline-delimited JSON)
	var vms []boshVM
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read task output: %w", err)
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var vm boshVM
		if err := json.Unmarshal([]byte(line), &vm); err != nil {
			slog.Warn("Failed to parse VM line", "line", line, "error", err)
			continue
		}
		vms = append(vms, vm)
	}

	return vms, nil
}

func parseIntOrZero(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func parseFloatOrZero(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
