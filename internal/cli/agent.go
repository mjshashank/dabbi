package cli

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent <vm_name>",
		Short: "Open interactive OpenCode session in VM",
		Long: `Open an interactive OpenCode CLI session in the specified VM.

This directly executes 'multipass exec <vm> -- opencode' for native performance.
The VM will be started automatically if it's stopped.

Example:
  dabbi agent my-dev-vm`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			// Find multipass binary
			multipassPath, err := exec.LookPath("multipass")
			if err != nil {
				return err
			}

			// Direct exec to multipass for native performance
			// This replaces the current process
			// Use full path since ~/.opencode/bin is not in PATH for non-login shells
			return syscall.Exec(multipassPath, []string{
				"multipass", "exec", vmName, "--", "/home/ubuntu/.opencode/bin/opencode",
			}, os.Environ())
		},
	}
}
