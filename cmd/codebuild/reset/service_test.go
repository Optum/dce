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
		"RESET_ACCOUNT_ADMIN_ROLE_NAME",
		"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME",
		"RESET_ACCOUNT_PRINCIPAL_POLICY_NAME",
		"RESET_NUKE_TOGGLE",
		"RESET_NUKE_TEMPLATE_DEFAULT",
		"RESET_NUKE_TEMPLATE_BUCKET",
		"RESET_NUKE_TEMPLATE_KEY",
	}
	for _, envKey := range envVars {
		_ = os.Setenv(envKey, envKey+"_VAL")
	}

	// Set toggle env vars
	_ = os.Setenv("RESET_NUKE_TOGGLE", "true")

	// Set regions env var
	_ = os.Setenv("RESET_NUKE_REGIONS", "us-east-1,us-west-1")

	t.Run("getConfig", func(t *testing.T) {

		t.Run("should configure from env vars", func(t *testing.T) {
			svc := &service{}
			// Make sure config isn't set elsewhere (global singleton)
			svc.setConfig(nil)
			config := svc.config()

			// Check configs from env vars
			require.Equal(t, "RESET_ACCOUNT_VAL", config.childAccountID)
			require.Equal(t, "RESET_ACCOUNT_ADMIN_ROLE_NAME_VAL", config.accountAdminRoleName)
			require.Equal(t, "RESET_ACCOUNT_PRINCIPAL_ROLE_NAME_VAL", config.accountPrincipalRoleName)
			require.Equal(t, "RESET_ACCOUNT_PRINCIPAL_POLICY_NAME_VAL", config.accountPrincipalPolicyName)
			require.Equal(t, "RESET_NUKE_TEMPLATE_DEFAULT_VAL", config.nukeTemplateDefault)
			require.Equal(t, "RESET_NUKE_TEMPLATE_BUCKET_VAL", config.nukeTemplateBucket)
			require.Equal(t, "RESET_NUKE_TEMPLATE_KEY_VAL", config.nukeTemplateKey)
			require.Equal(t, []string{"us-east-1", "us-west-1"}, config.nukeRegions)

			// Check computed config vals
			require.Equal(t, "arn:aws:iam::RESET_ACCOUNT_VAL:role/RESET_ACCOUNT_ADMIN_ROLE_NAME_VAL", config.accountAdminRoleARN)

			// Check toggle env vars
			require.Equal(t, true, config.isNukeEnabled)
		})

		t.Run("should be a singleton", func(t *testing.T) {
			svc := &service{}

			firstRes := svc.config()
			secondRes := svc.config()
			require.True(t, firstRes == secondRes)
		})

	})
}
