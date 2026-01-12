package tunnel

import (
	"errors"
	"testing"

	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	m := NewManager(mockMP)

	require.NotNil(t, m)
	assert.NotNil(t, m.tunnels)
	assert.Equal(t, mockMP, m.mp)
}

func TestManager_Create_Success(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "test-vm").Return(testutil.RunningVM("test-vm", "192.168.64.5"), nil)

	m := NewManager(mockMP)

	tunnel, err := m.Create("test-vm", 8080)
	require.NoError(t, err)
	require.NotNil(t, tunnel)

	assert.Equal(t, "test-vm", tunnel.VMName)
	assert.Equal(t, 8080, tunnel.VMPort)
	assert.Greater(t, tunnel.HostPort, 0)
	assert.Equal(t, "192.168.64.5", tunnel.vmIP)

	// Clean up
	m.Delete(tunnel.HostPort)

	mockMP.AssertExpectations(t)
}

func TestManager_Create_VMNotRunning(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "stopped-vm").Return(testutil.StoppedVM("stopped-vm"), nil)

	m := NewManager(mockMP)

	tunnel, err := m.Create("stopped-vm", 8080)
	assert.Error(t, err)
	assert.Nil(t, tunnel)
	assert.Contains(t, err.Error(), "not running")

	mockMP.AssertExpectations(t)
}

func TestManager_Create_VMNotFound(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "nonexistent").Return(nil, errors.New("VM not found"))

	m := NewManager(mockMP)

	tunnel, err := m.Create("nonexistent", 8080)
	assert.Error(t, err)
	assert.Nil(t, tunnel)
	assert.Contains(t, err.Error(), "failed to get VM info")

	mockMP.AssertExpectations(t)
}

func TestManager_Create_VMNoIP(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	vmInfo := &multipass.InstanceInfo{
		State: multipass.StateRunning,
		IPv4:  []string{},
	}
	mockMP.On("Info", "no-ip-vm").Return(vmInfo, nil)

	m := NewManager(mockMP)

	tunnel, err := m.Create("no-ip-vm", 8080)
	assert.Error(t, err)
	assert.Nil(t, tunnel)
	assert.Contains(t, err.Error(), "no IP address")

	mockMP.AssertExpectations(t)
}

func TestManager_Delete_Success(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "test-vm").Return(testutil.RunningVM("test-vm", "192.168.64.5"), nil)

	m := NewManager(mockMP)

	tunnel, err := m.Create("test-vm", 8080)
	require.NoError(t, err)

	err = m.Delete(tunnel.HostPort)
	assert.NoError(t, err)

	// Verify tunnel was removed
	tunnels := m.List()
	assert.Empty(t, tunnels)

	mockMP.AssertExpectations(t)
}

func TestManager_Delete_NotFound(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	m := NewManager(mockMP)

	err := m.Delete(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tunnel not found")
}

func TestManager_List(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "vm1").Return(testutil.RunningVM("vm1", "192.168.64.5"), nil)
	mockMP.On("Info", "vm2").Return(testutil.RunningVM("vm2", "192.168.64.6"), nil)

	m := NewManager(mockMP)

	// Initially empty
	tunnels := m.List()
	assert.Empty(t, tunnels)

	// Create two tunnels
	t1, err := m.Create("vm1", 8080)
	require.NoError(t, err)

	t2, err := m.Create("vm2", 3000)
	require.NoError(t, err)

	tunnels = m.List()
	assert.Len(t, tunnels, 2)

	// Clean up
	m.Delete(t1.HostPort)
	m.Delete(t2.HostPort)

	tunnels = m.List()
	assert.Empty(t, tunnels)

	mockMP.AssertExpectations(t)
}

func TestManager_ConcurrentAccess(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Info", "test-vm").Return(testutil.RunningVM("test-vm", "192.168.64.5"), nil)

	m := NewManager(mockMP)

	// Create multiple tunnels concurrently
	done := make(chan *Tunnel, 10)
	for i := 0; i < 10; i++ {
		go func(port int) {
			tunnel, err := m.Create("test-vm", 8000+port)
			if err == nil {
				done <- tunnel
			} else {
				done <- nil
			}
		}(i)
	}

	// Collect results
	var tunnels []*Tunnel
	for i := 0; i < 10; i++ {
		if t := <-done; t != nil {
			tunnels = append(tunnels, t)
		}
	}

	// Clean up
	for _, t := range tunnels {
		m.Delete(t.HostPort)
	}
}
