package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Optum/Redbox/pkg/usage"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

type getController struct {
	Dao usage.DB
}

// Call - function to return usages for input date range
func (controller getController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the usage records.
	startDate := time.Date(2019, 9, 16, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 3)

	usages, err := controller.Dao.GetUsageByDateRange(startDate, endDate)

	if err != nil {
		log.Printf("Error Getting usages for startDate %d: %s", startDate.Unix(), err)
		return response.CreateAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed get usages for start date %d",
					startDate.Unix()))), nil
	}

	// Serialize them for the JSON response.
	usageResponses := []*response.UsageResponse{}

	for _, a := range usages {
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
					u[i].CostAmount = u[i].CostAmount + item.CostAmount
					break
				}

			}
		}
	}

	return u
}
