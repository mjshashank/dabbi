package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/mjshashank/dabbi/internal/network"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var (
		cpus         int
		memory       string
		disk         string
		cloudInit    string
		image        string
		networkMode  string
		networkAllow []string
		networkBlock []string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new VM",
		Long: `Create a new VM with the specified configuration.

If options are not provided, defaults from ~/.dabbi/config.json are used.

Network restrictions can be applied at creation time:
  dabbi create my-vm --network-mode allowlist --allow github.com
  dabbi create my-vm --network-mode isolated`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Use defaults from config if not specified
			if cpus == 0 {
				cpus = cfg.Defaults.CPU
			}
			if memory == "" {
				memory = cfg.Defaults.Mem
			}
			if disk == "" {
				disk = cfg.Defaults.Disk
			}

			// Resolve cloud-init path (explicit > config default > ~/.dabbi/cloud-init.yaml)
			resolvedCloudInit := cfg.GetCloudInitPath(cloudInit)

			// Build network config if specified
			var netConfig *multipass.NetworkConfig
			if networkMode != "" {
				var mode multipass.NetworkMode
				switch networkMode {
				case "none":
					mode = multipass.NetworkModeNone
				case "allowlist":
					mode = multipass.NetworkModeAllowlist
				case "blocklist":
					mode = multipass.NetworkModeBlocklist
				case "isolated":
					mode = multipass.NetworkModeIsolated
				default:
					return fmt.Errorf("invalid network mode: %s", networkMode)
				}

				var rules []multipass.NetworkRule
				if mode == multipass.NetworkModeAllowlist {
					for _, host := range networkAllow {
						rules = append(rules, parseNetworkHost(host))
					}
					if len(rules) == 0 {
						return fmt.Errorf("allowlist mode requires at least one --allow")
					}
				} else if mode == multipass.NetworkModeBlocklist {
					for _, host := range networkBlock {
						rules = append(rules, parseNetworkHost(host))
					}
					if len(rules) == 0 {
						return fmt.Errorf("blocklist mode requires at least one --block")
					}
				}

				netConfig = &multipass.NetworkConfig{Mode: mode, Rules: rules}

				if err := network.ValidateConfig(netConfig); err != nil {
					return fmt.Errorf("invalid network config: %w", err)
				}
			} else if cfg.Defaults.NetworkConfig != nil && cfg.Defaults.NetworkConfig.Mode != multipass.NetworkModeNone {
				// Use default network config if set
				netConfig = cfg.Defaults.NetworkConfig
			}

			// If we have network config, generate modified cloud-init
			var finalCloudInit string
			var tempCloudInitFile string
			if netConfig != nil && netConfig.Mode != multipass.NetworkModeNone {
				// Read base cloud-init
				var baseContent string
				if resolvedCloudInit != "" {
					data, err := os.ReadFile(resolvedCloudInit)
					if err != nil {
						return fmt.Errorf("failed to read cloud-init: %w", err)
					}
					baseContent = string(data)
				} else {
					baseContent = config.DefaultCloudInit
				}

				// Generate cloud-init with network config
				modifiedContent, err := config.GenerateCloudInitWithNetwork(baseContent, netConfig)
				if err != nil {
					return fmt.Errorf("failed to generate cloud-init with network: %w", err)
				}

				// Inject auth token for OpenCode
				modifiedContent = config.GenerateCloudInitWithAuthToken(modifiedContent, cfg.AuthToken)

				// Write to temp file
				tmpDir, err := os.MkdirTemp("", "dabbi-cloudinit-*")
				if err != nil {
					return fmt.Errorf("failed to create temp dir: %w", err)
				}
				defer os.RemoveAll(tmpDir)

				tempCloudInitFile = filepath.Join(tmpDir, "cloud-init.yaml")
				if err := os.WriteFile(tempCloudInitFile, []byte(modifiedContent), 0644); err != nil {
					return fmt.Errorf("failed to write temp cloud-init: %w", err)
				}

				finalCloudInit = tempCloudInitFile
				fmt.Printf("Network mode: %s\n", netConfig.Mode)
			} else {
				// No network config, but still need to inject auth token for OpenCode
				var baseContent string
				if resolvedCloudInit != "" {
					data, err := os.ReadFile(resolvedCloudInit)
					if err != nil {
						return fmt.Errorf("failed to read cloud-init: %w", err)
					}
					baseContent = string(data)
				} else {
					baseContent = config.DefaultCloudInit
				}

				// Inject auth token
				modifiedContent := config.GenerateCloudInitWithAuthToken(baseContent, cfg.AuthToken)

				// Write to temp file
				tmpDir, err := os.MkdirTemp("", "dabbi-cloudinit-*")
				if err != nil {
					return fmt.Errorf("failed to create temp dir: %w", err)
				}
				defer os.RemoveAll(tmpDir)

				tempCloudInitFile = filepath.Join(tmpDir, "cloud-init.yaml")
				if err := os.WriteFile(tempCloudInitFile, []byte(modifiedContent), 0644); err != nil {
					return fmt.Errorf("failed to write temp cloud-init: %w", err)
				}

				finalCloudInit = tempCloudInitFile
			}

			opts := multipass.LaunchOptions{
				Name:          name,
				CPUs:          cpus,
				Memory:        memory,
				Disk:          disk,
				CloudInit:     finalCloudInit,
				Image:         image,
				NetworkConfig: netConfig,
			}

			fmt.Printf("Creating VM '%s' (cpus=%d, memory=%s, disk=%s)...\n",
				name, cpus, memory, disk)
			if resolvedCloudInit != "" {
				fmt.Printf("Using cloud-init: %s\n", resolvedCloudInit)
			}

			if err := mpClient.Launch(opts); err != nil {
				return err
			}

			fmt.Printf("VM '%s' created successfully\n", name)
			return nil
		},
	}

	cmd.Flags().IntVar(&cpus, "cpu", 0, "Number of CPUs (default from config)")
	cmd.Flags().StringVar(&memory, "mem", "", "Memory size, e.g., 4G (default from config)")
	cmd.Flags().StringVar(&disk, "disk", "", "Disk size, e.g., 20G (default from config)")
	cmd.Flags().StringVar(&cloudInit, "cloud-init", "", "Path to cloud-init file (default: ~/.dabbi/cloud-init.yaml if exists)")
	cmd.Flags().StringVar(&image, "image", "", "Image to use, e.g., 22.04 or jammy")
	cmd.Flags().StringVar(&networkMode, "network-mode", "", "Network restriction mode: none, allowlist, blocklist, isolated")
	cmd.Flags().StringArrayVar(&networkAllow, "allow", nil, "Host to allow (use with --network-mode=allowlist)")
	cmd.Flags().StringArrayVar(&networkBlock, "block", nil, "Host to block (use with --network-mode=blocklist)")

	return cmd
}

// parseNetworkHost converts a host string to a NetworkRule
func parseNetworkHost(host string) multipass.NetworkRule {
	if strings.Contains(host, "/") {
		return multipass.NetworkRule{Type: "cidr", Value: host}
	}
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		allDigits := true
		for _, p := range parts {
			for _, c := range p {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
		}
		if allDigits {
			return multipass.NetworkRule{Type: "ip", Value: host}
		}
	}
	return multipass.NetworkRule{Type: "domain", Value: host}
}
