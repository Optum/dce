package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {

	// Stub required env vars
	envVars := []string{
		"RESET_ROLE",
		"RESET_ACCOUNT",
		"RESET_ACCOUNT_ADMIN_ROLE",
		"RESET_ACCOUNT_USER_ROLE",
		"RESET_NUKE_TOGGLE",
		"RESET_NUKE_TEMPLATE_DEFAULT",
		"RESET_NUKE_TEMPLATE_BUCKET",
		"RESET_NUKE_TEMPLATE_KEY",
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
			require.Equal(t, "RESET_ACCOUNT_ADMIN_ROLE_VAL", config.accountAdminRoleName)
			require.Equal(t, "RESET_ACCOUNT_USER_ROLE_VAL", config.accountUserRoleName)
			require.Equal(t, "RESET_NUKE_TEMPLATE_DEFAULT_VAL", config.nukeTemplateDefault)
			require.Equal(t, "RESET_NUKE_TEMPLATE_BUCKET_VAL", config.nukeTemplateBucket)
			require.Equal(t, "RESET_NUKE_TEMPLATE_KEY_VAL", config.nukeTemplateKey)
			require.Equal(t, "RESET_LAUNCHPAD_BASE_ENDPOINT_VAL", config.launchpadBaseEndpoint)
			require.Equal(t, "RESET_LAUNCHPAD_AUTH_ENDPOINT_VAL", config.launchpadAuthEndpoint)
			require.Equal(t, "RESET_LAUNCHPAD_MASTER_ACCOUNT_VAL", config.launchpadMasterAccount)
			require.Equal(t, "RESET_LAUNCHPAD_BACKEND_VAL", config.launchpadBackend)

			// Check computed config vals
			require.Equal(t, "arn:aws:iam::RESET_ACCOUNT_VAL:role/RESET_ACCOUNT_ADMIN_ROLE_VAL", config.accountAdminRoleARN)

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
