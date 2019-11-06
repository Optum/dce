package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/api/mocks"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAccountsRouter(t *testing.T) {
	require.True(t, true, "Placeholder assertion")

	t.Run("When handling a GET /accounts request", func(t *testing.T) {
		mockListController := mocks.Controller{}
		mockListController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(200, "Hello World"), nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts"}

		router := Router{
			ListController: &mockListController,
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, "Hello World", result.Body, "it calls ListController")
	})

	t.Run("When handling a GET /accounts/{id} request", func(t *testing.T) {
		mockGetController := mocks.Controller{}
		mockGetController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(200, "Hello World"), nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts/123456789"}

		router := Router{
			GetController: &mockGetController,
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, "Hello World", result.Body, "Test calls GetAccountController")
	})

	t.Run("When handling a DELETE /accounts/{id} request", func(t *testing.T) {
		mockDeleteController := mocks.Controller{}
		mockDeleteController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(204, "Goodbye World"), nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789"}

		router := Router{
			DeleteController: &mockDeleteController,
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, result.StatusCode, http.StatusNoContent, "returns a status no content.")
		require.Equal(t, result.Body, "Goodbye World", "returns a status no content.")
	})

	t.Run("When handling an unsupported endpoint", func(t *testing.T) {
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/unsupported"}
		router := Router{
			ListController: &mocks.Controller{},
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, 404, result.StatusCode, "it returns a 404")
	})

}
