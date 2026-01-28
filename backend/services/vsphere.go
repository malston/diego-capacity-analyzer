// ABOUTME: vSphere client for infrastructure discovery via govmomi
// ABOUTME: Retrieves cluster, host, and VM inventory for capacity analysis

package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// VSphereCredentials holds vCenter connection info
type VSphereCredentials struct {
	Host       string
	Username   string
	Password   string
	Datacenter string
	Insecure   bool
}

// VSphereClient wraps govmomi client for infrastructure discovery
type VSphereClient struct {
	creds      VSphereCredentials
	client     *govmomi.Client
	finder     *find.Finder
	datacenter *object.Datacenter
}

// NewVSphereClient creates a new vSphere client
func NewVSphereClient(creds VSphereCredentials) *VSphereClient {
	return &VSphereClient{
		creds: creds,
	}
}

// Connect establishes connection to vCenter
func (v *VSphereClient) Connect(ctx context.Context) error {
	host := v.creds.Host
	if !strings.HasPrefix(host, "https://") && !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}

	u, err := url.Parse(host + "/sdk")
	if err != nil {
		return fmt.Errorf("invalid vCenter URL '%s': %w", v.creds.Host, err)
	}
	u.User = url.UserPassword(v.creds.Username, v.creds.Password)

	client, err := govmomi.NewClient(ctx, u, v.creds.Insecure)
	if err != nil {
		// Provide more specific error messages
		errStr := err.Error()
		if strings.Contains(errStr, "connection refused") {
			return fmt.Errorf("connection refused to vCenter at %s - verify the host is reachable", v.creds.Host)
		}
		if strings.Contains(errStr, "no such host") {
			return fmt.Errorf("cannot resolve vCenter hostname '%s' - verify DNS", v.creds.Host)
		}
		if strings.Contains(errStr, "401") || strings.Contains(errStr, "Cannot complete login") {
			return fmt.Errorf("authentication failed - verify username and password")
		}
		if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout") {
			return fmt.Errorf("connection timeout to vCenter at %s - check network connectivity", v.creds.Host)
		}
		if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "x509") {
			return fmt.Errorf("SSL certificate error connecting to %s - try setting VSPHERE_INSECURE=true", v.creds.Host)
		}
		return fmt.Errorf("failed to connect to vCenter at %s: %w", v.creds.Host, err)
	}

	v.client = client
	v.finder = find.NewFinder(client.Client, true)

	// Set datacenter
	dc, err := v.finder.Datacenter(ctx, v.creds.Datacenter)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("datacenter '%s' not found - verify the datacenter name", v.creds.Datacenter)
		}
		return fmt.Errorf("error accessing datacenter '%s': %w", v.creds.Datacenter, err)
	}
	v.datacenter = dc
	v.finder.SetDatacenter(dc)

	slog.Info("vSphere connected successfully")
	slog.Debug("vSphere connection details", "host", v.creds.Host, "datacenter", v.creds.Datacenter)
	return nil
}

// Disconnect closes the vCenter connection
func (v *VSphereClient) Disconnect(ctx context.Context) error {
	if v.client != nil {
		return v.client.Logout(ctx)
	}
	return nil
}

// ClusterInfo holds cluster inventory data
type ClusterInfo struct {
	Name           string
	Hosts          []HostInfo
	TotalMemoryMB  int64
	TotalCPUCores  int32
	DiegoCellCount int
	DiegoCells     []VMInfo
}

// HostInfo holds ESXi host data
type HostInfo struct {
	Name        string
	MemoryMB    int64
	CPUCores    int32
	InCluster   string
	PowerState  string
	Maintenance bool
}

// VMInfo holds virtual machine data
type VMInfo struct {
	Name         string
	MemoryMB     int32
	NumCPU       int32
	PowerState   string
	Host         string
	Cluster      string
	IsDiegoCell  bool
	CellMemoryGB int
	CellCPU      int
}

// GetClusters retrieves all compute clusters in the datacenter
func (v *VSphereClient) GetClusters(ctx context.Context) ([]ClusterInfo, error) {
	clusters, err := v.finder.ClusterComputeResourceList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("listing clusters: %w", err)
	}

	result := make([]ClusterInfo, 0, len(clusters))

	for _, cluster := range clusters {
		info, err := v.getClusterInfo(ctx, cluster)
		if err != nil {
			return nil, fmt.Errorf("getting cluster %s info: %w", cluster.Name(), err)
		}
		result = append(result, info)
	}

	return result, nil
}

