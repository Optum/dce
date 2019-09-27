package response

import "github.com/aws/aws-lambda-go/events"

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

func RequestValidationError(message string) events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		400,
		CreateErrorResponse("RequestValidationError", message),
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

func AlreadyExistsError() events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		409,
		CreateErrorResponse("AlreadyExistsError", "The requested resource cannot be created, as it conflicts with an existing resource"),
	)
}

func NotFoundError() events.APIGatewayProxyResponse {
	return CreateAPIErrorResponse(
		404,
		CreateErrorResponse("NotFound", "The requested resource could not be found."),
	)
}
