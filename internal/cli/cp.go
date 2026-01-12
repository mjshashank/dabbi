package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cp <source> <dest>",
		Short: "Copy files between host and VM",
		Long: `Copy files between the host and a VM.

Use vm_name:/path syntax for VM paths.

Examples:
  # Copy from host to VM
  dabbi cp ./local.txt my-vm:/home/ubuntu/remote.txt

  # Copy from VM to host
  dabbi cp my-vm:/home/ubuntu/remote.txt ./local.txt

  # Copy directories (multipass supports this)
  dabbi cp ./mydir my-vm:/home/ubuntu/`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, dst := args[0], args[1]

			fmt.Printf("Copying %s -> %s...\n", src, dst)
			if err := mpClient.Transfer(src, dst); err != nil {
				return err
			}
			fmt.Println("Copy complete")
			return nil
		},
	}
}
