package main

import (
	"context"
	"errors"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestDeleteAccountController_Call(t *testing.T) {
	t.Run("When there are no errors", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(nil)
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := deleteAccountController{Dao: &mockDb}
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode)
	})

	t.Run("When the account is not found", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(&db.AccountNotFoundError{})
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := deleteAccountController{Dao: &mockDb}
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("When the account is assigned", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(&db.AccountAssignedError{})
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := deleteAccountController{Dao: &mockDb}
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusConflict, response.StatusCode)
	})

	t.Run("When handling any other error", func(t *testing.T) {
		mockDb := mocks.DBer{}
		mockDb.On("DeleteAccount", "1").Return(errors.New("Test"))
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodDelete, Path: "/accounts/1"}
		controller := deleteAccountController{Dao: &mockDb}
		response, err := controller.Call(context.TODO(), &mockRequest)
		require.Nil(t, err)
		require.Equal(t, http.StatusInternalServerError, response.StatusCode)
	})
}
