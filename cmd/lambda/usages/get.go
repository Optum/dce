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

	usages, err := controller.Dao.GetUsageByDateRange(startDate, startDate.AddDate(0, 0, 3))

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
		usageResponses = append(usageResponses, &usageRes)
	}

	messageBytes, err := json.Marshal(usageResponses)

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
