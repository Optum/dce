package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-lambda-go/events"
)

type listController struct {
	Dao db.DBer
}

// Call handles GET /accounts requests. Returns a response object with a serialized list of accounts.
func (controller listController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the accounts.
	accounts, err := controller.Dao.GetAccounts()

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse(
				"ServerError", errorMessage)), nil
	}

	// Serialize them for the JSON response.
	accountResponses := []*response.AccountResponse{}

	for _, a := range accounts {
		acctRes := response.AccountResponse(*a)
		accountResponses = append(accountResponses, &acctRes)
	}

	messageBytes, err := json.Marshal(accountResponses)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to serialize data: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse(
				"ServerError", errorMessage)), nil
	}

	body := string(messageBytes)

	return response.CreateAPIResponse(http.StatusOK, body), nil
}
