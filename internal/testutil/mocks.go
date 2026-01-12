// Package testutil provides shared test utilities and mocks
package testutil

import (
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/stretchr/testify/mock"
)

// MockMultipassClient is a testify mock for multipass.Client
type MockMultipassClient struct {
	mock.Mock
}

// Ensure MockMultipassClient implements multipass.Client
var _ multipass.Client = (*MockMultipassClient)(nil)

// List mocks the List method
func (m *MockMultipassClient) List() ([]multipass.ListInstance, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]multipass.ListInstance), args.Error(1)
}

// Info mocks the Info method
func (m *MockMultipassClient) Info(name string) (*multipass.InstanceInfo, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*multipass.InstanceInfo), args.Error(1)
}

// Launch mocks the Launch method
func (m *MockMultipassClient) Launch(opts multipass.LaunchOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}

// Start mocks the Start method
func (m *MockMultipassClient) Start(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// Stop mocks the Stop method
func (m *MockMultipassClient) Stop(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// Restart mocks the Restart method
func (m *MockMultipassClient) Restart(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockMultipassClient) Delete(name string, purge bool) error {
	args := m.Called(name, purge)
	return args.Error(0)
}

// Clone mocks the Clone method
func (m *MockMultipassClient) Clone(source, dest string) error {
	args := m.Called(source, dest)
	return args.Error(0)
}

// ListSnapshots mocks the ListSnapshots method
func (m *MockMultipassClient) ListSnapshots(vmName string) (map[string]multipass.Snapshot, error) {
	args := m.Called(vmName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]multipass.Snapshot), args.Error(1)
}

// CreateSnapshot mocks the CreateSnapshot method
func (m *MockMultipassClient) CreateSnapshot(vmName, snapshotName string) error {
	args := m.Called(vmName, snapshotName)
	return args.Error(0)
}

// RestoreSnapshot mocks the RestoreSnapshot method
func (m *MockMultipassClient) RestoreSnapshot(vmName, snapshotName string, destructive bool) error {
	args := m.Called(vmName, snapshotName, destructive)
	return args.Error(0)
}

// DeleteSnapshot mocks the DeleteSnapshot method
func (m *MockMultipassClient) DeleteSnapshot(vmName, snapshotName string) error {
	args := m.Called(vmName, snapshotName)
	return args.Error(0)
}

// Transfer mocks the Transfer method
func (m *MockMultipassClient) Transfer(src, dst string) error {
	args := m.Called(src, dst)
	return args.Error(0)
}

// Exec mocks the Exec method
func (m *MockMultipassClient) Exec(vmName string, cmd ...string) (string, error) {
	args := m.Called(vmName, cmd)
	return args.String(0), args.Error(1)
}

// Mount mocks the Mount method
func (m *MockMultipassClient) Mount(vmName, hostPath, vmPath string) error {
	args := m.Called(vmName, hostPath, vmPath)
	return args.Error(0)
}

// Unmount mocks the Unmount method
func (m *MockMultipassClient) Unmount(vmName, path string) error {
	args := m.Called(vmName, path)
	return args.Error(0)
}

// Helper functions for creating test fixtures

// RunningVM creates a mock InstanceInfo for a running VM
func RunningVM(name string, ip string) *multipass.InstanceInfo {
	return &multipass.InstanceInfo{
		State:         multipass.StateRunning,
		IPv4:          []string{ip},
		CPUCount:      "2",
		Release:       "Ubuntu 24.04 LTS",
		ImageRelease:  "24.04 LTS",
		ImageHash:     "abc123def456",
		SnapshotCount: "0",
		Load:          []float64{0.1, 0.15, 0.1},
		Memory: multipass.Memory{
			Total: 4294967296, // 4GB
			Used:  1073741824, // 1GB
		},
		Disks: map[string]multipass.Disk{
			"sda1": {
				Total: "21474836480", // 20GB
				Used:  "5368709120",  // 5GB
			},
		},
		Mounts: map[string]multipass.Mount{},
	}
}

// StoppedVM creates a mock InstanceInfo for a stopped VM
func StoppedVM(name string) *multipass.InstanceInfo {
	return &multipass.InstanceInfo{
		State:         multipass.StateStopped,
		IPv4:          []string{},
		CPUCount:      "2",
		Release:       "Ubuntu 24.04 LTS",
		ImageRelease:  "24.04 LTS",
		ImageHash:     "abc123def456",
		SnapshotCount: "0",
		Load:          []float64{},
		Memory: multipass.Memory{
			Total: 4294967296,
			Used:  0,
		},
		Disks: map[string]multipass.Disk{
			"sda1": {
				Total: "21474836480",
				Used:  "5368709120",
			},
		},
		Mounts: map[string]multipass.Mount{},
	}
}

// RunningVMList creates a mock list of running VMs
func RunningVMList(names ...string) []multipass.ListInstance {
	vms := make([]multipass.ListInstance, len(names))
	for i, name := range names {
		vms[i] = multipass.ListInstance{
			Name:    name,
			State:   multipass.StateRunning,
			IPv4:    []string{"192.168.64." + string(rune('1'+i))},
			Release: "Ubuntu 24.04 LTS",
		}
	}
	return vms
}

// MixedVMList creates a mock list with mixed VM states
func MixedVMList() []multipass.ListInstance {
	return []multipass.ListInstance{
		{
			Name:    "running-vm",
			State:   multipass.StateRunning,
			IPv4:    []string{"192.168.64.1"},
			Release: "Ubuntu 24.04 LTS",
		},
		{
			Name:    "stopped-vm",
			State:   multipass.StateStopped,
			IPv4:    []string{},
			Release: "Ubuntu 22.04 LTS",
		},
		{
			Name:    "suspended-vm",
			State:   multipass.StateSuspended,
			IPv4:    []string{},
			Release: "Ubuntu 24.04 LTS",
		},
	}
}

// TestSnapshots creates a mock snapshot map
func TestSnapshots() map[string]multipass.Snapshot {
	return map[string]multipass.Snapshot{
		"snap1": {Comment: "Before update", Parent: ""},
		"snap2": {Comment: "After update", Parent: "snap1"},
	}
}
