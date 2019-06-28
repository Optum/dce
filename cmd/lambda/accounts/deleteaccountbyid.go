package main

import (
	"context"
	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
	"path"
)

type deleteAccountController struct {
	Dao db.DBer
}

// Call handles DELETE /accounts/{id} requests. Returns no content if the operation succeeds.
func (controller deleteAccountController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accountID := path.Base(req.Path)
	err := controller.Dao.DeleteAccount(accountID)

	if err == nil {
		return response.CreateAPIResponse(http.StatusNoContent, ""), nil
	}

	switch err.(type) {
	case *db.AccountNotFoundError:
		return response.NotFoundError(), nil
	case *db.AccountAssignedError:
		return response.CreateAPIErrorResponse(http.StatusConflict, response.CreateErrorResponse("Conflict", err.Error())), nil
	default:
		return response.CreateAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError", "Internal Server Error")), nil
	}
}
