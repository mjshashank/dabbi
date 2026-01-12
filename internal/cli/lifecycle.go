package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a stopped VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Starting VM '%s'...\n", name)
			if err := mpClient.Start(name); err != nil {
				return err
			}
			fmt.Printf("VM '%s' started\n", name)
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Stopping VM '%s'...\n", name)
			if err := mpClient.Stop(name); err != nil {
				return err
			}
			fmt.Printf("VM '%s' stopped\n", name)
			return nil
		},
	}
}

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Restarting VM '%s'...\n", name)
			if err := mpClient.Restart(name); err != nil {
				return err
			}
			fmt.Printf("VM '%s' restarted\n", name)
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	var keepRecoverable bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a VM",
		Long: `Delete a VM permanently.

Use --keep-recoverable to allow recovery with 'multipass recover'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Deleting VM '%s'...\n", name)
			if err := mpClient.Delete(name, !keepRecoverable); err != nil {
				return err
			}
			fmt.Printf("VM '%s' deleted\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&keepRecoverable, "keep-recoverable", false, "Keep VM recoverable (don't purge)")

	return cmd
}
