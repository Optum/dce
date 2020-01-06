package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-lambda-go/events"
)

type listController struct {
	Dao db.DBer
}

// Call handles GET /accounts requests. Returns a response object with a serialized list of accounts.
func (controller listController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the accounts.

	getAccountsInput, err := parseGetAccountsInput(r)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to parse query params: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse(
				"ServerError", errorMessage)), nil
	}

	accounts, err := controller.Dao.GetAccounts(getAccountsInput)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse(
				"ServerError", errorMessage)), nil
	}

	// Serialize them for the JSON response.
	accountResponses := []*response.AccountResponse{}

	for _, a := range accounts.Results {
		acctRes := response.AccountResponse(*a)
		accountResponses = append(accountResponses, &acctRes)
	}

	messageBytes, err := json.Marshal(accountResponses)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to serialize data: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse(
				"ServerError", errorMessage)), nil
	}

	body := string(messageBytes)

	// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
	link := ""
	if len(result.NextKeys) > 0 {
		nextURL := response.BuildNextURL(r, result.NextKeys, baseRequest)
		link = fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String())
	}

	return response.CreateAPIGatewayResponseWithLinkHeader(http.StatusOK, body, link), nil
}

// parseGetAccountsInput creates a GetAccountsInput from the query parameters
func parseGetAccountsInput(r *http.Request) (db.GetAccountsInput, error) {
	query := db.GetAccountsInput{
		StartKeys: make(map[string]string),
	}

	limit := r.FormValue(LimitParam)
	if len(limit) > 0 {
		limInt, err := strconv.ParseInt(limit, 10, 64)
		query.Limit = limInt
		if err != nil {
			return query, err
		}
	}

	accountStatus := r.FormValue(StatusParam)
	if len(accountStatus) > 0 {
		query.AccountStatus = accountStatus
	}

	accountID := r.FormValue(AccountIDParam)
	if len(accountID) > 0 {
		query.AccountID = accountID
	}

	nextAccountID := r.FormValue(NextAccountIDParam)
	if len(nextAccountID) > 0 {
		query.StartKeys["Id"] = nextAccountID
	}

	return query, nil
}
