package main

import (
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	eventMocks "github.com/Optum/dce/pkg/event/eventiface/mocks"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

// TestPopulateResetQeue tests and verifies the flow of adding all accounts
// provided into the reset queue and transition the finance lock if necessary
func TestPopulateResetQeue(t *testing.T) {
	tests := []struct {
		name         string
		expErr       error
		listAccounts *account.Accounts
		listErr      error
		alertErr     error
		nextId       *string
		status       *account.Status
	}{
		{
			name: "should send accounts to reset queue",
			listAccounts: &account.Accounts{
				{
					ID:               ptrString("123456789012"),
					Status:           account.StatusNotReady.StatusPtr(),
					AdminRoleArn:     arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
					PrincipalRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
				},
			},
			listErr: nil,
		},
		{
			name: "should send accounts to reset queue with pagination",
			listAccounts: &account.Accounts{
				{
					ID:               ptrString("123456789012"),
					Status:           account.StatusNotReady.StatusPtr(),
					AdminRoleArn:     arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
					PrincipalRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
				},
			},
			nextId:  ptrString("123456789013"),
			status:  account.StatusNotReady.StatusPtr(),
			listErr: nil,
		},
		{
			name: "should fail on list err",
			listAccounts: &account.Accounts{
				{
					ID:               ptrString("123456789012"),
					Status:           account.StatusNotReady.StatusPtr(),
					AdminRoleArn:     arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
					PrincipalRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
				},
			},
			listErr: errors.NewInternalServer("error", fmt.Errorf("error")),
			expErr:  errors.NewInternalServer("error", fmt.Errorf("error")),
		},
		{
			name: "should fail on alert err",
			listAccounts: &account.Accounts{
				{
					ID:               ptrString("123456789012"),
					Status:           account.StatusNotReady.StatusPtr(),
					AdminRoleArn:     arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
					PrincipalRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
				},
			},
			alertErr: errors.NewInternalServer("error", fmt.Errorf("error")),
			expErr: errors.NewMultiError("error when processing accounts", []error{
				errors.NewInternalServer("error", fmt.Errorf("error")),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			mocksRwd := &mocks.ReaderWriterDeleter{}

			mocksRwd.On("List", mock.MatchedBy(func(input *account.Account) bool {
				if input.Status.String() == "NotReady" {
					if input.NextID == nil {
						input.NextID = tt.nextId
						input.Status = tt.status

						return true
					}
				}
				return false
			})).Return(tt.listAccounts, tt.listErr)

			mocksRwd.On("List", mock.MatchedBy(func(input *account.Account) bool {
				if input.Status.String() == "NotReady" {
					if input.NextID == tt.nextId {
						input.NextID = nil
						return true
					}
				}
				return false
			})).Return(tt.listAccounts, tt.listErr)

			mocksEvent := &eventMocks.Servicer{}
			mocksEvent.On("AccountReset", mock.AnythingOfType("*account.Account")).
				Return(tt.alertErr)

			accountSvc := account.NewService(
				account.NewServiceInput{
					DataSvc:  mocksRwd,
					EventSvc: mocksEvent,
				},
			)

			svcBldr.Config.WithService(mocksEvent).WithService(accountSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				services = svcBldr
			}

			err = Handler(events.CloudWatchEvent{})
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
