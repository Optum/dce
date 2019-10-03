package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configureCmd)
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure DCE cli",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Master Account .aws Credentials Profile: " + *config.Admin.MasterAccount.Profile)
	},
}
