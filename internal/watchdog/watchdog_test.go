package watchdog

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	w := New(mockMP, 30*time.Minute)

	require.NotNil(t, w)
	assert.Equal(t, 30*time.Minute, w.timeout)
	assert.Equal(t, mockMP, w.mp)

	// Clean up
	w.Stop()
}

func TestWatchdog_GetTimeout(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	w := New(mockMP, 45*time.Minute)
	defer w.Stop()

	assert.Equal(t, 45*time.Minute, w.GetTimeout())
}

func TestWatchdog_Stop(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	w := New(mockMP, 30*time.Minute)

	// Should not panic
	w.Stop()
}

func TestAbsDiff(t *testing.T) {
	tests := []struct {
		a, b   uint64
		expect uint64
	}{
		{10, 5, 5},
		{5, 10, 5},
		{0, 0, 0},
		{100, 100, 0},
		{1000000, 500000, 500000},
	}

	for _, tt := range tests {
		result := absDiff(tt.a, tt.b)
		assert.Equal(t, tt.expect, result)
	}
}

func TestHasImmediateActivity(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	w := New(mockMP, 30*time.Minute)
	defer w.Stop()

	tests := []struct {
		name   string
		stats  *activityStats
		expect bool
	}{
		{
			name: "active PTY",
			stats: &activityStats{
				PTYIdleSeconds:  60, // 1 minute, less than 30 min timeout
				LoadAverage1Min: 0.01,
			},
			expect: true,
		},
		{
			name: "high CPU load",
			stats: &activityStats{
				PTYIdleSeconds:  -1, // No PTY
				LoadAverage1Min: 0.5,
			},
			expect: true,
		},
		{
			name: "no PTY, low load",
			stats: &activityStats{
				PTYIdleSeconds:  -1,
				LoadAverage1Min: 0.01,
			},
			expect: false,
		},
		{
			name: "stale PTY",
			stats: &activityStats{
				PTYIdleSeconds:  3600, // 1 hour, more than 30 min timeout
				LoadAverage1Min: 0.01,
			},
			expect: false,
		},
		{
			name: "exactly at threshold",
			stats: &activityStats{
				PTYIdleSeconds:  -1,
				LoadAverage1Min: loadAverageThreshold,
			},
			expect: false,
		},
		{
			name: "just above threshold",
			stats: &activityStats{
				PTYIdleSeconds:  -1,
				LoadAverage1Min: loadAverageThreshold + 0.01,
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := w.hasImmediateActivity(tt.stats)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCheckAllVMs(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)

	// Mock the List call to return mixed VMs
	mockMP.On("List").Return([]multipass.ListInstance{
		{Name: "running-vm", State: multipass.StateRunning},
		{Name: "stopped-vm", State: multipass.StateStopped},
	}, nil)

	// For running VMs, we need to mock the Exec calls for activity stats
	// Note: Exec receives (vmName string, cmd []string) due to variadic
	mockMP.On("Exec", "running-vm", mock.MatchedBy(func(cmd []string) bool {
		return len(cmd) >= 2 && cmd[0] == "sh" && cmd[1] == "-c"
	})).Return("1000 2000\n60\n0.5", nil).Maybe()
	mockMP.On("Exec", "running-vm", []string{"cat", checkpointPath}).Return("", nil).Maybe()

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	// Call checkAllVMs directly
	w.checkAllVMs()

	// The important thing is that it doesn't panic and only checks running VMs
}

func TestCheckVM_WithImmediateActivity(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)

	// Activity stats with high CPU load (immediate activity)
	mockMP.On("Exec", "active-vm", mock.MatchedBy(func(cmd []string) bool {
		return len(cmd) >= 2 && cmd[0] == "sh" && cmd[1] == "-c"
	})).Return("1000 2000\n-1\n0.8", nil).Maybe()

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	w.checkVM("active-vm")
	// Should not stop VM since it has activity
}

func TestReadCheckpoint(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)

	cp := checkpoint{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RxBytes:   1000,
		TxBytes:   2000,
	}
	cpJSON, _ := json.Marshal(cp)

	mockMP.On("Exec", "test-vm", []string{"cat", checkpointPath}).Return(string(cpJSON), nil)

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	result, err := w.readCheckpoint("test-vm")
	require.NoError(t, err)
	assert.Equal(t, cp.RxBytes, result.RxBytes)
	assert.Equal(t, cp.TxBytes, result.TxBytes)

	mockMP.AssertExpectations(t)
}

func TestReadCheckpoint_Error(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Exec", "test-vm", []string{"cat", checkpointPath}).Return("", assert.AnError)

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	result, err := w.readCheckpoint("test-vm")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestReadCheckpoint_InvalidJSON(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Exec", "test-vm", []string{"cat", checkpointPath}).Return("not valid json", nil)

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	result, err := w.readCheckpoint("test-vm")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetActivityStats(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)

	// Mock output: "rx_bytes tx_bytes\npty_idle\nload_avg"
	mockMP.On("Exec", "test-vm", mock.MatchedBy(func(cmd []string) bool {
		return len(cmd) >= 2 && cmd[0] == "sh" && cmd[1] == "-c"
	})).Return("123456 789012\n120\n0.25", nil)

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	stats, err := w.getActivityStats("test-vm")
	require.NoError(t, err)

	assert.Equal(t, uint64(123456), stats.RxBytes)
	assert.Equal(t, uint64(789012), stats.TxBytes)
	assert.Equal(t, 120, stats.PTYIdleSeconds)
	assert.InDelta(t, 0.25, stats.LoadAverage1Min, 0.001)

	mockMP.AssertExpectations(t)
}

func TestGetActivityStats_Error(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Exec", "test-vm", mock.MatchedBy(func(cmd []string) bool {
		return len(cmd) >= 2 && cmd[0] == "sh" && cmd[1] == "-c"
	})).Return("", assert.AnError)

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	stats, err := w.getActivityStats("test-vm")
	assert.Error(t, err)
	assert.Nil(t, stats)
}

func TestGetActivityStats_InvalidOutput(t *testing.T) {
	mockMP := new(testutil.MockMultipassClient)
	mockMP.On("Exec", "test-vm", mock.MatchedBy(func(cmd []string) bool {
		return len(cmd) >= 2 && cmd[0] == "sh" && cmd[1] == "-c"
	})).Return("invalid output", nil)

	w := &Watchdog{
		timeout: 30 * time.Minute,
		mp:      mockMP,
		stopCh:  make(chan struct{}),
	}

	stats, err := w.getActivityStats("test-vm")
	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "unexpected output")
}
