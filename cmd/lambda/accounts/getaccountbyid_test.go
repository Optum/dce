package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func TestGetAccountByID(t *testing.T) {

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {
		expectedAccount := &db.RedboxAccount{
			ID:             "123456789",
			AccountStatus:  db.Ready,
			LastModifiedOn: 1561149393,
		}
		expectedAccountResponse := &response.AccountResponse{
			ID:             "123456789",
			AccountStatus:  db.Ready,
			LastModifiedOn: 1561149393,
		}
		mockDb := mocks.DBer{}
		mockDb.On("GetAccount", "123456789").Return(expectedAccount, nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts/123456789"}

		controller := getAccountByIDController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)

		parsedResponse := &response.AccountResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), parsedResponse)
		require.Nil(t, err)

		require.Equal(t, expectedAccountResponse, parsedResponse, "Returns a single account.")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

	t.Run("When the query fails", func(t *testing.T) {
		expectedError := errors.New("Error")
		mockDb := mocks.DBer{}
		mockDb.On("GetAccount", "123456789").Return(nil, expectedError)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts/123456789"}

		controller := getAccountByIDController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, actualResponse.StatusCode, 500, "Returns a 500.")
		require.Equal(t, actualResponse.Body, "{\"error\":{\"code\":\"ServerError\",\"message\":\"Failed List on Account Assignment 123456789\"}}", "Returns an error response.")
	})

}
