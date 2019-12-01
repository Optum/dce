package main

import (
	"context"
	"fmt"
	"net/http"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

const (
	StartDateParam   = "startDate"
	EndDateParam     = "endDate"
	PrincipalIDParam = "principalId"
	AccountIDParam   = "accountId"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	// Config - The configuration client
	Config common.DefaultEnvConfig
	// AWSSession - The AWS session
	AWSSession *session.Session

	// UsageSvc - Service for getting usage
	UsageSvc *usage.DB
)

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for Provision and Decommission
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

func init() {
	log.Println("Cold start; creating router for /usage")

	usageRoutes := api.Routes{

		api.Route{
			"GetUsageByStartDateAndEndDate",
			"GET",
			"/usage",
			[]string{StartDateParam, EndDateParam},
			GetUsageByStartDateAndEndDate,
		},
		api.Route{
			"GetUsageByStartDateAndPrincipalID",
			"GET",
			"/usage",
			[]string{StartDateParam, PrincipalIDParam},
			GetUsageByStartDateAndPrincipalID,
		},
		api.Route{
			"GetAllUsage",
			"GET",
			"/usage",
			api.EmptyQueryString,
			GetAllUsage,
		},
	}
	r := api.NewRouter(usageRoutes)
	muxLambda = gorillamux.New(r)
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

// buildBaseURL returns a base API url from the request properties.
func buildBaseURL(r *http.Request) string {
	return r.URL.String()
}

func main() {

	AWSSession = newAWSSession()

	UsageSvc = newUsage()

	lambda.Start(Handler)
}

func newAWSSession() *session.Session {
	awsSession, err := session.NewSession()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create AWS session: %s", err)
		log.Fatal(errorMessage)
	}
	return awsSession
}

func newUsage() *usage.DB {
	usageSvc, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	return usageSvc
}