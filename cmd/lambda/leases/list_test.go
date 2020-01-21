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

func TestListLeases(t *testing.T) {

	t.Run("Building next URL", func(t *testing.T) {
		body := strings.NewReader("")
		request, err := http.NewRequest("GET", "http://example.com/api/leases?limit=2", body)

		assert.Nil(t, err)

		nextParams := make(map[string]string)

		nextParams["AccountId"] = "1"
		nextParams["PrincipalId"] = "b"

		nextURL := response.BuildNextURL(request, nextParams, url.URL{})

		assert.Equal(t, url.Values{
			"limit": {
				"2",
			},
			"nextAccountId": {
				"1",
			},
			"nextPrincipalId": {
				"b",
			},
		}, nextURL.Query())

	})

	t.Run("Empty leases", func(t *testing.T) {

		leasesResult := createEmptyLeasesOutput()

		mockLeaseInput := createGetEmptyLeasesInput()
		mockDb := mocks.DBer{}
		mockDb.On("GetLeases", *mockLeaseInput).Return(*leasesResult, nil)
		mockRequest := createGetEmptyLeasesRequest()

		dao = &mockDb

		actualResponse, err := Handler(context.Background(), mockRequest)
		require.Nil(t, err)

		parsedResponse := []response.LeaseResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), &parsedResponse)
		require.Nil(t, err)

		require.Equal(t, 0, len(parsedResponse), "empty result")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {

		expectedLeaseResponses := []*response.LeaseResponse{
			{
				ID:             "unique-id",
				AccountID:      "987654321",
				PrincipalID:    "12345",
				LeaseStatus:    db.Active,
				LastModifiedOn: 1561149393,
			},
		}

		leasesResult := createSingleLeaseOutput()

		mockDb := mocks.DBer{}
		mockDb.On("GetLease", "987654321", "12345").Return(leasesResult, nil)
		mockRequest := createGetSingleLeaseRequest()

		dao = &mockDb

		actualResponse, err := Handler(context.Background(), mockRequest)
		require.Nil(t, err)

		parsedResponse := []*response.LeaseResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), &parsedResponse)
		require.Nil(t, err)

		require.Equal(t, expectedLeaseResponses, parsedResponse, "Returns a single lease.")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

	t.Run("When the query fails", func(t *testing.T) {
		expectedError := errors.New("Error")
		mockLeaseInput := createGetEmptyLeasesInput()
		leasesResult := createLeasesOutput()
		mockDb := mocks.DBer{}
		mockDb.On("GetLeases", *mockLeaseInput).Return(*leasesResult, expectedError)
		mockRequest := createGetLeasesRequest()

		dao = &mockDb

		actualResponse, err := Handler(context.Background(), mockRequest)
		require.Nil(t, err)

		require.Equal(t, actualResponse.StatusCode, 500, "Returns a 500.")
		require.Equal(t, actualResponse.Body, "{\"error\":{\"code\":\"ServerError\",\"message\":\"Internal server error\"}}")
	})

}

func createGetEmptyLeasesInput() *db.GetLeasesInput {
	keys := make(map[string]string)
	return &db.GetLeasesInput{
		StartKeys: keys,
	}
}

func createGetSingleLeaseRequest() events.APIGatewayProxyRequest {
	q := make(map[string]string)
	q[PrincipalIDParam] = "12345"
	q[AccountIDParam] = "987654321"
	return events.APIGatewayProxyRequest{
		HTTPMethod:            http.MethodGet,
		QueryStringParameters: q,
		Path:                  "/leases",
	}
}

func createGetLeasesRequest() events.APIGatewayProxyRequest {
	q := make(map[string]string)
	return events.APIGatewayProxyRequest{
		HTTPMethod:            http.MethodGet,
		QueryStringParameters: q,
		Path:                  "/leases",
	}
}

func createGetEmptyLeasesRequest() events.APIGatewayProxyRequest {
	q := make(map[string]string)
	return events.APIGatewayProxyRequest{
		HTTPMethod:            http.MethodGet,
		QueryStringParameters: q,
		Path:                  "/leases",
	}
}

func createLeasesOutput() *db.GetLeasesOutput {

	nextKeys := make(map[string]string)
	leases := []*db.Lease{
		{
			ID:             "unique-id",
			AccountID:      "987654321",
			PrincipalID:    "12345",
			LeaseStatus:    db.Active,
			LastModifiedOn: 1561149393,
		},
	}
	return &db.GetLeasesOutput{
		NextKeys: nextKeys,
		Results:  leases,
	}
}

func createSingleLeaseOutput() *db.Lease {
	return &db.Lease{
		ID:             "unique-id",
		AccountID:      "987654321",
		PrincipalID:    "12345",
		LeaseStatus:    db.Active,
		LastModifiedOn: 1561149393,
	}
}

func createEmptyLeasesOutput() *db.GetLeasesOutput {

	nextKeys := make(map[string]string)
	leases := []*db.Lease{}
	return &db.GetLeasesOutput{
		NextKeys: nextKeys,
		Results:  leases,
	}
}
