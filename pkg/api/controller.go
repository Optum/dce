package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

// Controller is the base controller interface for API Gateway Lambda handlers.
type Controller interface {
	// Call is invoked when an instance of a controller is handling a request. Returns a response to be returned to the
	// API consumer.
	Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
}

// Router structure holds AccountController instance for request
type Router struct {
	ResourceName     string
	ListController   Controller
	DeleteController Controller
	GetController    Controller
	CreateController Controller
}

// Route - provides a router for the given resource
func (router *Router) Route(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var res events.APIGatewayProxyResponse
	var err error
	switch {
	case req.HTTPMethod == http.MethodGet && strings.Compare(req.Path, router.ResourceName) == 0:
		res, err = router.ListController.Call(ctx, req)
	case req.HTTPMethod == http.MethodGet && strings.Contains(req.Path, fmt.Sprintf("%s/", router.ResourceName)):
		res, err = router.GetController.Call(ctx, req)
	case req.HTTPMethod == http.MethodDelete && strings.Contains(req.Path, fmt.Sprintf("%s/", router.ResourceName)):
		res, err = router.DeleteController.Call(ctx, req)
	case req.HTTPMethod == http.MethodPost && strings.Compare(req.Path, router.ResourceName) == 0:
		res, err = router.CreateController.Call(ctx, req)
	default:
		return response.NotFoundError(), nil
	}

	// Handle errors that the controllers did not know how to handle
	if err != nil {
		log.Printf("Controller error: %s", err)
		return response.ServerError(), nil
	}

	return res, nil
}
