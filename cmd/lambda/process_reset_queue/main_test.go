package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestPopulateResetQeue tests and verifies the flow of adding all accounts
// provided into the reset queue and transition the finance lock if necessary
func TestProcessResetQueue(t *testing.T) {
	tests := []struct {
		name         string
		input        events.SQSEvent
		expErr       error
		codeBuildErr error
	}{
		{
			name: "should send account to code build",
			input: events.SQSEvent{
				Records: []events.SQSMessage{
					{
						Body: "{\"id\":\"123456789012\",\"adminRoleArn\":\"arn:aws:iam::123456789012:role/AdminRole\",\"principalRoleArn\":\"arn:aws:iam::123456789012:role/PrincipalRole\",\"leaseStatus\":\"Active\",\"status\":\"NotReady\"}\n",
					},
				},
			},
		},
		{
			name: "should fail on parse err",
			input: events.SQSEvent{
				Records: []events.SQSMessage{
					{
						Body: "{\"id\":123456789012\",\"adminRoleArn\":\"arn:aws:iam::123456789012:role/AdminRole\",\"principalRoleArn\":\"arn:aws:iam::123456789012:role/PrincipalRole\",\"leaseStatus\":\"Active\",\"status\":\"NotReady\"}\n",
					},
				},
			},
			expErr: errors.NewInternalServer("unexpected error unmarshaling sqs message", fmt.Errorf("invalid character '\"' after object key:value pair")),
		},
		{
			name: "should fail on codebuild err",
			input: events.SQSEvent{
				Records: []events.SQSMessage{
					{
						Body: "{\"id\":\"123456789012\",\"adminRoleArn\":\"arn:aws:iam::123456789012:role/AdminRole\",\"principalRoleArn\":\"arn:aws:iam::123456789012:role/PrincipalRole\",\"leaseStatus\":\"Active\",\"status\":\"NotReady\"}\n",
					},
				},
			},
			codeBuildErr: fmt.Errorf("error"),
			expErr:       errors.NewInternalServer("unexpected error starting code build", fmt.Errorf("error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			mocksCodeBuild := &mocks.CodeBuildAPI{}
			mocksCodeBuild.On("StartBuild", mock.Anything).Return(nil, tt.codeBuildErr)
			svcBldr.Config.WithService(mocksCodeBuild)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				services = svcBldr
			}

			err = handler(context.TODO(), tt.input)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
