package main

import (
	"encoding/json"
	"errors"
	"testing"

	awsMocks "github.com/Optum/Redbox/pkg/awsiface/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbMocks "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLambdaHandler(t *testing.T) {
	t.Run("should invoke a lambda for each active lease", func(t *testing.T) {
		// Mock the DB to return some leases
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.RedboxLease{
				{AccountID: "1"},
				{AccountID: "2"},
				{AccountID: "3"},
			}, nil)

		// Mock Lambda Invoke method
		lambdaSvc := &awsMocks.LambdaAPI{}
		lambdaSvc.On("Invoke",
			mock.MatchedBy(func(input *lambda.InvokeInput) bool {
				assert.Equal(t, "update_lease_status", *input.FunctionName)
				assert.Equal(t, "Event", *input.InvocationType)

				// Make sure the payload is a lease
				var lease db.RedboxLease
				err := json.Unmarshal(input.Payload, &lease)
				assert.Nil(t, err)

				return true
			}),
		).Return(&lambda.InvokeOutput{}, nil)

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                         dbSvc,
			lambdaSvc:                     lambdaSvc,
			updateLeaseStatusFunctionName: "update_lease_status",
		})
		require.Nil(t, err)

		// Check that we invoked a lambda for each lease
		lambdaSvc.AssertNumberOfCalls(t, "Invoke", 3)
	})

	t.Run("should return DB errors", func(t *testing.T) {
		// Mock the DB to return an error
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.RedboxLease{}, errors.New("db error"))

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                         dbSvc,
			lambdaSvc:                     &awsMocks.LambdaAPI{},
			updateLeaseStatusFunctionName: "update_lease_status",
		})
		require.Equal(t, errors.New("db error"), err)
	})

	t.Run("should continue to invoke Lambdas, even if one fails", func(t *testing.T) {
		// Mock the DB to return some leases
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.RedboxLease{
				{AccountID: "1"},
				{AccountID: "2"},
				{AccountID: "3"},
			}, nil)

		// Mock Lambda Invoke method, to fail the second time
		lambdaSvc := &awsMocks.LambdaAPI{}
		for _, i := range []int{1, 2, 3} {
			shouldErr := i == 2
			lambdaSvc.On("Invoke", mock.Anything).
				Return(&lambda.InvokeOutput{}, func(input *lambda.InvokeInput) error {
					if shouldErr {
						return errors.New("second lambda invoke failed")
					}
					return nil
				}).Times(1)
		}

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                         dbSvc,
			lambdaSvc:                     lambdaSvc,
			updateLeaseStatusFunctionName: "update_lease_status",
		})
		// should return an error
		require.Regexp(t, "second lambda invoke failed", err)

		// Check that we invoked a lambda for each lease
		// (even though the second one failed)
		lambdaSvc.AssertNumberOfCalls(t, "Invoke", 3)
	})

	t.Run("should do nothing, if there are no active leases", func(t *testing.T) {
		// Mock the DB to return no leases
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.RedboxLease{}, nil)

		lambdaSvc := &awsMocks.LambdaAPI{}

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                         dbSvc,
			lambdaSvc:                     lambdaSvc,
			updateLeaseStatusFunctionName: "update_lease_status",
		})
		require.Nil(t, err)

		// Check that we invoked a lambda for each lease
		lambdaSvc.AssertNumberOfCalls(t, "Invoke", 0)
	})
}
