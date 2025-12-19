// ABOUTME: BOSH API client for Diego cell VM metrics
// ABOUTME: Queries BOSH Director for deployment VMs with vitals

package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
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
}

func NewBOSHClient(environment, clientID, secret, caCert, deployment string) *BOSHClient {
	tlsConfig := &tls.Config{}

	if caCert != "" {
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM([]byte(caCert)); ok {
			tlsConfig.RootCAs = certPool
		} else {
			log.Printf("Warning: Failed to parse BOSH_CA_CERT, using InsecureSkipVerify")
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

// createSOCKS5DialContextFunc creates a dial function for SSH+SOCKS5 proxy connections.
// Supports format: ssh+socks5://user@host:port?private-key=/path/to/key
func createSOCKS5DialContextFunc(allProxy string) func(ctx context.Context, network, address string) (net.Conn, error) {
	// Strip ssh+ prefix if present
	allProxy = strings.TrimPrefix(allProxy, "ssh+")

	proxyURL, err := url.Parse(allProxy)
	if err != nil {
		log.Printf("Failed to parse BOSH_ALL_PROXY URL: %v", err)
		return nil
	}

	queryMap, err := url.ParseQuery(proxyURL.RawQuery)
	if err != nil {
		log.Printf("Failed to parse BOSH_ALL_PROXY query params: %v", err)
		return nil
	}

	username := ""
	if proxyURL.User != nil {
		username = proxyURL.User.Username()
	}

	proxySSHKeyPath := queryMap.Get("private-key")
	if proxySSHKeyPath == "" {
		log.Printf("BOSH_ALL_PROXY missing required 'private-key' query param")
		return nil
	}

	proxySSHKey, err := os.ReadFile(proxySSHKeyPath)
	if err != nil {
		log.Printf("Failed to read SSH private key: %v", err)
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

func (b *BOSHClient) GetDiegoCells() ([]models.DiegoCell, error) {
	if b.deployment == "" {
		return nil, fmt.Errorf("BOSH_DEPLOYMENT is not configured")
	}
	url := fmt.Sprintf("%s/deployments/%s/vms?format=full", b.environment, b.deployment)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(b.clientID, b.secret)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VMs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BOSH API returned status %d", resp.StatusCode)
	}

	var vms []struct {
		JobName string `json:"job_name"`
		Index   int    `json:"index"`
		ID      string `json:"id"`
		Vitals  struct {
			Mem struct {
				KB      int `json:"kb"`
				Percent int `json:"percent"`
			} `json:"mem"`
			CPU struct {
				Sys int `json:"sys"`
			} `json:"cpu"`
			Disk struct {
				System struct {
					Percent int `json:"percent"`
				} `json:"system"`
			} `json:"disk"`
		} `json:"vitals"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vms); err != nil {
		return nil, fmt.Errorf("failed to parse VMs: %w", err)
	}

	var cells []models.DiegoCell
	for _, vm := range vms {
		if vm.JobName == "diego_cell" || vm.JobName == "compute" {
			memoryMB := vm.Vitals.Mem.KB / 1024
			cells = append(cells, models.DiegoCell{
				ID:               vm.ID,
				Name:             fmt.Sprintf("%s/%d", vm.JobName, vm.Index),
				MemoryMB:         memoryMB,
				AllocatedMB:      (memoryMB * vm.Vitals.Mem.Percent) / 100,
				UsedMB:           0, // Will be calculated from apps
				CPUPercent:       vm.Vitals.CPU.Sys,
				IsolationSegment: "default", // Will be refined later
			})
		}
	}

	return cells, nil
}
