package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	commonMocks "github.com/Optum/Dce/pkg/common/mocks"
	"github.com/Optum/Dce/pkg/db"
	"github.com/Optum/Dce/pkg/db/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResetPipeline(t *testing.T) {
	t.Run("updateDBPostReset", func(t *testing.T) {

		t.Run("Should change any ResetLock leases to Active", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			snsSvc := &commonMocks.Notificationer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.DceLease{
					{
						AccountID:   "111",
						PrincipalID: "222",
						LeaseStatus: db.ResetLock,
					},
					// Should not change the status of decommissioned leases
					{
						AccountID:   "111",
						PrincipalID: "222",
						LeaseStatus: db.Decommissioned,
					},
				}, nil)

			// Mock Lease status change
			dbSvc.
				On("TransitionLeaseStatus", "111", "222", db.ResetLock, db.Active).
				Return(&db.DceLease{}, nil)

			// Mock Account status change
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.DceAccount{}, nil)

			snsSvc.On("PublishMessage",
				mock.MatchedBy(func(arn *string) bool {
					return *arn == "Topic"
				}),
				mock.MatchedBy(func(message *string) bool {
					// Parse the message JSON
					messageObj := unmarshal(t, *message)
					// `default` and `body` and JSON embedded within the message JSON
					msgDefault := unmarshal(t, messageObj["default"].(string))
					msgBody := unmarshal(t, messageObj["Body"].(string))

					assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

					// Check that we're sending the account object
					assert.Equal(t, "", msgBody["Id"])

					return true
				}), true,
			).Return(aws.String("mock message"), nil)
			defer snsSvc.AssertExpectations(t)

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should change any ResetFinanceLock leases to FinanceLock", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			snsSvc := &commonMocks.Notificationer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.DceLease{
					// Should call TransitionLeaseStatus for this ResetFinanceLock lease status
					{
						AccountID:   "111",
						PrincipalID: "222",
						LeaseStatus: db.ResetFinanceLock,
					},
					// Should not call TransitionLeaseStatus for this decommisioned lease status
					{
						AccountID:   "111",
						PrincipalID: "333",
						LeaseStatus: db.Decommissioned,
					},
				}, nil)

			// Mock Lease status change
			dbSvc.
				On("TransitionLeaseStatus", "111", "222", db.ResetFinanceLock, db.FinanceLock).
				Return(&db.DceLease{}, nil)

			// Mock Account status change
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.DceAccount{}, nil)

			snsSvc.On("PublishMessage",
				mock.MatchedBy(func(arn *string) bool {
					return *arn == "Topic"
				}),
				mock.MatchedBy(func(message *string) bool {
					// Parse the message JSON
					messageObj := unmarshal(t, *message)
					// `default` and `body` and JSON embedded within the message JSON
					msgDefault := unmarshal(t, messageObj["default"].(string))
					msgBody := unmarshal(t, messageObj["Body"].(string))

					assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

					// Check that we're sending the account object
					assert.Equal(t, "", msgBody["Id"])

					return true
				}), true,
			).Return(aws.String("mock message"), nil)
			defer snsSvc.AssertExpectations(t)

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should change account status from NotReady to Ready", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			snsSvc := &commonMocks.Notificationer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.DceLease{}, nil)

			// Should change the Account Status
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.DceAccount{}, nil)

			snsSvc.On("PublishMessage",
				mock.MatchedBy(func(arn *string) bool {
					return *arn == "Topic"
				}),
				mock.MatchedBy(func(message *string) bool {
					// Parse the message JSON
					messageObj := unmarshal(t, *message)
					// `default` and `body` and JSON embedded within the message JSON
					msgDefault := unmarshal(t, messageObj["default"].(string))
					msgBody := unmarshal(t, messageObj["Body"].(string))

					assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

					// Check that we're sending the account object
					assert.Equal(t, "", msgBody["Id"])

					return true
				}), true,
			).Return(aws.String("mock message"), nil)
			defer snsSvc.AssertExpectations(t)

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should not change account status of Leased accounts", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			snsSvc := &commonMocks.Notificationer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.DceLease{}, nil)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, &db.StatusTransitionError{})

			dbSvc.
				On("GetAccount", "111").
				Return(&db.DceAccount{}, nil)

			snsSvc.On("PublishMessage",
				mock.MatchedBy(func(arn *string) bool {
					return *arn == "Topic"
				}),
				mock.MatchedBy(func(message *string) bool {
					// Parse the message JSON
					messageObj := unmarshal(t, *message)
					// `default` and `body` and JSON embedded within the message JSON
					msgDefault := unmarshal(t, messageObj["default"].(string))
					msgBody := unmarshal(t, messageObj["Body"].(string))

					assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

					// Check that we're sending the account object
					assert.Equal(t, "", msgBody["Id"])

					return true
				}), true,
			).Return(aws.String("mock message"), nil)
			defer snsSvc.AssertExpectations(t)

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should handle DB errors (FindLeasesByAccount)", func(t *testing.T) {
			snsSvc := &commonMocks.Notificationer{}
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 0)
			require.Equal(t, errors.New("test error"), err)
		})

		t.Run("Should handle DB errors (TransitionAccountStatus)", func(t *testing.T) {
			snsSvc := &commonMocks.Notificationer{}
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.DceLease{}, nil)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Equal(t, errors.New("test error"), err)
		})
	})
}

func unmarshal(t *testing.T, jsonStr string) map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	require.Nil(t, err,
		fmt.Sprintf("Failed to unmarshal JSON: %s; %s", jsonStr, err),
	)

	return data
}
