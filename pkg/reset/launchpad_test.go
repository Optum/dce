package reset

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockLaunchpader is a mocked implementation of Launchpader
type mockLaunchpader struct {
	mock.Mock
}

// Setup used for mock testing
func (mock mockLaunchpader) Setup(accountID string) error {
	args := mock.Called(accountID)
	err := args.Error(0)
	return err
}

// TriggerLaunchpad used for mock testing
func (mock mockLaunchpader) TriggerLaunchpad(accountID string,
	masterAccount string, bearerToken string) (string, error) {
	args := mock.Called(accountID, masterAccount, bearerToken)
	status := args.String(0)
	err := args.Error(1)
	return status, err
}

// Check used for mock testing
func (mock mockLaunchpader) CheckLaunchpad(accountID string, deployID string,
	bearerToken string) (string, error) {
	args := mock.Called(accountID, deployID)
	status := args.String(0)
	err := args.Error(1)
	return status, err
}

// Authenticate used for mock testing
func (mock mockLaunchpader) Authenticate() (string, error) {
	args := mock.Called()
	token := args.String(0)
	err := args.Error(1)
	return token, err
}

// testLaunchpadInput is the structure input used for table driven testing
// for LaunchpadAccount
type testLaunchpadAccountInput struct {
	Error              error
	AuthenticateToken  string
	AuthenticateError  error
	TriggerID          string
	TriggerError       error
	CheckStatus        string
	CheckError         error
	ExpectAuthenticate bool
	ExpectTrigger      bool
	ExpectCheck        bool
}

// TestLaunchpadAccount verifies the flow LaunchpadAccount correctly follows
// setting up, triggering, and retrieving the status of Launchpad being applied
// to an account
func TestLaunchpadAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testLaunchpadAccountInput{
		// Happy Path
		{
			Error:              nil,
			TriggerID:          "123",
			TriggerError:       nil,
			CheckStatus:        "SUCCESS",
			CheckError:         nil,
			AuthenticateToken:  "abcdef",
			AuthenticateError:  nil,
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
			ExpectCheck:        true,
		},
		// Authenticate Fail
		{
			Error:              errors.New("Error : Failed to authenticate - Failed to make request"),
			AuthenticateError:  errors.New("Failed to make request"),
			ExpectAuthenticate: true,
		},
		// TriggerLaunchpad Fail
		{
			Error:              errors.New("Error : Couldn't Deploy Launchpad to 123456789012 - Failed to make request"),
			TriggerID:          "",
			TriggerError:       errors.New("Failed to make request"),
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
		},
		// Check Fail - This will take ~5 seconds.
		{
			Error:              errors.New("Error : Failed Deploying Launchpad to 123456789012 - Failed to make request"),
			TriggerID:          "123",
			TriggerError:       nil,
			CheckStatus:        "",
			CheckError:         errors.New("Failed to make request"),
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
			ExpectCheck:        true,
		},
		// Aborted Build
		{
			Error:              errors.New("Error : Failed Deploying Launchpad to 123456789012 - ABORTED Launchpad Build 123"),
			TriggerID:          "123",
			TriggerError:       nil,
			CheckStatus:        "ABORTED",
			CheckError:         nil,
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
			ExpectCheck:        true,
		},
		// Failure Build
		{
			Error:              errors.New("Error : Failed Deploying Launchpad to 123456789012 - FAILURE Launchpad Build 123"),
			TriggerID:          "123",
			TriggerError:       nil,
			CheckStatus:        "FAILURE",
			CheckError:         nil,
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
			ExpectCheck:        true,
		},
		// Unstable Build
		{
			Error:              errors.New("Error : Failed Deploying Launchpad to 123456789012 - UNSTABLE Launchpad Build 123"),
			TriggerID:          "123",
			TriggerError:       nil,
			CheckStatus:        "UNSTABLE",
			CheckError:         nil,
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
			ExpectCheck:        true,
		},
		// Unknown Build
		{
			Error:              errors.New("Error : Unknown Status Deploying Launchpad to 123456789012 - UNKNOWN Launchpad Build 123"),
			TriggerID:          "123",
			TriggerError:       nil,
			CheckStatus:        "UNKNOWN",
			CheckError:         nil,
			ExpectAuthenticate: true,
			ExpectTrigger:      true,
			ExpectCheck:        true,
		},
	}

	// Iterate through each test in the list
	account := "123456789012"
	for _, test := range tests {
		// Setup mocks
		launchpad := mockLaunchpader{}
		if test.ExpectAuthenticate {
			launchpad.On("Authenticate").Return(test.AuthenticateToken,
				test.AuthenticateError)
		}
		if test.ExpectTrigger {
			launchpad.On("TriggerLaunchpad", mock.Anything, mock.Anything,
				mock.Anything).Return(test.TriggerID, test.TriggerError)
		}
		if test.ExpectCheck {
			launchpad.On("CheckLaunchpad", mock.Anything, mock.Anything).Return(
				test.CheckStatus, test.CheckError)
		}

		// Create LaunchpadAccountInput
		input := LaunchpadAccountInput{
			Launchpad:   launchpad,
			AccountID:   account,
			WaitSeconds: 1, // Only wait 1 second per retry
		}

		// Execute LaunchpadAccount
		err := LaunchpadAccount(&input)
		launchpad.AssertExpectations(t)

		// Assert that the expected output is correct
		require.Equal(t, test.Error, err)
	}
}
