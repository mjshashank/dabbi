package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mjshashank/dabbi/internal/multipass"
)

const (
	// Path inside VM where network config and scripts are stored
	vmNetworkDir    = "/opt/dabbi/network"
	vmConfigFile    = "/opt/dabbi/network/config.json"
	vmScriptFile    = "/opt/dabbi/network/apply-rules.sh"
	vmServiceFile   = "/etc/systemd/system/dabbi-network.service"
)

// Applier handles applying network rules to VMs
type Applier struct {
	mp multipass.Client
}

// NewApplier creates a new network applier
func NewApplier(mp multipass.Client) *Applier {
	return &Applier{mp: mp}
}

// ApplyToVM applies network configuration to a running VM
func (a *Applier) ApplyToVM(vmName string, config *multipass.NetworkConfig) error {
	if config == nil {
		config = &multipass.NetworkConfig{Mode: multipass.NetworkModeNone}
	}

	// Validate the configuration
	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid network config: %w", err)
	}

	// Generate the iptables script
	script, err := GenerateIptablesScript(config)
	if err != nil {
		return fmt.Errorf("failed to generate iptables script: %w", err)
	}

	// Create temp files for transfer
	tmpDir, err := os.MkdirTemp("", "dabbi-network-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write config JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Write script
	scriptPath := filepath.Join(tmpDir, "apply-rules.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	// Write systemd service
	servicePath := filepath.Join(tmpDir, "dabbi-network.service")
	if err := os.WriteFile(servicePath, []byte(GenerateSystemdService()), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Ensure the network directory exists in VM
	if _, err := a.mp.Exec(vmName, "sudo", "mkdir", "-p", vmNetworkDir); err != nil {
		return fmt.Errorf("failed to create network dir in VM: %w", err)
	}

	// Transfer files to /tmp first (multipass transfer runs as ubuntu user)
	if err := a.mp.Transfer(configPath, fmt.Sprintf("%s:/tmp/dabbi-config.json", vmName)); err != nil {
		return fmt.Errorf("failed to transfer config: %w", err)
	}
	if err := a.mp.Transfer(scriptPath, fmt.Sprintf("%s:/tmp/dabbi-apply-rules.sh", vmName)); err != nil {
		return fmt.Errorf("failed to transfer script: %w", err)
	}
	if err := a.mp.Transfer(servicePath, fmt.Sprintf("%s:/tmp/dabbi-network.service", vmName)); err != nil {
		return fmt.Errorf("failed to transfer service: %w", err)
	}

	// Move files to final locations (requires sudo)
	if _, err := a.mp.Exec(vmName, "sudo", "mv", "/tmp/dabbi-config.json", vmConfigFile); err != nil {
		return fmt.Errorf("failed to install config: %w", err)
	}
	if _, err := a.mp.Exec(vmName, "sudo", "mv", "/tmp/dabbi-apply-rules.sh", vmScriptFile); err != nil {
		return fmt.Errorf("failed to install script: %w", err)
	}
	if _, err := a.mp.Exec(vmName, "sudo", "mv", "/tmp/dabbi-network.service", vmServiceFile); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	// Make script executable
	if _, err := a.mp.Exec(vmName, "sudo", "chmod", "+x", vmScriptFile); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	// Reload systemd and enable service
	if _, err := a.mp.Exec(vmName, "sudo", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	if _, err := a.mp.Exec(vmName, "sudo", "systemctl", "enable", "dabbi-network.service"); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Execute the script to apply rules immediately
	if _, err := a.mp.Exec(vmName, "sudo", vmScriptFile); err != nil {
		return fmt.Errorf("failed to apply rules: %w", err)
	}

	return nil
}

// GetCurrentConfig retrieves the current network configuration from a VM
func (a *Applier) GetCurrentConfig(vmName string) (*multipass.NetworkConfig, error) {
	// Try to read the config file from the VM
	output, err := a.mp.Exec(vmName, "cat", vmConfigFile)
	if err != nil {
		// Check if it's just a "file not found" error
		if strings.Contains(err.Error(), "No such file") {
			return nil, nil // No config = no restrictions
		}
		return nil, fmt.Errorf("failed to read config from VM: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var config multipass.NetworkConfig
	if err := json.Unmarshal([]byte(output), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// RemoveFromVM removes all network restrictions from a VM
func (a *Applier) RemoveFromVM(vmName string) error {
	// Apply "none" mode to remove all restrictions
	return a.ApplyToVM(vmName, &multipass.NetworkConfig{Mode: multipass.NetworkModeNone})
}

// IsConfigured checks if a VM has network restrictions configured
func (a *Applier) IsConfigured(vmName string) (bool, error) {
	_, err := a.mp.Exec(vmName, "test", "-f", vmConfigFile)
	if err != nil {
		// File doesn't exist or error
		return false, nil
	}
	return true, nil
}
