package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount",
		Short: "Manage VM mounts",
		Long: `Mount or unmount host directories to VMs.

Mounts persist across VM reboots (managed by multipass).`,
	}

	cmd.AddCommand(
		newMountAddCmd(),
		newMountRemoveCmd(),
		newMountListCmd(),
	)

	return cmd
}

func newMountAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <vm_name> <host_path> <vm_path>",
		Short: "Mount host directory to VM",
		Long: `Mount a host directory into a VM.

Example:
  dabbi mount add my-vm /home/user/projects /home/ubuntu/projects`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName, hostPath, vmPath := args[0], args[1], args[2]

			fmt.Printf("Mounting %s -> %s:%s...\n", hostPath, vmName, vmPath)
			if err := mpClient.Mount(vmName, hostPath, vmPath); err != nil {
				return err
			}
			fmt.Println("Mount added")
			return nil
		},
	}
}

func newMountRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <vm_name> <vm_path>",
		Short:   "Remove mount from VM",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName, vmPath := args[0], args[1]

			fmt.Printf("Unmounting %s:%s...\n", vmName, vmPath)
			if err := mpClient.Unmount(vmName, vmPath); err != nil {
				return err
			}
			fmt.Println("Mount removed")
			return nil
		},
	}
}

func newMountListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <vm_name>",
		Short:   "List mounts for a VM",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]

			info, err := mpClient.Info(vmName)
			if err != nil {
				return err
			}

			if len(info.Mounts) == 0 {
				fmt.Printf("No mounts for VM '%s'\n", vmName)
				return nil
			}

			fmt.Printf("Mounts for VM '%s':\n", vmName)
			for vmPath, mount := range info.Mounts {
				fmt.Printf("  %s -> %s\n", mount.SourcePath, vmPath)
			}
			return nil
		},
	}
}
