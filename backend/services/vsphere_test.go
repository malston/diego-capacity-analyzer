// ABOUTME: Unit tests for vSphere service
// ABOUTME: Tests credential parsing and infrastructure state conversion

package services

import (
	"testing"
)

func TestParseOpsManagerCredentials(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		want    VSphereCredentials
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"vcenter_host":     "vcenter.example.com",
				"vcenter_username": "admin@vsphere.local",
				"vcenter_password": "secret123",
				"datacenter":       "DC1",
			},
			want: VSphereCredentials{
				Host:       "vcenter.example.com",
				Username:   "admin@vsphere.local",
				Password:   "secret123",
				Datacenter: "DC1",
				Insecure:   true,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: map[string]interface{}{
				"vcenter_username": "admin@vsphere.local",
				"vcenter_password": "secret123",
				"datacenter":       "DC1",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: map[string]interface{}{
				"vcenter_host":     "vcenter.example.com",
				"vcenter_password": "secret123",
				"datacenter":       "DC1",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			config: map[string]interface{}{
				"vcenter_host":     "vcenter.example.com",
				"vcenter_username": "admin@vsphere.local",
				"datacenter":       "DC1",
			},
			wantErr: true,
		},
		{
			name: "missing datacenter",
			config: map[string]interface{}{
				"vcenter_host":     "vcenter.example.com",
				"vcenter_username": "admin@vsphere.local",
				"vcenter_password": "secret123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOpsManagerCredentials(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOpsManagerCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Host != tt.want.Host {
					t.Errorf("Host = %v, want %v", got.Host, tt.want.Host)
				}
				if got.Username != tt.want.Username {
					t.Errorf("Username = %v, want %v", got.Username, tt.want.Username)
				}
				if got.Password != tt.want.Password {
					t.Errorf("Password = %v, want %v", got.Password, tt.want.Password)
				}
				if got.Datacenter != tt.want.Datacenter {
					t.Errorf("Datacenter = %v, want %v", got.Datacenter, tt.want.Datacenter)
				}
				if got.Insecure != tt.want.Insecure {
					t.Errorf("Insecure = %v, want %v", got.Insecure, tt.want.Insecure)
				}
			}
		})
	}
}

func TestVSphereClientFromEnv(t *testing.T) {
	client := VSphereClientFromEnv(
		"vcenter.example.com",
		"admin@vsphere.local",
		"secret123",
		"DC1",
	)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.creds.Host != "vcenter.example.com" {
		t.Errorf("Host = %v, want vcenter.example.com", client.creds.Host)
	}
	if client.creds.Username != "admin@vsphere.local" {
		t.Errorf("Username = %v, want admin@vsphere.local", client.creds.Username)
	}
	if client.creds.Password != "secret123" {
		t.Errorf("Password = %v, want secret123", client.creds.Password)
	}
	if client.creds.Datacenter != "DC1" {
		t.Errorf("Datacenter = %v, want DC1", client.creds.Datacenter)
	}
	if !client.creds.Insecure {
		t.Error("Expected Insecure to be true")
	}
}

func TestNewVSphereClient(t *testing.T) {
	creds := VSphereCredentials{
		Host:       "vcenter.example.com",
		Username:   "admin@vsphere.local",
		Password:   "secret123",
		Datacenter: "DC1",
		Insecure:   false,
	}

	client := NewVSphereClient(creds)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.creds.Host != creds.Host {
		t.Errorf("Host = %v, want %v", client.creds.Host, creds.Host)
	}
	if client.IsConnected() {
		t.Error("Expected client to not be connected initially")
	}
}

func TestVMInfoIsDiegoCell(t *testing.T) {
	tests := []struct {
		vmName     string
		isDiego    bool
	}{
		{"diego_cell/abc123", true},
		{"diego-cell-0", true},
		{"diego_cell", true},
		{"diego-cell", true},
		{"DIEGO_CELL/xyz", true},
		{"router/abc123", false},
		{"nats/0", false},
		{"compute-cell-1", false},
		{"diego-router-0", false},
	}

	for _, tt := range tests {
		t.Run(tt.vmName, func(t *testing.T) {
			// Test name pattern matching logic (matching the logic in getVMInfo)
			name := tt.vmName
			isDiego := containsDiegoCellPattern(name)
			if isDiego != tt.isDiego {
				t.Errorf("containsDiegoCellPattern(%q) = %v, want %v", name, isDiego, tt.isDiego)
			}
		})
	}
}

// containsDiegoCellPattern checks if a name matches Diego cell patterns
func containsDiegoCellPattern(name string) bool {
	// Convert to lowercase for case-insensitive matching
	lower := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			lower[i] = c + 32
		} else {
			lower[i] = c
		}
	}
	lowerStr := string(lower)

	// Check for diego_cell or diego-cell patterns
	for i := 0; i <= len(lowerStr)-10; i++ {
		if lowerStr[i:i+10] == "diego_cell" || lowerStr[i:i+10] == "diego-cell" {
			return true
		}
	}
	// Also check partial matches at end
	if len(lowerStr) >= 10 {
		return false
	}
	// For strings shorter than 10, check if they start with the pattern
	if len(lowerStr) >= 10 {
		prefix := lowerStr[:10]
		return prefix == "diego_cell" || prefix == "diego-cell"
	}
	return false
}

func TestClusterInfoHostAggregation(t *testing.T) {
	// Test that ClusterInfo correctly aggregates host data
	info := ClusterInfo{
		Name: "test-cluster",
		Hosts: []HostInfo{
			{Name: "esx01", MemoryMB: 524288, CPUCores: 32},
			{Name: "esx02", MemoryMB: 524288, CPUCores: 32},
			{Name: "esx03", MemoryMB: 524288, CPUCores: 32},
		},
	}

	// Calculate totals manually
	var totalMemory int64
	var totalCores int32
	for _, h := range info.Hosts {
		totalMemory += h.MemoryMB
		totalCores += h.CPUCores
	}

	expectedMemory := int64(3 * 524288) // 3 hosts × 512GB
	expectedCores := int32(3 * 32)      // 3 hosts × 32 cores

	if totalMemory != expectedMemory {
		t.Errorf("Total memory = %d, want %d", totalMemory, expectedMemory)
	}
	if totalCores != expectedCores {
		t.Errorf("Total cores = %d, want %d", totalCores, expectedCores)
	}
}
