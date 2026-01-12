package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <source> <new_name>",
		Short: "Clone a VM",
		Long: `Clone an existing VM to create a new instance.

This creates a deep copy of the VM including disk and state.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			dest := args[1]

			fmt.Printf("Cloning VM '%s' to '%s'...\n", source, dest)
			if err := mpClient.Clone(source, dest); err != nil {
				return err
			}
			fmt.Printf("VM '%s' cloned to '%s'\n", source, dest)
			return nil
		},
	}
}
