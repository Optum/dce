package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {

	adminAccountsCmd.AddCommand(adminAccountsRemoveCmd)
	adminAccountsCmd.AddCommand(adminAccountsAddCmd)
	adminAccountsCmd.AddCommand(adminAccountsDescribeCmd)
	adminAccountsCmd.AddCommand(adminAccountsListCmd)

	adminCmd.AddCommand(adminUpgradeCmd)
	adminCmd.AddCommand(adminInitCmd)
	adminCmd.AddCommand(adminLoginCmd)
	adminCmd.AddCommand(adminAccountsCmd)
	rootCmd.AddCommand(adminCmd)
}

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Administer DCE",
}

var adminAccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Administer all DCE accounts",
}

// Filter using Flags, e.g. --leased, --available, --budget-above, --budget-below, --budget, --include-attributes [all | status | budget | etc. ]
var adminAccountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DCE accounts based on filters.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("login command")
	},
}

var adminAccountsDescribeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe any DCE account.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("login command")
	},
}

var adminAccountsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add DCE account(s) to the accounts pool. Increases the total number of accounts.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("login command")
	},
}

var adminAccountsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a DCE account(s) from the accounts pull. Reduces the total number of accounts.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("remove account command")
	},
}

var adminLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to DCE master account",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("login command")
	},
}

var adminInitCmd = &cobra.Command{
	Use:   "init",
	Short: "First time initialization of DCE. Specify an account as master and deploy DCE to it.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("init command")
	},
}

var adminUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade DCE to the latest version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("upgrade command")
	},
}
