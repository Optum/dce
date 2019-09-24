package main

import (
	"encoding/json"
	"errors"
	awsMocks "github.com/Optum/Dcs/pkg/awsiface/mocks"
	"github.com/Optum/Dcs/pkg/db"
	dbMocks "github.com/Optum/Dcs/pkg/db/mocks"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLambdaHandler(t *testing.T) {
	t.Run("should invoke a lambda for each active lease", func(t *testing.T) {
		// Mock the DB to return some leases
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.DcsLease{
				{AccountID: "1"},
				{AccountID: "2"},
				{AccountID: "3"},
			}, nil)

		// Mock Lambda Invoke method
		lambdaSvc := &awsMocks.LambdaAPI{}
		lambdaSvc.On("Invoke",
			mock.MatchedBy(func(input *lambda.InvokeInput) bool {
				assert.Equal(t, "check_budget", *input.FunctionName)
				assert.Equal(t, "Event", *input.InvocationType)

				// Make sure the payload is a lease
				var lease db.DcsLease
				err := json.Unmarshal(input.Payload, &lease)
				assert.Nil(t, err)

				return true
			}),
		).Return(&lambda.InvokeOutput{}, nil)

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                   dbSvc,
			lambdaSvc:               lambdaSvc,
			checkBudgetFunctionName: "check_budget",
		})
		require.Nil(t, err)

		// Check that we invoked a lambda for each lease
		lambdaSvc.AssertNumberOfCalls(t, "Invoke", 3)
	})

	t.Run("should return DB errors", func(t *testing.T) {
		// Mock the DB to return an error
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.DcsLease{}, errors.New("db error"))

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                   dbSvc,
			lambdaSvc:               &awsMocks.LambdaAPI{},
			checkBudgetFunctionName: "check_budget",
		})
		require.Equal(t, errors.New("db error"), err)
	})

	t.Run("should continue to invoke Lambdas, even if one fails", func(t *testing.T) {
		// Mock the DB to return some leases
		dbSvc := &dbMocks.DBer{}
		dbSvc.On("FindLeasesByStatus", db.Active).
			Return([]*db.DcsLease{
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
			dbSvc:                   dbSvc,
			lambdaSvc:               lambdaSvc,
			checkBudgetFunctionName: "check_budget",
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
			Return([]*db.DcsLease{}, nil)

		lambdaSvc := &awsMocks.LambdaAPI{}

		// Call the handler
		err := lambdaHandler(&lambdaHandlerInput{
			dbSvc:                   dbSvc,
			lambdaSvc:               lambdaSvc,
			checkBudgetFunctionName: "check_budget",
		})
		require.Nil(t, err)

		// Check that we invoked a lambda for each lease
		lambdaSvc.AssertNumberOfCalls(t, "Invoke", 0)
	})
}
