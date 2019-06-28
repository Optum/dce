package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/Optum/Redbox/pkg/api/mocks"
	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAccountsRouter(t *testing.T) {
	require.True(t, true, "Placeholder assertion")

	t.Run("When handling a GET /accounts request", func(t *testing.T) {
		mockGetAccountsController := mocks.Controller{}
		mockGetAccountsController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(200, "Hello World"), nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts"}

		router := Router{
			GetAccountsController: &mockGetAccountsController,
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, "Hello World", result.Body, "it calls GetAccountsController")
	})

	t.Run("When handling a GET /accounts/{id} request", func(t *testing.T) {
		mockGetAccountsController := mocks.Controller{}
		mockGetAccountsController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(200, "Hello World"), nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts/123456789"}

		router := Router{
			GetAccountByIDController: &mockGetAccountsController,
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, "Hello World", result.Body, "Test calls GetAccountController")
	})

	t.Run("When handling a DELETE /accounts/{id} request", func(t *testing.T) {
		mockGetAccountsController := mocks.Controller{}
		mockGetAccountsController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(204, "Goodbye World"), nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789"}

		router := Router{
			DeleteAccountController: &mockGetAccountsController,
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, result.StatusCode, http.StatusNoContent, "returns a status no content.")
		require.Equal(t, result.Body, "Goodbye World", "returns a status no content.")
	})

	t.Run("When handling an unsupported endpoint", func(t *testing.T) {
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/unsupported"}
		router := Router{
			GetAccountsController: &mocks.Controller{},
		}

		result, err := router.route(context.TODO(), &mockRequest)
		require.Nil(t, err)

		require.Equal(t, 404, result.StatusCode, "it returns a 404")
	})

}
