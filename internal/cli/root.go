package cli

import (
	"fmt"
	"os"

	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/multipass"
	"github.com/spf13/cobra"
)

var (
	cfg       *config.Config
	mpClient  multipass.Client
	version   = "dev"
	buildTime = "unknown"
)

// SetVersion sets the version and build time for the CLI
func SetVersion(v, bt string) {
	version = v
	buildTime = bt
}

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dabbi",
		Short: "dabbi - Minimalist VM Manager",
		Long: `dabbi is a tool for managing ephemeral, scalable, and persistent
development environments using multipass VMs.

It provides:
  - Zero-config HTTP routing to VMs
  - Wake-on-request for stopped VMs
  - Ephemeral TCP tunnels for database/SSH access
  - Web terminal and file browser
  - Automatic inactivity shutdown`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip config loading for help commands
			if cmd.Name() == "help" || cmd.Name() == "version" {
				return nil
			}

			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			mpClient = multipass.NewRealClient()
			return nil
		},
		SilenceUsage: true,
	}

	// Add subcommands
	rootCmd.AddCommand(
		newServeCmd(),
		newListCmd(),
		newCreateCmd(),
		newStartCmd(),
		newStopCmd(),
		newRestartCmd(),
		newDeleteCmd(),
		newCloneCmd(),
		newSnapshotCmd(),
		newShellCmd(),
		newAgentCmd(),
		newTunnelCmd(),
		newMountCmd(),
		newCpCmd(),
		newNetworkCmd(),
		newVersionCmd(),
	)

	return rootCmd
}

// Execute runs the root command
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dabbi version %s (built %s)\n", version, buildTime)
		},
	}
}
