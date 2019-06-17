package main

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestService(t *testing.T) {

	// Stub required env vars
	envVars := []string{
		"RESET_ROLE",
		"RESET_ACCOUNT",
		"RESET_TEMPLATE",
		"RESET_NUKE_TOGGLE",
		"RESET_LAUNCHPAD_TOGGLE",
		"RESET_LAUNCHPAD_BASE_ENDPOINT",
		"RESET_LAUNCHPAD_AUTH_ENDPOINT",
		"RESET_LAUNCHPAD_MASTER_ACCOUNT",
		"RESET_LAUNCHPAD_BACKEND",
	}
	for _, envKey := range envVars {
		err := os.Setenv(envKey, envKey+"_VAL")
		require.Nil(t, err)
	}

	// Set toggle env vars
	err := os.Setenv("RESET_NUKE_TOGGLE", "true")
	require.Nil(t, err)
	err = os.Setenv("RESET_LAUNCHPAD_TOGGLE", "false")
	require.Nil(t, err)
	t.Run("getConfig", func(t *testing.T) {

		t.Run("should configure from env vars", func(t *testing.T) {
			svc := &service{}
			config := svc.config()

			// Check configs from env vars
			require.Equal(t, "RESET_ACCOUNT_VAL", config.accountID)
			require.Equal(t, "RESET_ROLE_VAL", config.customerRoleName)
			require.Equal(t, "RESET_TEMPLATE_VAL", config.nukeTemplate)
			require.Equal(t, "RESET_LAUNCHPAD_BASE_ENDPOINT_VAL", config.launchpadBaseEndpoint)
			require.Equal(t, "RESET_LAUNCHPAD_AUTH_ENDPOINT_VAL", config.launchpadAuthEndpoint)
			require.Equal(t, "RESET_LAUNCHPAD_MASTER_ACCOUNT_VAL", config.launchpadMasterAccount)
			require.Equal(t, "RESET_LAUNCHPAD_BACKEND_VAL", config.launchpadBackend)

			// Check computed config vals
			require.Equal(t, "arn:aws:iam::RESET_ACCOUNT_VAL:role/RESET_ROLE_VAL", config.customerRoleArn)

			// Check toggle env vars
			require.Equal(t, true, config.isNukeEnabled)
			require.Equal(t, false, config.isLaunchpadEnabled)
		})

		t.Run("should be a singleton", func(t *testing.T) {
			svc := &service{}

			require.True(t, svc.config() == svc.config())
		})

	})
}
