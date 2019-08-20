package reset

import (
	"errors"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/rebuy-de/aws-nuke/cmd"
	"github.com/rebuy-de/aws-nuke/pkg/awsutil"
	"github.com/rebuy-de/aws-nuke/pkg/config"
)

// mockTokenService is a mocked implementation of TokenService
type mockTokenService struct {
	common.TokenService
}

// AssumeRole returns a mocked *sts.AssumeRoleOutput
func (token mockTokenService) AssumeRole(input *sts.AssumeRoleInput) (
	*sts.AssumeRoleOutput, error) {
	// Failure case
	if *input.RoleSessionName == "RedboxNukeTestAssumeRoleError" {
		return nil, errors.New("Error: Failed to Assume Role")
	}

	accessKeyID := "access"
	secretAccessKey := "secret"
	credentials := sts.Credentials{
		AccessKeyId:     &accessKeyID,
		Expiration:      nil,
		SecretAccessKey: &secretAccessKey,
		SessionToken:    input.RoleSessionName,
	}
	assumeRoleOutput := sts.AssumeRoleOutput{
		Credentials: &credentials,
	}
	return &assumeRoleOutput, nil
}

// NewCredentials returns nil and is not used
func (token mockTokenService) NewCredentials(inputClient client.ConfigProvider,
	inputRole string) *credentials.Credentials {
	return nil
}

// mockNukeService is a mocked implementation of mockNukeService
type mockNukeService struct{}

// NewAccount returns a mocked *awsutil.Account for testing
func (nuke mockNukeService) NewAccount(creds awsutil.Credentials) (
	*awsutil.Account, error) {
	// Failure case
	if creds.SessionToken == "RedboxNukeTestNewAccountError" {
		return nil, errors.New("Error: Failed to create a New Account")
	}

	account := awsutil.Account{
		Credentials: creds,
	}
	return &account, nil
}

// Load returns a mocked *config.Nuke for testing
func (nuke mockNukeService) Load(configPath string) (*config.Nuke, error) {
	// Failure case
	if configPath == "TestLoadError" {
		return nil, errors.New("Error: Failed to Load Configuration")
	}

	nukeConfig := config.Nuke{}
	return &nukeConfig, nil
}

// Run mocks an execution of a Nuke process
func (nuke mockNukeService) Run(cmd *cmd.Nuke) error {
	// Failure case
	if cmd.Account.Credentials.SessionToken == "RedboxNukeTestRunError" {
		return errors.New("Error: Failed to Run")
	}

	return nil
}

// testNukeAccountInput is the testing infrastructure used to test NukeAccount
type testNukeAccountInput struct {
	Input         *NukeAccountInput
	ExpectedError string
}

// TestNuke verifies that the Nuke works as intended with the provided Nuke
// Inputs and the TokenService and NukeService implementations
func TestNukeAccount(t *testing.T) {
	// Set up the list of tests to execute
	tests := []testNukeAccountInput{
		// Full nuke success
		{
			Input: &NukeAccountInput{
				AccountID:  "TestSuccess",
				RoleName:   "TestSuccess",
				ConfigPath: "TestSuccess",
				NoDryRun:   true,
				Token:      mockTokenService{},
				Nuke:       mockNukeService{},
			},
		},
		// AssumeRole error
		{
			Input: &NukeAccountInput{
				AccountID:  "TestAssumeRoleError",
				RoleName:   "TestAssumeRoleError",
				ConfigPath: "TestAssumeRoleError",
				NoDryRun:   true,
				Token:      mockTokenService{},
				Nuke:       mockNukeService{},
			},
			ExpectedError: "Error: Failed to Assume Role",
		},
		// NewAccount error
		{
			Input: &NukeAccountInput{
				AccountID:  "TestNewAccountError",
				RoleName:   "TestNewAccountError",
				ConfigPath: "TestNewAccountError",
				NoDryRun:   true,
				Token:      mockTokenService{},
				Nuke:       mockNukeService{},
			},
			ExpectedError: "Error: Failed to create a New Account",
		},
		// Load error
		{
			Input: &NukeAccountInput{
				AccountID:  "TestLoadError",
				RoleName:   "TestLoadError",
				ConfigPath: "TestLoadError",
				NoDryRun:   true,
				Token:      mockTokenService{},
				Nuke:       mockNukeService{},
			},
			ExpectedError: "Error: Failed to Load Configuration",
		},
		// Run error
		{
			Input: &NukeAccountInput{
				AccountID:  "TestRunError",
				RoleName:   "TestRunError",
				ConfigPath: "TestRunError",
				NoDryRun:   true,
				Token:      mockTokenService{},
				Nuke:       mockNukeService{},
			},
			ExpectedError: "Error: Failed to Run",
		},
	}

	// Iterate through each test in the list
	for _, test := range tests {
		// Call the NukeAccount function and get the respective error
		err := NukeAccount(test.Input)

		// Assert that error is expected correctly
		if test.ExpectedError == "" {
			require.Nil(t, err)
		} else {
			require.NotNil(t, err)
			require.Regexp(t, test.ExpectedError, err.Error())
		}
	}
}
