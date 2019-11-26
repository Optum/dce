package response

import (
	"fmt"
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
	return CreateAPIErrorResponse(
		http.StatusBadRequest,
		CreateErrorResponse("ClientError", message),
	)
}

func RequestValidationError(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		400,
		CreateErrorResponse("RequestValidationError", message),
	)
}

func UnsupportedMethodError(method string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		http.StatusMethodNotAllowed,
		CreateErrorResponse("ClientError", fmt.Sprintf("Method %s is not allowed", method)),
	)
}

func ClientErrorWithResponse(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		500,
		CreateErrorResponse("ClientError", message),
	)
}

func ClientBadRequestError(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		http.StatusBadRequest,
		CreateErrorResponse("ClientError", message),
	)
}
func ServerError() events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		500,
		CreateErrorResponse("ServerError", "Internal server error"),
	)
}

func ServerErrorWithResponse(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		500,
		CreateErrorResponse("ServerError", message),
	)
}

func ServiceUnavailableError(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		http.StatusServiceUnavailable,
		CreateErrorResponse("ServerError", message),
	)
}

func AlreadyExistsError() events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		409,
		CreateErrorResponse("AlreadyExistsError", "The requested resource cannot be created, as it conflicts with an existing resource"),
	)
}

func ConflictError(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		http.StatusConflict,
		CreateErrorResponse("ClientError", message),
	)
}

func NotFoundError() events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		404,
		CreateErrorResponse("NotFound", "The requested resource could not be found."),
	)
}

func UnauthorizedError() events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		401,
		CreateErrorResponse("Unauthorized", "Could not access the resource requested."),
	)
}
