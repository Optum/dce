package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	accountsCmd.AddCommand(listAccountCmd)
	accountsCmd.AddCommand(requestAccountCmd)
	accountsCmd.AddCommand(releaseAccountCmd)
	accountsCmd.AddCommand(describeAccountCmd)
	rootCmd.AddCommand(accountsCmd)
}

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage your dce accounts",
}

var listAccountCmd = &cobra.Command{
	Use:   "list",
	Short: "list accounts",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("List command")
	},
}

var describeAccountCmd = &cobra.Command{
	Use:   "describe",
	Short: "describe an account",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Describe command")
	},
}

var requestAccountCmd = &cobra.Command{
	Use:   "request",
	Short: "Request an account",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Request command")
	},
}

var releaseAccountCmd = &cobra.Command{
	Use:   "release",
	Short: "Release an account",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Release command")
	},
}
