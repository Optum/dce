package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Optum/dce/pkg/db"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

type ListController struct {
	Dao db.DBer
}

func (c ListController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getLeasesInput, err := parseGetLeasesInput(req.QueryStringParameters)

	if err != nil {
		return response.RequestValidationError(fmt.Sprintf("Error parsing query params")), nil
	}

	result, err := c.Dao.GetLeases(getLeasesInput)

	if err != nil {
		return response.ServerErrorWithResponse(fmt.Sprintf("Error querying leases: %s", err)), nil
	}

	// Convert DB Lease model to API Response model
	leaseResponseItems := []response.LeaseResponse{}
	for _, lease := range result.Results {
		leaseResponseItems = append(leaseResponseItems, response.LeaseResponse(*lease))
	}

	responseBytes, err := json.Marshal(leaseResponseItems)

	if err != nil {
		return response.ServerErrorWithResponse(fmt.Sprintf("Error serializing response: %s", err)), nil
	}

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(responseBytes),
	}
	// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
	if len(result.NextKeys) > 0 {
		nextURL := buildNextURL(req, result.NextKeys)
		res.Headers["Link"] = fmt.Sprintf("<%s>; rel=\"next\"", nextURL)
	}

	return res, nil
}

// parseGetLeasesInput creates a GetLeasesInput from the query parameters
func parseGetLeasesInput(queryParams map[string]string) (db.GetLeasesInput, error) {
	query := db.GetLeasesInput{
		StartKeys: make(map[string]string),
	}

	status, ok := queryParams[StatusParam]
	if ok && len(status) > 0 {
		query.Status = status
	}

	limit, ok := queryParams[LimitParam]
	if ok && len(limit) > 0 {
		limInt, err := strconv.ParseInt(limit, 10, 64)
		query.Limit = limInt
		if err != nil {
			return query, err
		}
	}

	principalID, ok := queryParams[PrincipalIDParam]
	if ok && len(principalID) > 0 {
		query.PrincipalID = principalID
	}

	accountID, ok := queryParams[AccountIDParam]
	if ok && len(accountID) > 0 {
		query.AccountID = accountID
	}

	nextAccountID, ok := queryParams[NextAccountIDParam]
	if ok && len(nextAccountID) > 0 {
		query.StartKeys["AccountId"] = nextAccountID
	}

	nextPrincipalID, ok := queryParams[NextPrincipalIDParam]
	if ok && len(nextPrincipalID) > 0 {
		query.StartKeys["PrincipalId"] = nextPrincipalID
	}

	return query, nil
}

// buildNextURL merges the next parameters into the request parameters and returns an API URL.
func buildNextURL(req *events.APIGatewayProxyRequest, nextParams map[string]string) string {
	responseParams := make(map[string]string)
	responseQueryStrings := make([]string, 0)
	base := buildBaseURL(req)

	for k, v := range req.QueryStringParameters {
		responseParams[k] = v
	}

	for k, v := range nextParams {
		responseParams[fmt.Sprintf("next%s", k)] = v
	}

	for k, v := range responseParams {
		responseQueryStrings = append(responseQueryStrings, fmt.Sprintf("%s=%s", k, v))
	}

	queryString := strings.Join(responseQueryStrings, "&")
	return fmt.Sprintf("%s%s?%s", base, req.Path, queryString)
}
