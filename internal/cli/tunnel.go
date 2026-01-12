package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/mjshashank/dabbi/internal/tunnel"
	"github.com/spf13/cobra"
)

func newTunnelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tunnel <vm_name> <vm_port>",
		Short: "Create a TCP tunnel to a VM port",
		Long: `Create a TCP tunnel to a port inside a VM.

This opens a local port that forwards traffic to the specified
port inside the VM. Useful for database connections (PostgreSQL, MySQL)
or SSH access.

The tunnel stays open until you press Ctrl+C.

Example:
  dabbi tunnel my-db 5432
  # Then connect to localhost:<printed_port>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]
			vmPort, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid port: %s", args[1])
			}

			// Create tunnel manager with multipass client
			tm := tunnel.NewManager(mpClient)

			fmt.Printf("Creating tunnel to %s:%d...\n", vmName, vmPort)

			t, err := tm.Create(vmName, vmPort)
			if err != nil {
				return fmt.Errorf("failed to create tunnel: %w", err)
			}

			fmt.Printf("Tunnel created: localhost:%d -> %s:%d\n", t.HostPort, vmName, vmPort)
			fmt.Println("Press Ctrl+C to close")

			// Wait for interrupt
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nClosing tunnel...")
			tm.Delete(t.HostPort)
			fmt.Println("Tunnel closed")

			return nil
		},
	}

	return cmd
}
