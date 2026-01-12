package cli

import (
	"fmt"
	"strings"

	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/network"
	"github.com/spf13/cobra"
)

func newNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage VM network restrictions",
		Long: `Manage network restrictions for VMs.

Supports three modes:
  - allowlist: Only allow specified hosts (block everything else)
  - blocklist: Block specified hosts (allow everything else)
  - isolated:  No network access at all`,
	}

	cmd.AddCommand(
		newNetworkGetCmd(),
		newNetworkSetCmd(),
		newNetworkRemoveCmd(),
		newNetworkApplyCmd(),
	)

	return cmd
}

func newNetworkGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <vm-name>",
		Short: "Get current network configuration for a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			// Check if VM exists and is running
			info, err := mpClient.Info(vmName)
			if err != nil {
				return fmt.Errorf("VM not found: %w", err)
			}

			if info.State != multipass.StateRunning {
				return fmt.Errorf("VM must be running to query network config (current state: %s)", info.State)
			}

			applier := network.NewApplier(mpClient)
			config, err := applier.GetCurrentConfig(vmName)
			if err != nil {
				return fmt.Errorf("failed to get network config: %w", err)
			}

			if config == nil || config.Mode == multipass.NetworkModeNone {
				fmt.Printf("Network mode: none (no restrictions)\n")
				return nil
			}

			fmt.Printf("Network mode: %s\n", config.Mode)
			if len(config.Rules) > 0 {
				fmt.Printf("Rules:\n")
				for _, rule := range config.Rules {
					comment := ""
					if rule.Comment != "" {
						comment = fmt.Sprintf(" (%s)", rule.Comment)
					}
					fmt.Printf("  - %s: %s%s\n", rule.Type, rule.Value, comment)
				}
			}
			return nil
		},
	}
}

func newNetworkSetCmd() *cobra.Command {
	var (
		mode        string
		allowHosts  []string
		blockHosts  []string
	)

	cmd := &cobra.Command{
		Use:   "set <vm-name>",
		Short: "Set network restrictions for a VM",
		Long: `Set network restrictions for a VM.

Examples:
  # Allow only specific hosts
  dabbi network set my-vm --mode allowlist --allow github.com --allow 10.0.0.0/8

  # Block specific hosts
  dabbi network set my-vm --mode blocklist --block facebook.com --block 192.168.1.100

  # Completely isolate VM from network
  dabbi network set my-vm --mode isolated`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			// Validate mode
			var networkMode multipass.NetworkMode
			switch mode {
			case "none":
				networkMode = multipass.NetworkModeNone
			case "allowlist":
				networkMode = multipass.NetworkModeAllowlist
			case "blocklist":
				networkMode = multipass.NetworkModeBlocklist
			case "isolated":
				networkMode = multipass.NetworkModeIsolated
			default:
				return fmt.Errorf("invalid mode: %s (must be none, allowlist, blocklist, or isolated)", mode)
			}

			// Build rules
			var rules []multipass.NetworkRule

			if networkMode == multipass.NetworkModeAllowlist {
				for _, host := range allowHosts {
					rule := parseHostToRule(host)
					rules = append(rules, rule)
				}
				if len(rules) == 0 {
					return fmt.Errorf("allowlist mode requires at least one --allow flag")
				}
			} else if networkMode == multipass.NetworkModeBlocklist {
				for _, host := range blockHosts {
					rule := parseHostToRule(host)
					rules = append(rules, rule)
				}
				if len(rules) == 0 {
					return fmt.Errorf("blocklist mode requires at least one --block flag")
				}
			}

			config := &multipass.NetworkConfig{
				Mode:  networkMode,
				Rules: rules,
			}

			// Validate config
			if err := network.ValidateConfig(config); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			// Check if VM exists and is running
			info, err := mpClient.Info(vmName)
			if err != nil {
				return fmt.Errorf("VM not found: %w", err)
			}

			if info.State != multipass.StateRunning {
				return fmt.Errorf("VM must be running to set network config (current state: %s)", info.State)
			}

			fmt.Printf("Applying network config (mode=%s) to VM '%s'...\n", mode, vmName)

			applier := network.NewApplier(mpClient)
			if err := applier.ApplyToVM(vmName, config); err != nil {
				return fmt.Errorf("failed to apply network config: %w", err)
			}

			fmt.Printf("Network restrictions applied successfully\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "", "Network mode: none, allowlist, blocklist, isolated (required)")
	cmd.Flags().StringArrayVar(&allowHosts, "allow", nil, "Host to allow (IP, CIDR, or domain) - use with allowlist mode")
	cmd.Flags().StringArrayVar(&blockHosts, "block", nil, "Host to block (IP, CIDR, or domain) - use with blocklist mode")
	cmd.MarkFlagRequired("mode")

	return cmd
}

func newNetworkRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <vm-name>",
		Short: "Remove all network restrictions from a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			// Check if VM exists and is running
			info, err := mpClient.Info(vmName)
			if err != nil {
				return fmt.Errorf("VM not found: %w", err)
			}

			if info.State != multipass.StateRunning {
				return fmt.Errorf("VM must be running to remove network config (current state: %s)", info.State)
			}

			fmt.Printf("Removing network restrictions from VM '%s'...\n", vmName)

			applier := network.NewApplier(mpClient)
			if err := applier.RemoveFromVM(vmName); err != nil {
				return fmt.Errorf("failed to remove network config: %w", err)
			}

			fmt.Printf("Network restrictions removed successfully\n")
			return nil
		},
	}
}

func newNetworkApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <vm-name>",
		Short: "Re-apply current network configuration to a VM",
		Long:  `Re-apply the current network configuration. Useful after VM restart or if rules were manually cleared.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			// Check if VM exists and is running
			info, err := mpClient.Info(vmName)
			if err != nil {
				return fmt.Errorf("VM not found: %w", err)
			}

			if info.State != multipass.StateRunning {
				return fmt.Errorf("VM must be running to apply network config (current state: %s)", info.State)
			}

			applier := network.NewApplier(mpClient)

			// Get current config
			config, err := applier.GetCurrentConfig(vmName)
			if err != nil {
				return fmt.Errorf("failed to get current config: %w", err)
			}

			if config == nil {
				return fmt.Errorf("no network configuration found for VM '%s'", vmName)
			}

			fmt.Printf("Re-applying network config (mode=%s) to VM '%s'...\n", config.Mode, vmName)

			if err := applier.ApplyToVM(vmName, config); err != nil {
				return fmt.Errorf("failed to apply network config: %w", err)
			}

			fmt.Printf("Network rules re-applied successfully\n")
			return nil
		},
	}
}

// parseHostToRule converts a host string to a NetworkRule
// It auto-detects the type based on the format
func parseHostToRule(host string) multipass.NetworkRule {
	// Check if it's a CIDR
	if strings.Contains(host, "/") {
		return multipass.NetworkRule{Type: "cidr", Value: host}
	}

	// Check if it looks like an IP address
	if isIPLike(host) {
		return multipass.NetworkRule{Type: "ip", Value: host}
	}

	// Otherwise, treat as domain
	return multipass.NetworkRule{Type: "domain", Value: host}
}

// isIPLike checks if a string looks like an IP address
func isIPLike(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
