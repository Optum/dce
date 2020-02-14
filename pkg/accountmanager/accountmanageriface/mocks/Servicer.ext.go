package mocks

import (
	"github.com/Optum/dce/pkg/accountmanager/accountmanageriface"
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/stretchr/testify/mock"
	"time"
)

// MockCredentials is a helper method to mock the credentials
// returned by AccountManager.Credentials()
func (_m *Servicer) MockCredentials(
	roleArn *arn.ARN,
	sessionName string,
	creds credentials.Value,
	credsError error,
) *Credentialer {
	// Mock the credentials object
	mockCreds := &Credentialer{}
	mockCreds.
		On("Get").
		Return(creds, credsError)

	// Mock the Service to return our mocked credentials
	_m.
		On("Credentials", roleArn, sessionName, mock.Anything).
		Return(func(role *arn.ARN, roleSessionName string, duration *time.Duration) accountmanageriface.Credentialer {
			// Default duration is 15 min
			if duration == nil {
				defaultDuration := time.Minute * 15
				duration = &defaultDuration
			}

			// Mock Credentials.ExpiresAt(), based on the provided duration
			mockCreds.
				On("ExpiresAt").
				Return(time.Now().Add(*duration), nil)

			return mockCreds
		})

	return mockCreds
}
