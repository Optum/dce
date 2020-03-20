package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-lambda-go/events"
)

func TestUpdatePrincipalPolicy(t *testing.T) {

	tests := []struct {
		name      string
		acctID    string
		input     events.SNSEvent
		getAcct   *account.Account
		getErr    error
		upsertErr error
		expErr    error
	}{
		{
			name:   "when valid lease provided upsert happens",
			acctID: "123456789012",
			input: events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						SNS: events.SNSEntity{
							Message: "{\"accountId\": \"123456789012\"}",
						},
					},
				},
			},
		},
		{
			name: "when invalid lease provided an error occurs",
			input: events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						SNS: events.SNSEntity{
							Message: "{\"accountId\", \"123456789012\"}",
						},
					},
				},
			},
			expErr: errors.NewInternalServer("unexpected error parsing SNS message", fmt.Errorf("invalid character ',' after object key")),
		},
		{
			name:   "when valid lease provided but an account is not found return error",
			acctID: "123456789012",
			input: events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						SNS: events.SNSEntity{
							Message: "{\"accountId\": \"123456789012\"}",
						},
					},
				},
			},
			getErr: errors.NewNotFound("account", "123456789012"),
			expErr: errors.NewNotFound("account", "123456789012"),
		},
		{
			name:   "when valid lease provided but there is an error upserting the policy",
			acctID: "123456789012",
			input: events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						SNS: events.SNSEntity{
							Message: "{\"accountId\": \"123456789012\"}",
						},
					},
				},
			},
			upsertErr: errors.NewInternalServer("failure", fmt.Errorf("error")),
			expErr:    errors.NewInternalServer("failure", fmt.Errorf("error")),
		},
	}

	// Iterate through each test in the list
	for _, tt := range tests {
		cfgBldr := &config.ConfigurationBuilder{}
		svcBldr := &config.ServiceBuilder{Config: cfgBldr}
		// Setup mocks

		acctServiceMock := mocks.Servicer{}
		acctServiceMock.On("Get", tt.acctID).Return(tt.getAcct, tt.getErr)
		acctServiceMock.On("UpsertPrincipalAccess", tt.getAcct).Return(tt.upsertErr)

		svcBldr.Config.WithService(&acctServiceMock)
		_, err := svcBldr.Build()
		assert.Nil(t, err)
		if err == nil {
			services = svcBldr
		}

		err = handler(context.TODO(), tt.input)
		assert.True(t, errors.Is(err, tt.expErr))
	}
}
