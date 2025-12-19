// ABOUTME: BOSH API client for Diego cell VM metrics
// ABOUTME: Queries BOSH Director for deployment VMs with vitals

package services

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"net/http"
	"time"
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
		certPool.AppendCertsFromPEM([]byte(caCert))
		tlsConfig.RootCAs = certPool
	} else {
		tlsConfig.InsecureSkipVerify = true
	}

	return &BOSHClient{
		environment: environment,
		clientID:    clientID,
		secret:      secret,
		caCert:      caCert,
		deployment:  deployment,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}
}

func (b *BOSHClient) GetDiegoCells() ([]models.DiegoCell, error) {
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
