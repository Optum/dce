package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Optum/dce/pkg/usage"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

type getController struct {
	Dao usage.DB
}

// Call - function to return usage for input date range
func (controller getController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the usage records.
	i, err := strconv.ParseInt(req.QueryStringParameters["startDate"], 10, 64)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to parse usage start date: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse(
				"Invalid startDate", errorMessage)), nil
	}
	startDate := time.Unix(i, 0)

	j, err := strconv.ParseInt(req.QueryStringParameters["endDate"], 10, 64)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to parse usage end date: %s", err)
		log.Print(errorMessage)
		return response.CreateAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse(
				"Invalid endDate", errorMessage)), nil
	}
	endDate := time.Unix(j, 0)

	usageRecords, err := controller.Dao.GetUsageByDateRange(startDate, endDate)

	if err != nil {
		log.Printf("Error Getting usage records for startDate %d: %s", startDate.Unix(), err)
		return response.CreateAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed to get usage records for start date %d",
					startDate.Unix()))), nil
	}

	// Serialize them for the JSON response.
	usageResponses := []*response.UsageResponse{}

	for _, a := range usageRecords {
		usageRes := response.UsageResponse(*a)
		usageRes.StartDate = startDate.Unix()
		usageRes.EndDate = endDate.Unix()
		log.Printf("usage: %v", usageRes)
		usageResponses = append(usageResponses, &usageRes)
	}

	outputResponses := SumCostAmountByPrincipalID(usageResponses)

	messageBytes, err := json.Marshal(outputResponses)

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

// SumCostAmountByPrincipalID returns a unique subset of the input slice by finding unique PrincipalIds and adding cost amount for it.
func SumCostAmountByPrincipalID(input []*response.UsageResponse) []*response.UsageResponse {
	u := make([]*response.UsageResponse, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val.PrincipalID]; !ok {
			m[val.PrincipalID] = true
			u = append(u, val)
		} else {
			for i, item := range u {
				if item.PrincipalID == val.PrincipalID {
					log.Printf("item: %v", item)
					log.Printf("val: %v", val)
					u[i].CostAmount = u[i].CostAmount + val.CostAmount
					break
				}

			}
		}
	}

	return u
}
