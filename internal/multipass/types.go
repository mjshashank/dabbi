package multipass

// ListResponse represents the JSON output of `multipass list --format json`
type ListResponse struct {
	List []ListInstance `json:"list"`
}

// ListInstance represents a VM in the list output
type ListInstance struct {
	Name    string   `json:"name"`
	State   string   `json:"state"` // "Running", "Stopped", "Suspended", "Deleted"
	IPv4    []string `json:"ipv4"`
	Release string   `json:"release"` // e.g., "Ubuntu 24.04 LTS"
}

// InfoResponse represents the JSON output of `multipass info <vm> --format json`
type InfoResponse struct {
	Errors []string                `json:"errors"`
	Info   map[string]InstanceInfo `json:"info"`
}

// InstanceInfo represents detailed information about a VM
type InstanceInfo struct {
	CPUCount      string           `json:"cpu_count"` // NOTE: string, not int
	Disks         map[string]Disk  `json:"disks"`     // key is device name (e.g., "sda1")
	ImageHash     string           `json:"image_hash"`
	ImageRelease  string           `json:"image_release"`
	IPv4          []string         `json:"ipv4"`
	Load          []float64        `json:"load"` // 1, 5, 15 min load averages
	Memory        Memory           `json:"memory"`
	Mounts        map[string]Mount `json:"mounts"`         // key is target path
	Release       string           `json:"release"`        // e.g., "Ubuntu 24.04.3 LTS"
	SnapshotCount string           `json:"snapshot_count"` // NOTE: string, not int
	State         string           `json:"state"`
}

// Disk represents disk usage information
type Disk struct {
	Total string `json:"total"` // NOTE: string (bytes as string)
	Used  string `json:"used"`
}

// Memory represents memory usage information
type Memory struct {
	Total int64 `json:"total"` // bytes
	Used  int64 `json:"used"`
}

// Mount represents a mount point
type Mount struct {
	SourcePath string `json:"source_path"`
}

// SnapshotsResponse represents the JSON output of `multipass list --snapshots --format json`
type SnapshotsResponse struct {
	Errors []string                       `json:"errors"`
	Info   map[string]map[string]Snapshot `json:"info"` // vm_name -> snapshot_name -> snapshot
}

// Snapshot represents a VM snapshot
type Snapshot struct {
	Comment string `json:"comment"`
	Parent  string `json:"parent"` // parent snapshot name, empty if base
}

// LaunchOptions holds options for creating a new VM
type LaunchOptions struct {
	Name          string
	CPUs          int
	Memory        string         // e.g., "4G"
	Disk          string         // e.g., "20G"
	CloudInit     string         // path to cloud-init file
	Image         string         // e.g., "22.04" or "jammy"
	NetworkConfig *NetworkConfig // network restrictions (nil = no restrictions)
}

// NetworkMode defines the type of network restriction for a VM
type NetworkMode string

const (
	NetworkModeNone      NetworkMode = "none"      // No restrictions (default)
	NetworkModeAllowlist NetworkMode = "allowlist" // Only allow specified hosts
	NetworkModeBlocklist NetworkMode = "blocklist" // Block specified hosts
	NetworkModeIsolated  NetworkMode = "isolated"  // No network access at all
)

// NetworkRule represents a single network rule (host to allow/block)
type NetworkRule struct {
	Type    string `json:"type"`              // "ip", "cidr", "domain"
	Value   string `json:"value"`             // e.g., "192.168.1.1", "10.0.0.0/8", "github.com"
	Comment string `json:"comment,omitempty"` // optional description
}

// NetworkConfig holds network restriction configuration for a VM
type NetworkConfig struct {
	Mode  NetworkMode   `json:"mode"`
	Rules []NetworkRule `json:"rules,omitempty"`
}

// VM States
const (
	StateRunning   = "Running"
	StateStopped   = "Stopped"
	StateSuspended = "Suspended"
	StateDeleted   = "Deleted"
)
