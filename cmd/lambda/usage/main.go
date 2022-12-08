package main

import (
	"context"
	"fmt"
	"net/url"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

const (
	StartDateParam       = "startDate"
	EndDateParam         = "endDate"
	PrincipalIDParam     = "principalId"
	AccountIDParam       = "accountId"
	NextPrincipalIDParam = "nextPrincipalId"
	NextStartDateParam   = "nextStartDate"
	LimitParam           = "limit"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	// UsageSvc - Service for getting usage
	UsageSvc    *usage.DB
	baseRequest url.URL
)

func init() {
	log.Println("Cold start; creating router for /usage")

	usageRoutes := api.Routes{

		api.Route{
			Name:        "GetUsageByStartDateAndEndDate",
			Method:      "GET",
			Pattern:     "/usage",
			Queries:     []string{StartDateParam, EndDateParam},
			HandlerFunc: GetUsageByStartDateAndEndDate,
		},
		api.Route{
			Name:        "GetUsageByStartDateAndPrincipalID",
			Method:      "GET",
			Pattern:     "/usage",
			Queries:     []string{StartDateParam, PrincipalIDParam},
			HandlerFunc: GetUsageByStartDateAndPrincipalID,
		},
		api.Route{
			Name:        "GetAllUsage",
			Method:      "GET",
			Pattern:     "/usage",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetUsage,
		},
	}
	r := api.NewRouter(usageRoutes)
	muxLambda = gorillamux.New(r)
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error

	// Set baseRequest information lost by integration with gorilla mux
	baseRequest = url.URL{}
	baseRequest.Scheme = req.Headers["X-Forwarded-Proto"]
	baseRequest.Host = req.Headers["Host"]
	baseRequest.Path = req.RequestContext.Stage

	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {

	UsageSvc = newUsage()

	lambda.Start(Handler)
}

func newUsage() *usage.DB {
	usageSvc, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	return usageSvc
}
