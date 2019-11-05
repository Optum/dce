package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func TestListController_Call(t *testing.T) {

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {
		expectedAccounts := []*db.Account{
			{
				ID:             "123456789",
				AccountStatus:  "READY",
				LastModifiedOn: 1561149393,
				CreatedOn:      1561149393,
				AdminRoleArn:   "mock:role:arn",
			},
		}
		expectedAccountsResponse := &[]*response.AccountResponse{
			{
				ID:             "123456789",
				AccountStatus:  "READY",
				LastModifiedOn: 1561149393,
				CreatedOn:      1561149393,
				AdminRoleArn:   "mock:role:arn",
			},
		}
		mockDb := mocks.DBer{}
		mockDb.On("GetAccounts").Return(expectedAccounts, nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts"}

		controller := listController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)

		parsedResponse := &[]*response.AccountResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), parsedResponse)
		require.Nil(t, err)

		require.Equal(t, expectedAccountsResponse, parsedResponse, "it returns a list of accounts.")
		require.Equal(t, actualResponse.StatusCode, 200, "it returns a 200.")
	})

	t.Run("When the query fails", func(t *testing.T) {
		expectedError := errors.New("Error")
		mockDb := mocks.DBer{}
		mockDb.On("GetAccounts").Return(nil, expectedError)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts"}

		controller := listController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, actualResponse.StatusCode, 500, "it returns a 500.")
	})

}