// getClusterInfo retrieves detailed info for a single cluster
func (v *VSphereClient) getClusterInfo(ctx context.Context, cluster *object.ClusterComputeResource) (ClusterInfo, error) {
	info := ClusterInfo{
		Name: cluster.Name(),
	}

	// Get cluster properties
	var clusterMo mo.ClusterComputeResource
	err := cluster.Properties(ctx, cluster.Reference(), []string{"host"}, &clusterMo)
	if err != nil {
		return info, fmt.Errorf("getting cluster properties: %w", err)
	}

	// Get hosts
	for _, hostRef := range clusterMo.Host {
		host := object.NewHostSystem(v.client.Client, hostRef)
		hostInfo, err := v.getHostInfo(ctx, host, cluster.Name())
		if err != nil {
			return info, fmt.Errorf("getting host info: %w", err)
		}
		info.Hosts = append(info.Hosts, hostInfo)
		info.TotalMemoryMB += hostInfo.MemoryMB
		info.TotalCPUCores += hostInfo.CPUCores
	}

	// Get Diego cells in this cluster
	cells, err := v.getDiegoCellsInCluster(ctx, cluster)
	if err != nil {
		return info, fmt.Errorf("getting Diego cells: %w", err)
	}
	info.DiegoCells = cells
	info.DiegoCellCount = len(cells)

	return info, nil
}

// getHostInfo retrieves host hardware summary
func (v *VSphereClient) getHostInfo(ctx context.Context, host *object.HostSystem, clusterName string) (HostInfo, error) {
	var hostMo mo.HostSystem
	err := host.Properties(ctx, host.Reference(), []string{"summary", "runtime"}, &hostMo)
	if err != nil {
		return HostInfo{}, fmt.Errorf("getting host properties: %w", err)
	}

	info := HostInfo{
		Name:        host.Name(),
		MemoryMB:    hostMo.Summary.Hardware.MemorySize / (1024 * 1024),
		CPUCores:    int32(hostMo.Summary.Hardware.NumCpuThreads), // Logical processors (includes hyperthreading)
		InCluster:   clusterName,
		PowerState:  string(hostMo.Runtime.PowerState),
		Maintenance: hostMo.Runtime.InMaintenanceMode,
	}

	return info, nil
}

// getDiegoCellsInCluster finds Diego cell VMs in a cluster
func (v *VSphereClient) getDiegoCellsInCluster(ctx context.Context, cluster *object.ClusterComputeResource) ([]VMInfo, error) {
	// List all VMs in the datacenter
	vms, err := v.finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("listing VMs: %w", err)
	}

	var cells []VMInfo
	for _, vm := range vms {
		vmInfo, err := v.getVMInfo(ctx, vm)
		if err != nil {
			continue // Skip VMs we can't read
		}

		// Filter to Diego cells in this cluster
		if vmInfo.Cluster == cluster.Name() && vmInfo.IsDiegoCell {
			cells = append(cells, vmInfo)
		}
	}
	return cells, nil
}

// getVMInfo retrieves VM configuration
func (v *VSphereClient) getVMInfo(ctx context.Context, vm *object.VirtualMachine) (VMInfo, error) {
	var vmMo mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"config", "runtime", "summary", "customValue"}, &vmMo)
	if err != nil {
		return VMInfo{}, err
	}

	info := VMInfo{
		Name:       vm.Name(),
		PowerState: string(vmMo.Runtime.PowerState),
	}

	if vmMo.Config != nil {
		info.MemoryMB = vmMo.Config.Hardware.MemoryMB
		info.NumCPU = vmMo.Config.Hardware.NumCPU
	}

	// Check custom attributes for BOSH job name
	// BOSH sets custom attributes like "job", "id", "deployment"
	for _, cv := range vmMo.CustomValue {
		if field, ok := cv.(*types.CustomFieldStringValue); ok {
			// Check if value looks like a diego cell job name
			val := strings.ToLower(field.Value)
			if strings.Contains(val, "diego_cell") || strings.Contains(val, "diego-cell") ||
				strings.HasPrefix(val, "compute") || strings.HasPrefix(val, "diego") ||
				strings.Contains(val, "isolated_diego_cell") {
				info.IsDiegoCell = true
				break
			}
		}
	}

	// Fallback to name-based detection if no custom attributes matched
	if !info.IsDiegoCell {
		name := strings.ToLower(vm.Name())
		info.IsDiegoCell = strings.Contains(name, "diego_cell") ||
			strings.Contains(name, "diego-cell") ||
			strings.HasPrefix(name, "compute") ||
			strings.HasPrefix(name, "diego")
	}

	if info.IsDiegoCell {
		info.CellMemoryGB = int(info.MemoryMB / 1024)
		info.CellCPU = int(info.NumCPU)
	}

	// Get host and cluster
	if vmMo.Runtime.Host != nil {
		host := object.NewHostSystem(v.client.Client, *vmMo.Runtime.Host)
		info.Host = host.Name()

		// Find cluster for this host
		var hostMo mo.HostSystem
		if err := host.Properties(ctx, host.Reference(), []string{"parent"}, &hostMo); err == nil {
			if hostMo.Parent != nil && hostMo.Parent.Type == "ClusterComputeResource" {
				cluster := object.NewClusterComputeResource(v.client.Client, *hostMo.Parent)
				info.Cluster = cluster.Name()
			}
		}
	}

	return info, nil
}

