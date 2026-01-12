package watchdog

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mjshashank/dabbi/internal/multipass"
)

const (
	checkpointPath       = "/tmp/dabbi-activity.json"
	loadAverageThreshold = 0.1    // Consider VM active if 1-min load avg exceeds this
	networkNoiseBytes    = 100000 // ~100KB/min threshold to filter out background noise (DHCP, NTP, etc.)
)

// checkpoint stores activity state inside the VM
type checkpoint struct {
	Timestamp string `json:"timestamp"`
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
}

// activityStats holds all activity indicators queried from a VM
type activityStats struct {
	RxBytes         uint64
	TxBytes         uint64
	PTYIdleSeconds  int     // Seconds since last PTY activity (-1 if no PTY)
	LoadAverage1Min float64
}

// Watchdog monitors VM activity and stops inactive VMs.
// Activity is determined by: PTY sessions, CPU load, or network traffic.
// State is stored inside each VM at /tmp/dabbi-activity.json, making the daemon stateless.
type Watchdog struct {
	timeout time.Duration
	mp      multipass.Client
	stopCh  chan struct{}
}

// New creates a new watchdog that monitors VMs for inactivity
func New(mp multipass.Client, timeout time.Duration) *Watchdog {
	w := &Watchdog{
		timeout: timeout,
		mp:      mp,
		stopCh:  make(chan struct{}),
	}
	go w.run()
	return w
}

// Stop shuts down the watchdog
func (w *Watchdog) Stop() {
	close(w.stopCh)
}

// GetTimeout returns the inactivity timeout
func (w *Watchdog) GetTimeout() time.Duration {
	return w.timeout
}

// run is the main watchdog loop
func (w *Watchdog) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkAllVMs()
		}
	}
}

// checkAllVMs queries all running VMs and stops inactive ones
func (w *Watchdog) checkAllVMs() {
	vms, err := w.mp.List()
	if err != nil {
		return
	}

	for _, vm := range vms {
		if vm.State == multipass.StateRunning {
			w.checkVM(vm.Name)
		}
	}
}

// checkVM checks a single VM for inactivity using hybrid detection
func (w *Watchdog) checkVM(vmName string) {
	stats, err := w.getActivityStats(vmName)
	if err != nil {
		return // Skip this VM, try again next tick
	}

	// Check immediate activity indicators (no history needed)
	if w.hasImmediateActivity(stats) {
		w.writeCheckpoint(vmName, stats.RxBytes, stats.TxBytes)
		return
	}

	// Check network stats against checkpoint
	prev, err := w.readCheckpoint(vmName)
	if err != nil {
		// No checkpoint exists - create initial one
		w.writeCheckpoint(vmName, stats.RxBytes, stats.TxBytes)
		return
	}

	checkpointTime, err := time.Parse(time.RFC3339, prev.Timestamp)
	if err != nil {
		w.writeCheckpoint(vmName, stats.RxBytes, stats.TxBytes)
		return
	}

	// Check if network stats changed significantly (above background noise)
	totalDelta := absDiff(stats.RxBytes, prev.RxBytes) + absDiff(stats.TxBytes, prev.TxBytes)
	if totalDelta > networkNoiseBytes {
		w.writeCheckpoint(vmName, stats.RxBytes, stats.TxBytes)
		return
	}

	// No significant activity - check if timeout exceeded
	if time.Since(checkpointTime) > w.timeout {
		log.Printf("[watchdog] stopping inactive VM: %s", vmName)
		go func(name string) {
			_ = w.mp.Stop(name)
		}(vmName)
	}
}

// hasImmediateActivity checks for activity indicators that don't need history
func (w *Watchdog) hasImmediateActivity(stats *activityStats) bool {
	// Active PTY with recent activity (idle time < timeout)
	if stats.PTYIdleSeconds >= 0 && stats.PTYIdleSeconds < int(w.timeout.Seconds()) {
		return true
	}

	// High load average indicates CPU work
	if stats.LoadAverage1Min > loadAverageThreshold {
		return true
	}

	return false
}

// getActivityStats queries all activity indicators from the VM in one exec call
func (w *Watchdog) getActivityStats(vmName string) (*activityStats, error) {
	// Combined command to get all stats efficiently:
	// 1. Network bytes from /proc/net/dev
	// 2. PTY idle time in seconds (min across all PTYs, -1 if none)
	// 3. Load average
	cmd := `awk 'NR>2 {rx+=$2; tx+=$10} END {print rx, tx}' /proc/net/dev; ` +
		`now=$(date +%s); idle=-1; for p in /dev/pts/[0-9]*; do [ -e "$p" ] && { t=$(stat -c %Y "$p"); i=$((now-t)); [ $idle -lt 0 ] || [ $i -lt $idle ] && idle=$i; }; done; echo $idle; ` +
		`cut -d' ' -f1 /proc/loadavg`

	output, err := w.mp.Exec(vmName, "sh", "-c", cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		return nil, fmt.Errorf("unexpected output: %s", output)
	}

	stats := &activityStats{}

	if parts := strings.Fields(lines[0]); len(parts) == 2 {
		stats.RxBytes, _ = strconv.ParseUint(parts[0], 10, 64)
		stats.TxBytes, _ = strconv.ParseUint(parts[1], 10, 64)
	}
	stats.PTYIdleSeconds, _ = strconv.Atoi(strings.TrimSpace(lines[1]))
	stats.LoadAverage1Min, _ = strconv.ParseFloat(strings.TrimSpace(lines[2]), 64)

	return stats, nil
}

// readCheckpoint reads the activity checkpoint from the VM
func (w *Watchdog) readCheckpoint(vmName string) (*checkpoint, error) {
	output, err := w.mp.Exec(vmName, "cat", checkpointPath)
	if err != nil {
		return nil, err
	}

	var cp checkpoint
	if err := json.Unmarshal([]byte(output), &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

// writeCheckpoint writes the activity checkpoint to the VM
func (w *Watchdog) writeCheckpoint(vmName string, rxBytes, txBytes uint64) {
	cp := checkpoint{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RxBytes:   rxBytes,
		TxBytes:   txBytes,
	}

	data, err := json.Marshal(cp)
	if err != nil {
		return
	}

	cmd := fmt.Sprintf("echo '%s' > %s", string(data), checkpointPath)
	_, _ = w.mp.Exec(vmName, "sh", "-c", cmd)
}

// absDiff returns the absolute difference between two uint64 values
func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
