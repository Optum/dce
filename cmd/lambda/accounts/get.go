package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/Optum/Dcs/pkg/db"

	"github.com/Optum/Dcs/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

type getController struct {
	Dao db.DBer
}

// Call - function to return a specific AWS Account record to the request
func (controller getController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the account.
	acctID := path.Base(req.Path)
	account, err := controller.Dao.GetAccount(acctID)
	if err != nil {
		log.Printf("Error Getting Account for AccountId: %s", err)
		return response.CreateAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed List on Account Lease %s",
					acctID))), nil
	}
	if account == nil {
		log.Printf("Error Getting Account for AccountId: %s", err)
		return response.NotFoundError(), nil
	}

	accountResponse := response.AccountResponse(*account)
	return response.CreateJSONResponse(http.StatusOK, accountResponse), nil
}
