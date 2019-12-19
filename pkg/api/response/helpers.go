package response

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// CreateAPIGatewayResponse is a helper function to create and return a valid response
// for an API Gateway
func CreateAPIGatewayResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: body,
	}
}

// CreateMultiValueHeaderAPIResponse - creates a response with multi-value headers
func CreateMultiValueHeaderAPIResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Body: fmt.Sprintln(body),
	}
}

// CreateMultiValueHeaderAPIErrorResponse - Creates an error response with mulit-value headers
func CreateMultiValueHeaderAPIErrorResponse(status int, errorCode string, message string) events.APIGatewayProxyResponse {

	errorJSON, _ := json.Marshal(CreateErrorResponse(errorCode, message))

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Body: string(errorJSON),
	}
}

// CreateAPIGatewayJSONResponse - Create a JSON response
func CreateAPIGatewayJSONResponse(status int, response interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(response)

	// Create an error response, to handle the marshalling error
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v; %v", response, err)
		return ServerError()
	}

	return CreateAPIGatewayResponse(status, string(body))
}

// CreateMultiValueHeaderJSONResponse - Creates a response with JSON in it with multi-value headers
func CreateMultiValueHeaderJSONResponse(status int, response interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(response)

	// Create an error response, to handle the marshalling error
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v; %v", response, err)
		return ServerError()
	}

	return CreateMultiValueHeaderAPIResponse(status, string(body))
}

// CreateAPIGatewayErrorResponse is a helper function to create and return a valid error
// response message for the API
func CreateAPIGatewayErrorResponse(responseCode int,
	errResp ErrorResponse) events.APIGatewayProxyResponse {
	// Create the Error Response
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		return CreateAPIGatewayResponse(http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Return an error
	return CreateAPIGatewayResponse(responseCode, string(apiResponse))
}

// BuildNextURL merges the next parameters of pagination into the request parameters and returns an API URL.
func BuildNextURL(r *http.Request, nextParams map[string]string) string {
	responseParams := make(map[string]string)
	responseQueryStrings := make([]string, 0)

	for k, v := range r.URL.Query() {
		responseParams[k] = v[0]
	}

	for k, v := range nextParams {
		responseParams[fmt.Sprintf("next%s", k)] = v
	}

	for k, v := range responseParams {
		responseQueryStrings = append(responseQueryStrings, fmt.Sprintf("%s=%s", k, v))
	}

	queryString := strings.Join(responseQueryStrings, "&")
	return fmt.Sprintf("%s?%s", r.URL.EscapedPath(), queryString)
}
