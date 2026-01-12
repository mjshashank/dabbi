package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage VM snapshots",
		Long: `Manage VM snapshots.

Note: VMs must be stopped before creating snapshots.`,
		Aliases: []string{"snap"},
	}

	cmd.AddCommand(
		newSnapshotListCmd(),
		newSnapshotCreateCmd(),
		newSnapshotRestoreCmd(),
		newSnapshotDeleteCmd(),
	)

	return cmd
}

func newSnapshotListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <vm_name>",
		Short:   "List snapshots for a VM",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]
			snapshots, err := mpClient.ListSnapshots(vmName)
			if err != nil {
				return err
			}

			if len(snapshots) == 0 {
				fmt.Printf("No snapshots for VM '%s'\n", vmName)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tPARENT\tCOMMENT")
			fmt.Fprintln(w, "----\t------\t-------")

			for name, snap := range snapshots {
				parent := snap.Parent
				if parent == "" {
					parent = "(base)"
				}
				comment := snap.Comment
				if comment == "" {
					comment = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", name, parent, comment)
			}

			return w.Flush()
		},
	}
}

func newSnapshotCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <vm_name> [snapshot_name]",
		Short: "Create a snapshot",
		Long: `Create a snapshot of a VM.

The VM must be stopped before creating a snapshot.
If no name is provided, multipass will generate one.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]
			snapshotName := ""
			if len(args) > 1 {
				snapshotName = args[1]
			}

			fmt.Printf("Creating snapshot for VM '%s'...\n", vmName)
			if err := mpClient.CreateSnapshot(vmName, snapshotName); err != nil {
				return err
			}
			fmt.Println("Snapshot created")
			return nil
		},
	}
}

func newSnapshotRestoreCmd() *cobra.Command {
	var destructive bool

	cmd := &cobra.Command{
		Use:   "restore <vm_name> <snapshot_name>",
		Short: "Restore a snapshot",
		Long:  `Restore a VM to a previous snapshot state.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]
			snapshotName := args[1]

			fmt.Printf("Restoring snapshot '%s' for VM '%s'...\n", snapshotName, vmName)
			if err := mpClient.RestoreSnapshot(vmName, snapshotName, destructive); err != nil {
				return err
			}
			fmt.Println("Snapshot restored")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&destructive, "destructive", "d", false, "Discard current VM state without confirmation")

	return cmd
}

func newSnapshotDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <vm_name> <snapshot_name>",
		Short:   "Delete a snapshot",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmName := args[0]
			snapshotName := args[1]

			fmt.Printf("Deleting snapshot '%s' for VM '%s'...\n", snapshotName, vmName)
			if err := mpClient.DeleteSnapshot(vmName, snapshotName); err != nil {
				return err
			}
			fmt.Println("Snapshot deleted")
			return nil
		},
	}
}
