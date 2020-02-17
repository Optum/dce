package response

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// CreateErrorResponse creates and returns a formatted JSON string of the
// structured ErrorResponse
func CreateErrorResponse(code string, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorBase{
			Code:    code,
			Message: message,
		},
	}
}

// ErrorResponse is the structured JSON Response for an Error to be returned
// for APIs
// {
// 	"error": {
// 		"code": "ServerError",
// 		"message": "Error Calculating"
// 	}
// }
type ErrorResponse struct {
	Error ErrorBase `json:"error"`
}

// ErrorBase is the base structure for the ErrorResponse containing the
// Error Code and Message
// {
// 	"code": "ServerError",
// 	"message": "Error Calculating"
// }
type ErrorBase struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func BadRequestError(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		http.StatusBadRequest,
		CreateErrorResponse("ClientError", message),
	)
}

func RequestValidationError(message string) events.APIGatewayProxyResponse {
	return CreateMultiValueHeaderAPIErrorResponse(
		400,
		"RequestValidationError", message,
	)
}

func UnsupportedMethodError(method string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		http.StatusMethodNotAllowed,
		CreateErrorResponse("ClientError", fmt.Sprintf("Method %s is not allowed", method)),
	)
}

func ClientErrorWithResponse(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		500,
		CreateErrorResponse("ClientError", message),
	)
}

func ClientBadRequestError(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		http.StatusBadRequest,
		CreateErrorResponse("ClientError", message),
	)
}
func ServerError() events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		500,
		CreateErrorResponse("ServerError", "Internal server error"),
	)
}

func ServerErrorWithResponse(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		500,
		CreateErrorResponse("ServerError", message),
	)
}

func ServiceUnavailableError(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		http.StatusServiceUnavailable,
		CreateErrorResponse("ServerError", message),
	)
}

func AlreadyExistsError() events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		409,
		CreateErrorResponse("AlreadyExistsError", "The requested resource cannot be created, as it conflicts with an existing resource"),
	)
}

func ConflictError(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		http.StatusConflict,
		CreateErrorResponse("ClientError", message),
	)
}

func NotFoundError() events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		404,
		CreateErrorResponse("NotFound", "The requested resource could not be found."),
	)
}

func UnauthorizedError() events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		401,
		CreateErrorResponse("Unauthorized", "Could not access the resource requested."),
	)
}

// WriteServerError - Writes a server error with the specific message.
func WriteServerError(w http.ResponseWriter) {
	WriteServerErrorWithResponse(w, "Internal server error")
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
	_, _ = w.Write([]byte(body))
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

// WriteBadRequestError - Writes a request validate error with the given message.
func WriteBadRequestError(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusBadRequest,
		"ClientError",
		message,
	)
}

// WriteUnauthorizedError - Writes the unauthorized error.
func WriteUnauthorizedError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusUnauthorized,
		"Unauthorized",
		"Could not access the resource requested.",
	)
}

// WriteConflictError - Writes a request validate error with the given message.
func WriteConflictError(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusConflict,
		"ClientError",
		message,
	)
}

// WriteServiceUnavailableError - Writes a request validate error with the given message.
func WriteServiceUnavailableError(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusServiceUnavailable,
		"StatusServiceUnavailable",
		message,
	)
}
