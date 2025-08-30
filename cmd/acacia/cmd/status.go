package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the running Acacia server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement the logic to connect to the running server
		// and fetch its status. This might involve IPC, like a Unix socket
		// or reading a status file.
		fmt.Println("Status command is not yet implemented.")
		fmt.Println("It will show the status of loaded modules and gateways.")
		return nil
	},
}
