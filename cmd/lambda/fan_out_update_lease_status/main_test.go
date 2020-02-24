package main

import (
	"fmt"
	"log"
	"testing"

	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/data/dataiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	lambdaSDK "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func TestLambdaHandler(t *testing.T) {
	type lambdaInvoke struct {
		input *lambdaSDK.InvokeInput
		err   error
	}
	tests := []struct {
		name         string
		retLeases    *lease.Leases
		retLeasesErr error
		retLambda    []lambdaInvoke
		expErr       error
	}{
		{
			name: "when given good leases. Everything works",
			retLeases: &lease.Leases{
				{
					ID:          ptrString("abc-123"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("TestUser1"),
				},
				{
					ID:          ptrString("def-456"),
					AccountID:   ptrString("123456789013"),
					PrincipalID: ptrString("TestUser2"),
				},
			},
			retLambda: []lambdaInvoke{
				{
					input: &lambdaSDK.InvokeInput{
						FunctionName:   aws.String("UpdateLeaseStatusFunction"),
						InvocationType: aws.String("Event"),
						Payload:        []byte("{\"accountId\":\"123456789012\",\"principalId\":\"TestUser1\",\"id\":\"abc-123\"}"),
					},
				},
				{
					input: &lambdaSDK.InvokeInput{
						FunctionName:   aws.String("UpdateLeaseStatusFunction"),
						InvocationType: aws.String("Event"),
						Payload:        []byte("{\"accountId\":\"123456789013\",\"principalId\":\"TestUser2\",\"id\":\"def-456\"}"),
					},
				},
			},
		},
		{
			name:         "when getting leases fails return the failure",
			retLeases:    nil,
			retLeasesErr: errors.NewInternalServer("failure", fmt.Errorf("error")),
			retLambda:    []lambdaInvoke{},
			expErr:       errors.NewInternalServer("failure", fmt.Errorf("error")),
		},
		{
			name: "when given good leases. Lambda execution failure is returned and execution continues.",
			retLeases: &lease.Leases{
				{
					ID:          ptrString("abc-123"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("TestUser1"),
				},
				{
					ID:          ptrString("def-456"),
					AccountID:   ptrString("123456789013"),
					PrincipalID: ptrString("TestUser2"),
				},
				{
					ID:          ptrString("ghi-789"),
					AccountID:   ptrString("123456789014"),
					PrincipalID: ptrString("TestUser3"),
				},
			},
			retLambda: []lambdaInvoke{
				{
					input: &lambdaSDK.InvokeInput{
						FunctionName:   aws.String("UpdateLeaseStatusFunction"),
						InvocationType: aws.String("Event"),
						Payload:        []byte("{\"accountId\":\"123456789012\",\"principalId\":\"TestUser1\",\"id\":\"abc-123\"}"),
					},
				},
				{
					input: &lambdaSDK.InvokeInput{
						FunctionName:   aws.String("UpdateLeaseStatusFunction"),
						InvocationType: aws.String("Event"),
						Payload:        []byte("{\"accountId\":\"123456789013\",\"principalId\":\"TestUser2\",\"id\":\"def-456\"}"),
					},
					err: fmt.Errorf("error"),
				},
				{
					input: &lambdaSDK.InvokeInput{
						FunctionName:   aws.String("UpdateLeaseStatusFunction"),
						InvocationType: aws.String("Event"),
						Payload:        []byte("{\"accountId\":\"123456789014\",\"principalId\":\"TestUser3\",\"id\":\"ghi-789\"}"),
					},
				},
			},
			expErr: errors.NewMultiError("error when processing accounts",
				[]error{
					fmt.Errorf("error"),
				}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			dataSvc := mocks.LeaseData{}
			dataSvc.On("List", &lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			}).Return(tt.retLeases, tt.retLeasesErr)
			lambdaSvc := awsMocks.LambdaAPI{}
			for _, m := range tt.retLambda {
				lambdaSvc.On("Invoke", m.input).Return(nil, m.err)
			}

			leaseSvc := lease.NewService(lease.NewServiceInput{
				DataSvc: &dataSvc,
			})

			svcBldr.Config.WithService(leaseSvc).WithService(&lambdaSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				services = svcBldr
			}

			err = handler(events.CloudWatchEvent{})

			lambdaSvc.AssertExpectations(t)

			if err != nil {
				log.Printf("%+s", err.Error())
			}

			assert.True(t, errors.Is(err, tt.expErr))
		})
	}
}
