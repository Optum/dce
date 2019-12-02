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

// WriteServerErrorWithResponse - Writes a server error with the specific message.
func WriteServerErrorWithResponse(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusInternalServerError,
		"ServerError",
		message,
	)
}

// WriteAPIErrorResponse - Writes the error response out to the provided ResponseWriter
func WriteAPIErrorResponse(w http.ResponseWriter, responseCode int,
	errCode string, errMessage string) {
	// Create the Error Response
	errResp := CreateErrorResponse(errCode, errMessage)
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		WriteAPIResponse(w, http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Write an error
	WriteAPIResponse(w, responseCode, string(apiResponse))
}

// WriteAPIResponse - Writes the response out to the provided ResponseWriter
func WriteAPIResponse(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// WriteAlreadyExistsError - Writes the already exists error.
func WriteAlreadyExistsError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusConflict,
		"AlreadyExistsError",
		"The requested resource cannot be created, as it conflicts with an existing resource",
	)
}

// WriteRequestValidationError - Writes a request validate error with the given message.
func WriteRequestValidationError(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusBadRequest,
		"RequestValidationError",
		message,
	)
}

// WriteNotFoundError - Writes a request validate error with the given message.
func WriteNotFoundError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusNotFound,
		"NotFound",
		"The requested resource could not be found.",
	)
}
