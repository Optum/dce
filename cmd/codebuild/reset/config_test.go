package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {

	// Stub required env vars
	envVars := []string{
		"RESET_ROLE",
		"RESET_ACCOUNT",
		"RESET_ACCOUNT_ADMIN_ROLE_NAME",
		"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME",
		"RESET_ACCOUNT_PRINCIPAL_POLICY_NAME",
		"RESET_NUKE_TOGGLE",
		"RESET_NUKE_TEMPLATE_DEFAULT",
		"RESET_NUKE_TEMPLATE_BUCKET",
		"RESET_NUKE_TEMPLATE_KEY",
	}
	for _, envKey := range envVars {
		err := os.Setenv(envKey, envKey+"_VAL")
		require.Nil(t, err)
	}

	// Set toggle env vars
	err := os.Setenv("RESET_NUKE_TOGGLE", "true")
	require.Nil(t, err)

	// Required env vars
	_ = os.Setenv("AWS_CURRENT_REGION", "us-east-1")
	_ = os.Setenv("ACCOUNT_DB", "AccountsTest")
	_ = os.Setenv("LEASE_DB", "LeasesTest")

	t.Run("initConfig", func(t *testing.T) {

		t.Run("should configure from env vars", func(t *testing.T) {
			config, err := initConfig()
			require.Nil(t, err)

			// Check configs from env vars
			require.Equal(t, "RESET_ACCOUNT_VAL", config.AccountID)
			require.Equal(t, "RESET_ACCOUNT_ADMIN_ROLE_NAME_VAL", config.AdminRoleName)
			require.Equal(t, "RESET_ACCOUNT_PRINCIPAL_ROLE_NAME_VAL", config.PrincipalRoleName)
			require.Equal(t, "RESET_ACCOUNT_PRINCIPAL_POLICY_NAME_VAL", config.PrincipalPolicyName)
			require.Equal(t, "RESET_NUKE_TEMPLATE_DEFAULT_VAL", config.NukeTemplateDefault)
			require.Equal(t, "RESET_NUKE_TEMPLATE_BUCKET_VAL", config.NukeTemplateBucket)
			require.Equal(t, "RESET_NUKE_TEMPLATE_KEY_VAL", config.NukeTemplateKey)

			// Check computed config vals
			require.Equal(t, "arn:aws:iam::RESET_ACCOUNT_VAL:role/RESET_ACCOUNT_ADMIN_ROLE_NAME_VAL", config.AdminRoleARN())

			// Check toggle env vars
			require.Equal(t, true, config.IsNukeEnabled)
		})

		t.Run("should configure services", func(t *testing.T) {
			config, err := initConfig()
			require.Nil(t, err)

			require.NotNil(t, config.Session)
			require.NotNil(t, config.TokenService)
			require.NotNil(t, config.S3)
			require.NotNil(t, config.DB)
			require.NotNil(t, config.SNS)
		})

	})
}
