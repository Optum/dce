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
	return CreateAPIGatewayErrorResponse(
		http.StatusBadRequest,
		CreateErrorResponse("ClientError", message),
	)
}

func RequestValidationError(message string) events.APIGatewayProxyResponse {
	return CreateAPIGatewayErrorResponse(
		400,
		CreateErrorResponse("RequestValidationError", message),
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
