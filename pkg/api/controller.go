package api

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// Controller is the base controller interface for API Gateway Lambda handlers.
type Controller interface {
	// Call is invoked when an instance of a controller is handling a request. Returns a response to be returned to the
	// API consumer.
	Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
}
