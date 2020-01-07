package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListController_Call(t *testing.T) {

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {
		expectedAccounts := createAccountsOutput()
		mockAccountInput := createGetAccountsInput()
		expectedAccountsResponse := &[]*response.AccountResponse{
			{
				ID:             "123456789",
				AccountStatus:  "Ready",
				LastModifiedOn: 1561149393,
				CreatedOn:      1561149393,
				AdminRoleArn:   "mock:role:arn",
			},
		}
		mockDb := mocks.DBer{}
		mockDb.On("GetAccounts", mockAccountInput).Return(*expectedAccounts, nil)
		mockRequest := createGetSingleAccountRequest()

		Dao = &mockDb

		actualResponse, err := Handler(context.TODO(), mockRequest)
		require.Nil(t, err)

		parsedResponse := &[]*response.AccountResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), parsedResponse)
		require.Nil(t, err)

		require.Equal(t, expectedAccountsResponse, parsedResponse, "it returns a list of accounts.")
		require.Equal(t, actualResponse.StatusCode, 200, "it returns a 200.")
	})

	t.Run("When the query fails", func(t *testing.T) {
		expectedError := errors.New("error")
		mockAccountInput := createGetEmptyAccountsInput()
		mockAccountOutput := createAccountsOutput()
		mockDb := mocks.DBer{}
		mockDb.On("GetAccounts", mockAccountInput).Return(*mockAccountOutput, expectedError)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts"}

		Dao = &mockDb

		actualResponse, err := Handler(context.TODO(), mockRequest)
		require.Nil(t, err)

		require.Equal(t, actualResponse.StatusCode, 500, "it returns a 500.")
	})

	t.Run("Building next URL", func(t *testing.T) {
		body := strings.NewReader("")
		request, err := http.NewRequest("GET", "http://example.com/api/accounts?limit=2", body)

		assert.Nil(t, err)

		nextParams := make(map[string]string)

		nextParams["AccountId"] = "1"

		nextURL := response.BuildNextURL(request, nextParams, url.URL{})

		assert.Equal(t, url.Values{
			"limit": {
				"2",
			},
			"nextAccountId": {
				"1",
			},
		}, nextURL.Query())

	})

	t.Run("Empty accounts", func(t *testing.T) {

		accountsResult := createEmptyAccountsOutput()

		mockAccountInput := createGetEmptyAccountsInput()
		mockDb := mocks.DBer{}
		mockDb.On("GetAccounts", mockAccountInput).Return(*accountsResult, nil)
		mockRequest := createGetEmptyAccountsRequest()

		Dao = &mockDb

		actualResponse, err := Handler(context.Background(), mockRequest)
		require.Nil(t, err)

		parsedResponse := []response.AccountResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), &parsedResponse)
		require.Nil(t, err)

		require.Equal(t, 0, len(parsedResponse), "empty result")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})
}

func createGetEmptyAccountsInput() db.GetAccountsInput {
	keys := make(map[string]string)
	return db.GetAccountsInput{
		StartKeys: keys,
	}
}

func createGetAccountsInput() db.GetAccountsInput {
	keys := make(map[string]string)
	return db.GetAccountsInput{
		StartKeys: keys,
		AccountID: "987654321",
		Status:    db.Ready,
	}
}

func createGetSingleAccountRequest() events.APIGatewayProxyRequest {
	q := make(map[string]string)
	q[AccountIDParam] = "987654321"
	q[StatusParam] = "ready"
	return events.APIGatewayProxyRequest{
		HTTPMethod:            http.MethodGet,
		QueryStringParameters: q,
		Path:                  "/accounts",
	}
}

func createAccountsOutput() *db.GetAccountsOutput {
	nextKeys := make(map[string]string)
	accounts := []*db.Account{
		{
			ID:             "123456789",
			AccountStatus:  db.Ready,
			LastModifiedOn: 1561149393,
			CreatedOn:      1561149393,
			AdminRoleArn:   "mock:role:arn",
		},
	}
	return &db.GetAccountsOutput{
		NextKeys: nextKeys,
		Results:  accounts,
	}
}

func createGetEmptyAccountsRequest() events.APIGatewayProxyRequest {
	q := make(map[string]string)
	return events.APIGatewayProxyRequest{
		HTTPMethod:            http.MethodGet,
		QueryStringParameters: q,
		Path:                  "/accounts",
	}
}

func createEmptyAccountsOutput() *db.GetAccountsOutput {
	nextKeys := make(map[string]string)
	accounts := []*db.Account{}
	return &db.GetAccountsOutput{
		NextKeys: nextKeys,
		Results:  accounts,
	}
}
