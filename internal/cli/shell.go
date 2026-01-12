package cli

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell <vm_name>",
		Short: "Open interactive shell in VM",
		Long: `Open an interactive shell session in the specified VM.

This directly executes 'multipass shell' for native performance.
The VM will be started automatically if it's stopped.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			// Find multipass binary
			multipassPath, err := exec.LookPath("multipass")
			if err != nil {
				return err
			}

			// Direct exec to multipass shell for native performance
			// This replaces the current process
			return syscall.Exec(multipassPath, []string{"multipass", "shell", vmName}, os.Environ())
		},
	}
}
