package response

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// CreateAPIResponse is a helper function to create and return a valid response
// for an API Gateway
func CreateHTMLResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":                "text/html",
			"Access-Control-Allow-Origin": "*",
		},
		Body: body,
	}
}

// CreateAPIResponse is a helper function to create and return a valid response
// for an API Gateway
func CreateAPIResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: body,
	}
}

// CreateJSONResponse - Create a JSON response
func CreateJSONResponse(status int, response interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(response)

	// Create an error response, to handle the marshalling error
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v; %v", response, err)
		return ServerError()
	}

	return CreateAPIResponse(status, string(body))
}

// CreateAPIErrorResponse is a helper function to create and return a valid error
// response message for the API
func CreateAPIErrorResponse(responseCode int,
	errResp ErrorResponse) events.APIGatewayProxyResponse {
	// Create the Error Response
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		return CreateAPIResponse(http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Return an error
	return CreateAPIResponse(responseCode, string(apiResponse))
}
