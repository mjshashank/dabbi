package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all VMs",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			vms, err := mpClient.List()
			if err != nil {
				return err
			}

			if len(vms) == 0 {
				fmt.Println("No VMs found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATE\tIPV4\tRELEASE")
			fmt.Fprintln(w, "----\t-----\t----\t-------")

			for _, vm := range vms {
				ipv4 := "-"
				if len(vm.IPv4) > 0 && vm.IPv4[0] != "" {
					ipv4 = vm.IPv4[0]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", vm.Name, vm.State, ipv4, vm.Release)
			}

			return w.Flush()
		},
	}
}
