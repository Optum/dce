package mocks

import (
	awsMocks "github.com/Optum/Redbox/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/stretchr/testify/mock"
)

func (_m *TokenService) MockNewSession(expectedRoleArn string) *awsMocks.AwsSession {
	mockAssumedSession := &awsMocks.AwsSession{}
	mockAssumedSession.
		On("ClientConfig", mock.Anything).
		Return(client.Config{
			Config: &aws.Config{},
		})

	_m.
		On("NewSession", mock.Anything, expectedRoleArn).
		Return(mockAssumedSession, nil)

	return mockAssumedSession
}