// GetInfrastructureState builds InfrastructureState from vSphere data
// Uses the same calculation logic as ManualInput.ToInfrastructureState() for consistency
func (v *VSphereClient) GetInfrastructureState(ctx context.Context) (models.InfrastructureState, error) {
	// Get all clusters for host/memory info
	clusters, err := v.GetClusters(ctx)
	if err != nil {
		return models.InfrastructureState{}, fmt.Errorf("getting clusters: %w", err)
	}

	// Find all Diego cells across entire datacenter
	allCells, err := v.getAllDiegoCells(ctx)
	if err != nil {
		return models.InfrastructureState{}, fmt.Errorf("getting Diego cells: %w", err)
	}

	slog.Info("vSphere Diego cell discovery complete", "cell_count", len(allCells))

	// Build ManualInput from vSphere data to leverage existing calculation logic
	manualInput := models.ManualInput{
		Name:     v.creds.Datacenter,
		Clusters: make([]models.ClusterInput, 0, len(clusters)),
	}

	// Aggregate all hosts into a single logical cluster if cells span multiple vSphere clusters
	// First, collect all host stats
	var totalHosts int
	var totalMemoryMB int64
	var totalCPUCores int32
	var avgMemoryPerHost int
	var avgCPUPerHost int

	for _, c := range clusters {
		for _, h := range c.Hosts {
			if h.PowerState == "poweredOn" && !h.Maintenance {
				totalHosts++
				totalMemoryMB += h.MemoryMB
				totalCPUCores += h.CPUCores
			}
		}
	}

	if totalHosts > 0 {
		avgMemoryPerHost = int(totalMemoryMB / int64(totalHosts) / 1024) // Convert to GB
		avgCPUPerHost = int(totalCPUCores) / totalHosts
	}

	// Group Diego cells by cluster for proper per-cluster analysis
	cellsByCluster := make(map[string][]VMInfo)
	for _, cell := range allCells {
		clusterName := cell.Cluster
		if clusterName == "" {
			clusterName = "default"
		}
		cellsByCluster[clusterName] = append(cellsByCluster[clusterName], cell)
	}

	// Create cluster inputs for each vSphere cluster with Diego cells
	for _, c := range clusters {
		cells := cellsByCluster[c.Name]
		if len(cells) == 0 {
			continue // Skip clusters without Diego cells
		}

		// Calculate per-host metrics for this cluster
		var clusterHosts int
		var clusterMemoryMB int64
		var clusterCPUCores int32

		for _, h := range c.Hosts {
			if h.PowerState == "poweredOn" && !h.Maintenance {
				clusterHosts++
				clusterMemoryMB += h.MemoryMB
				clusterCPUCores += h.CPUCores
			}
		}

		if clusterHosts == 0 {
			continue
		}

		memoryPerHost := int(clusterMemoryMB / int64(clusterHosts) / 1024) // GB
		cpuPerHost := int(clusterCPUCores) / clusterHosts

		// Use first cell's size (assuming uniform within cluster)
		cellMemoryGB := cells[0].CellMemoryGB
		cellCPU := cells[0].CellCPU
		if cellMemoryGB == 0 {
			cellMemoryGB = int(cells[0].MemoryMB / 1024)
		}
		if cellCPU == 0 {
			cellCPU = int(cells[0].NumCPU)
		}

		clusterInput := models.ClusterInput{
			Name:              c.Name,
			HostCount:         clusterHosts,
			MemoryGBPerHost:   memoryPerHost,
			CPUCoresPerHost:   cpuPerHost,
			DiegoCellCount:    len(cells),
			DiegoCellMemoryGB: cellMemoryGB,
			DiegoCellCPU:      cellCPU,
		}

		manualInput.Clusters = append(manualInput.Clusters, clusterInput)
	}

	// Handle cells without a cluster assignment
	defaultCells := cellsByCluster["default"]
	if len(defaultCells) > 0 && avgMemoryPerHost > 0 {
		cellMemoryGB := defaultCells[0].CellMemoryGB
		cellCPU := defaultCells[0].CellCPU
		if cellMemoryGB == 0 {
			cellMemoryGB = int(defaultCells[0].MemoryMB / 1024)
		}
		if cellCPU == 0 {
			cellCPU = int(defaultCells[0].NumCPU)
		}

		clusterInput := models.ClusterInput{
			Name:              "unassigned",
			HostCount:         totalHosts,
			MemoryGBPerHost:   avgMemoryPerHost,
			CPUCoresPerHost:   avgCPUPerHost,
			DiegoCellCount:    len(defaultCells),
			DiegoCellMemoryGB: cellMemoryGB,
			DiegoCellCPU:      cellCPU,
		}
		manualInput.Clusters = append(manualInput.Clusters, clusterInput)
	}

	// Use the standard ToInfrastructureState() for consistent calculations
	state := manualInput.ToInfrastructureState()
	state.Source = "vsphere" // Override source

	return state, nil
}

