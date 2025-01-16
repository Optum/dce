package reset

import (
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/Optum/dce/pkg/common"

	nukeconfig "github.com/ekristen/aws-nuke/v3/pkg/config"

	libconfig "github.com/ekristen/libnuke/pkg/config"
	"github.com/ekristen/libnuke/pkg/registry"
)

// NukeAccountInput is the container used for the TokenService and the
// NukeService to execute a Nuke for an AWS Account
type NukeAccountInput struct {
	ChildAccountID string
	RoleName       string
	ConfigPath     string
	Token          common.TokenService
}

// NukeAccount directly triggers aws-nuke to be called on the
// configuration file provided, bypassing any manual prompts.
// Returns an error if there's any, else nil.
func NukeAccount(input *NukeAccountInput) error {
	logger := logrus.StandardLogger()

	// Create a NukeParameter based on the configuration file
	// path and force to bypass prompts.
	parsedConfig, err := nukeconfig.New(libconfig.Options{
		Path:         input.ConfigPath,
		Deprecations: registry.GetDeprecatedResourceTypeMapping(),
		Log:          logger.WithField("component", "config"),
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to parse config for account %s for aws-nuke with file %s",
			input.ChildAccountID, input.ConfigPath)
	}

	nuker, err := NewNuker(parsedConfig, input)
	if err != nil {
		return errors.Wrapf(err, "Failed to configure nuke for account %s for aws-nuke as %s",
			input.ChildAccountID, input.RoleName)
	}

	c := make(chan error, 1)
	go func() { c <- nuker.Run() }()
	select {
	case err := <-c:
		if err != nil {
			return errors.Wrapf(err, "Failed to run nuke for account %s as %s",
				input.ChildAccountID, input.RoleName)
		}
		return nil
	case <-time.After(time.Minute * 60):
		return errors.New("Nuke Timed Out after 60 minutes")
	}
}
