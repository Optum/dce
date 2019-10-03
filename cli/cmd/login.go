package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to one of your DCE accounts",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("login command")
	},
}