// getAllDiegoCells finds all Diego cell VMs in the datacenter
func (v *VSphereClient) getAllDiegoCells(ctx context.Context) ([]VMInfo, error) {
	vms, err := v.finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("listing VMs: %w", err)
	}

	var cells []VMInfo
	for _, vm := range vms {
		vmInfo, err := v.getVMInfo(ctx, vm)
		if err != nil {
			continue
		}
		if vmInfo.IsDiegoCell {
			cells = append(cells, vmInfo)
		}
	}

	return cells, nil
}

// ParseOpsManagerCredentials extracts vCenter credentials from om staged-director-config output
// The input should be the iaas-configurations section from: om staged-director-config --no-redact
func ParseOpsManagerCredentials(iaasConfig map[string]interface{}) (VSphereCredentials, error) {
	creds := VSphereCredentials{}

	if host, ok := iaasConfig["vcenter_host"].(string); ok {
		creds.Host = host
	} else {
		return creds, fmt.Errorf("missing vcenter_host")
	}

	if user, ok := iaasConfig["vcenter_username"].(string); ok {
		creds.Username = user
	} else {
		return creds, fmt.Errorf("missing vcenter_username")
	}

	if pass, ok := iaasConfig["vcenter_password"].(string); ok {
		creds.Password = pass
	} else {
		return creds, fmt.Errorf("missing vcenter_password")
	}

	if dc, ok := iaasConfig["datacenter"].(string); ok {
		creds.Datacenter = dc
	} else {
		return creds, fmt.Errorf("missing datacenter")
	}

	// Default to insecure for lab environments
	creds.Insecure = true

	return creds, nil
}

// VSphereClientFromEnv creates a client from environment variables
func VSphereClientFromEnv(host, user, pass, datacenter string) *VSphereClient {
	return NewVSphereClient(VSphereCredentials{
		Host:       host,
		Username:   user,
		Password:   pass,
		Datacenter: datacenter,
		Insecure:   true,
	})
}

// IsConnected returns true if client has an active connection
func (v *VSphereClient) IsConnected() bool {
	return v.client != nil && v.client.Valid()
}

// GetClusterNames returns just the cluster names (useful for dropdowns)
func (v *VSphereClient) GetClusterNames(ctx context.Context) ([]string, error) {
	clusters, err := v.finder.ClusterComputeResourceList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("listing clusters: %w", err)
	}

	names := make([]string, len(clusters))
	for i, c := range clusters {
		names[i] = c.Name()
	}
	return names, nil
}

// FilterVMsByPattern finds VMs matching a name pattern
func (v *VSphereClient) FilterVMsByPattern(ctx context.Context, pattern string) ([]VMInfo, error) {
	vms, err := v.finder.VirtualMachineList(ctx, pattern)
	if err != nil {
		// No VMs found is not an error
		if _, ok := err.(*find.NotFoundError); ok {
			return nil, nil
		}
		return nil, err
	}

	result := make([]VMInfo, 0, len(vms))
	for _, vm := range vms {
		info, err := v.getVMInfo(ctx, vm)
		if err != nil {
			continue
		}
		result = append(result, info)
	}

	return result, nil
}

// Ensure types is used (govmomi requires it for ManagedObjectReference handling)
var _ = types.ManagedObjectReference{}
