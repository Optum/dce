package reset

import (
	"github.com/rebuy-de/aws-nuke/cmd"
	"github.com/rebuy-de/aws-nuke/pkg/awsutil"
	"github.com/rebuy-de/aws-nuke/pkg/config"
)

// Nuker interface requires methods that are necessary to set up and
// execute a Nuke in an AWS Account.
type Nuker interface {
	NewAccount(awsutil.Credentials) (*awsutil.Account, error)
	Load(string) (*config.Nuke, error)
	Run(*cmd.Nuke) error
}

// Nuke implements the NukeService interface using rebuy-de/aws-nuke
// https://github.com/rebuy-de/aws-nuke
type Nuke struct {
}

// NewAccount returns an aws-nuke Account that is created from the provided
// aws-nuke Credentials. This will provide the account information needed for
// aws-nuke to access an account to nuke.
func (nuke Nuke) NewAccount(creds awsutil.Credentials) (*awsutil.Account,
	error) {
	return awsutil.NewAccount(creds)
}

// Load returns an aws-nuke Nuke configuration with the provided configuration
// file. This will provide the information needed to know what can be nuked by
// aws-nuke.
func (nuke Nuke) Load(configPath string) (*config.Nuke, error) {
	return config.Load(configPath)
}

// Run executes and returns the result of the aws-nuke nuke.
func (nuke Nuke) Run(cmd *cmd.Nuke) error {
	return cmd.Run()
}
