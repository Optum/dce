package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("ACCOUNT_CREATED_TOPIC_ARN", "mock-account-created-topic")
	os.Setenv("PRINCIPAL_ROLE_NAME", "RedboxPrincipal")
	os.Setenv("RESET_SQS_URL", "mock.queue.url")
	os.Setenv("PRINCIPAL_MAX_SESSION_DURATION", "100")
	os.Setenv("PRINCIPAL_POLICY_NAME", "RedboxPrincipalDefaultPolicy")
	os.Setenv("PRINCIPAL_IAM_DENY_TAGS", "Redbox,CantTouchThis")
	os.Setenv("ACCOUNT_DELETED_TOPIC_ARN", "test:arn")
	os.Exit(m.Run())
}

// func TestAccountsRouter(t *testing.T) {
// 	require.True(t, true, "Placeholder assertion")

// 	t.Run("When handling a GET /accounts request", func(t *testing.T) {
// 		mockListController := mocks.Controller{}
// 		mockListController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(200, "Hello World"), nil)
// 		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts"}

// 		result, err := Handler(context.TODO(), mockRequest)
// 		require.Nil(t, err)

// 		require.Equal(t, "Hello World", result.Body, "it calls ListController")
// 	})

// 	t.Run("When handling a GET /accounts/{id} request", func(t *testing.T) {
// 		mockGetController := mocks.Controller{}
// 		mockGetController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(200, "Hello World"), nil)
// 		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/accounts/123456789"}

// 		result, err := Handler(context.TODO(), mockRequest)
// 		require.Nil(t, err)

// 		require.Equal(t, "Hello World", result.Body, "Test calls GetAccountController")
// 	})

// 	t.Run("When handling a DELETE /accounts/{id} request", func(t *testing.T) {
// 		mockDeleteController := mocks.Controller{}
// 		mockDeleteController.On("Call", mock.Anything, mock.Anything).Return(response.CreateAPIResponse(204, "Goodbye World"), nil)
// 		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/123456789"}

// 		result, err := Handler(context.TODO(), mockRequest)
// 		require.Nil(t, err)

// 		require.Equal(t, result.StatusCode, http.StatusNoContent, "returns a status no content.")
// 		require.Equal(t, result.Body, "Goodbye World", "returns a status no content.")
// 	})

// 	t.Run("When handling an unsupported endpoint", func(t *testing.T) {
// 		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/unsupported"}

// 		result, err := Handler(context.TODO(), mockRequest)
// 		require.Nil(t, err)

// 		require.Equal(t, 404, result.StatusCode, "it returns a 404")
// 	})

// }
