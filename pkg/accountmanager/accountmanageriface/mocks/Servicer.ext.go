package mocks

import (
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

// MockCredentials is a helper method to mock the credentials
// returned by AccountManager.Credentials()
func (_m *Servicer) MockCredentials(roleArn *arn.ARN, sessionName string, creds credentials.Value, credsError error) *Credentialer {
	mockCreds := &Credentialer{}
	mockCreds.
		On("Get").
		Return(creds, credsError)

	_m.
		On("Credentials", roleArn, sessionName).
		Return(mockCreds)

	return mockCreds
}
