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

func TestListLeases(t *testing.T) {

	t.Run("Empty leases", func(t *testing.T) {

		leasesResult := createEmptyLeasesOutput()

		mockLeaseInput := createGetEmptyLeasesInput()
		mockDb := mocks.DBer{}
		mockDb.On("GetLeases", *mockLeaseInput).Return(*leasesResult, nil)
		mockRequest := createGetEmptyLeasesRequest()

		controller := ListController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.Background(), mockRequest)
		require.Nil(t, err)

		parsedResponse := []response.LeaseResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), &parsedResponse)
		require.Nil(t, err)

		require.Equal(t, 0, len(parsedResponse), "empty result")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {

		expectedLeaseResponses := []response.LeaseResponse{
			{
				ID:             "unique-id",
				AccountID:      "987654321",
				PrincipalID:    "12345",
				LeaseStatus:    db.Active,
				LastModifiedOn: 1561149393,
			},
		}

		leasesResult := createLeasesOutput()

		mockLeaseInput := createGetLeasesInput()
		mockDb := mocks.DBer{}
		mockDb.On("GetLeases", *mockLeaseInput).Return(*leasesResult, nil)
		mockRequest := createGetLeasesRequest()

		controller := ListController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.Background(), mockRequest)
		require.Nil(t, err)

		parsedResponse := []response.LeaseResponse{}
		err = json.Unmarshal([]byte(actualResponse.Body), &parsedResponse)
		require.Nil(t, err)

		require.Equal(t, expectedLeaseResponses, parsedResponse, "Returns a single lease.")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

	t.Run("When the query fails", func(t *testing.T) {
		expectedError := errors.New("Error")
		mockLeaseInput := createGetLeasesInput()
		leasesResult := createLeasesOutput()
		mockDb := mocks.DBer{}
		mockDb.On("GetLeases", *mockLeaseInput).Return(*leasesResult, expectedError)
		mockRequest := createGetLeasesRequest()

		controller := ListController{
			Dao: &mockDb,
		}

		actualResponse, err := controller.Call(context.Background(), mockRequest)
		require.Nil(t, err)

		require.Equal(t, actualResponse.StatusCode, 500, "Returns a 500.")
		require.Equal(t, actualResponse.Body, "{\"error\":{\"code\":\"ServerError\",\"message\":\"Error querying leases: Error\"}}")
	})

}

func createGetLeasesInput() *db.GetLeasesInput {
	keys := make(map[string]string)
	return &db.GetLeasesInput{
		PrincipalID: "12345",
		AccountID:   "987654321",
		StartKeys:   keys,
	}
}

func createGetEmptyLeasesInput() *db.GetLeasesInput {
	keys := make(map[string]string)
	return &db.GetLeasesInput{
		StartKeys: keys,
	}
}

func createGetLeasesRequest() *events.APIGatewayProxyRequest {
	q := make(map[string]string)
	q[PrincipalIDParam] = "12345"
	q[AccountIDParam] = "987654321"
	return &events.APIGatewayProxyRequest{
		HTTPMethod:            http.MethodGet,
		QueryStringParameters: q,
		Path:                  "/leases",
	}
}

func createGetEmptyLeasesRequest() *events.APIGatewayProxyRequest {
	q := make(map[string]string)
	return &events.APIGatewayProxyRequest{
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

func createEmptyLeasesOutput() *db.GetLeasesOutput {

	nextKeys := make(map[string]string)
	leases := []*db.Lease{}
	return &db.GetLeasesOutput{
		NextKeys: nextKeys,
		Results:  leases,
	}
}
