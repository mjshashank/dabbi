package cli

import (
	"fmt"

	"github.com/mjshashank/dabbi/internal/config"
	"github.com/mjshashank/dabbi/internal/daemon"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var (
		port   int
		domain string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the dabbi daemon",
		Long: `Start the dabbi daemon server.

The daemon provides:
  - HTTP routing to VMs via <vm>-<port>.localhost
  - Wake-on-request for stopped VMs
  - REST API for VM management
  - WebSocket terminal access
  - Web UI for VM management

Note: Port 80 requires sudo or capabilities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure default cloud-init exists
			cloudInitPath, created, err := config.EnsureDefaultCloudInit()
			if err != nil {
				fmt.Printf("Warning: could not create default cloud-init: %v\n", err)
			} else if created {
				fmt.Printf("Created default cloud-init: %s\n", cloudInitPath)
			}

			srv := daemon.NewServer(daemon.ServerConfig{
				Port:            port,
				Domain:          domain,
				Config:          cfg,
				MultipassClient: mpClient,
			})

			fmt.Printf("Starting dabbi daemon on port %d...\n", port)
			if domain != "" {
				fmt.Printf("TLS enabled for domain: %s\n", domain)
			}
			fmt.Printf("Auth token: %s\n", cfg.AuthToken)
			fmt.Printf("\nVM routing: http://<vm>-<port>.localhost:%d\n", port)
			fmt.Printf("API: http://localhost:%d/api/\n", port)
			fmt.Printf("UI: http://localhost:%d/\n", port)

			return srv.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&port, "port", 80, "Port to listen on")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain for automatic TLS (Let's Encrypt)")

	return cmd
}
