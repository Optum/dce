package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Optum/Redbox/pkg/api"
	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/usage"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Router structure holds all Controller instances for request
type Router struct {
	GetController api.Controller
}

func (router *Router) route(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var res events.APIGatewayProxyResponse
	var err error
	switch {
	case req.HTTPMethod == http.MethodGet && strings.Contains(req.Path, "/usages"):
		res, err = router.GetController.Call(ctx, req)
	default:
		return response.NotFoundError(), nil
	}

	// Handle errors returned by controllers
	if err != nil {
		log.Printf("Controller error: %s", err)
		return response.ServerError(), nil
	}

	return res, nil
}

func main() {
	// Setup services
	usageSvc := newUsage()

	// Configure the Router + Controllers
	router := &Router{
		GetController: getController{Dao: usageSvc},
	}

	// Send Lambda requests to the router
	lambda.Start(router.route)
}

func newUsage() usage.DB {
	usageSvc, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	return *usageSvc
}
