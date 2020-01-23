package main

import (
	"context"
	"fmt"
	"net/url"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

const (
	PrincipalIDParam     = "principalId"
	AccountIDParam       = "accountId"
	NextPrincipalIDParam = "nextPrincipalId"
	NextAccountIDParam   = "nextAccountId"
	StatusParam          = "status"
	LimitParam           = "limit"
	Weekly               = "WEEKLY"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	config                   common.DefaultEnvConfig
	awsSession               *session.Session
	dao                      db.DBer
	snsSvc                   common.Notificationer
	usageSvc                 usage.Service
	leaseAddedTopicARN       *string
	principalBudgetAmount    *float64
	principalBudgetPeriod    *string
	maxLeaseBudgetAmount     *float64
	maxLeasePeriod           *int64
	defaultLeaseLengthInDays *int
	baseRequest              url.URL
)

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for lease creation/destruction
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

func init() {
	log.Println("Cold start; creating router for /leases")

	leaseAddedTopicARN = aws.String(config.GetEnvVar("LEASE_ADDED_TOPIC", "DCEDefaultProvisionTopic"))
	principalBudgetAmount = aws.Float64(config.GetEnvFloatVar("PRINCIPAL_BUDGET_AMOUNT", 1000.00))
	principalBudgetPeriod = aws.String(config.GetEnvVar("PRINCIPAL_BUDGET_PERIOD", Weekly))
	maxLeaseBudgetAmount = aws.Float64(config.GetEnvFloatVar("MAX_LEASE_BUDGET_AMOUNT", 1000.00))
	maxLeasePeriod = aws.Int64(int64(config.GetEnvIntVar("MAX_LEASE_PERIOD", 704800)))
	defaultLeaseLengthInDays = aws.Int(config.GetEnvIntVar("DEFAULT_LEASE_LENGTH_IN_DAYS", 7))

	leasesRoutes := api.Routes{
		api.Route{
			"GetLeasesByPrincipalIdAndAccountId",
			"GET",
			"/leases",
			[]string{PrincipalIDParam, AccountIDParam},
			GetLeasesByPrincipcalIDAndAccountID,
		},
		api.Route{
			"GetLeasesByPrincipalID",
			"GET",
			"/leases",
			[]string{PrincipalIDParam},
			GetLeasesByPrincipalID,
		},
		api.Route{
			"GetLeasesByAccountID",
			"GET",
			"/leases",
			[]string{AccountIDParam},
			GetLeasesByAccountID,
		},
		api.Route{
			"GetLeasesByStatus",
			"GET",
			"/leases",
			[]string{StatusParam},
			GetLeasesByStatus,
		},
		api.Route{
			"GetLeaseByID",
			"GET",
			"/leases/{leaseID}",
			api.EmptyQueryString,
			GetLeaseByID,
		},
		api.Route{
			"DeleteLeaseByID",
			"DELETE",
			"/leases/{leaseID}",
			api.EmptyQueryString,
			DeleteLeaseByID,
		},
		api.Route{
			"GetLeases",
			"GET",
			"/leases",
			api.EmptyQueryString,
			GetLeases,
		},
		api.Route{
			"DeleteLease",
			"DELETE",
			"/leases",
			api.EmptyQueryString,
			DeleteLease,
		},
		api.Route{
			"CreateLease",
			"POST",
			"/leases",
			api.EmptyQueryString,
			CreateLease,
		},
	}
	r := api.NewRouter(leasesRoutes)
	muxLambda = gorillamux.New(r)
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	// requestUser := userDetails.GetUser(&req)
	// ctxWithUser := context.WithValue(ctx, api.DceCtxKey, *requestUser)
	// return muxLambda.ProxyWithContext(ctxWithUser, req)

	// Set baseRequest information lost by integration with gorilla mux
	baseRequest = url.URL{}
	baseRequest.Scheme = req.Headers["X-Forwarded-Proto"]
	baseRequest.Host = req.Headers["Host"]
	baseRequest.Path = req.RequestContext.Stage

	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {

	awsSession = newAWSSession()
	// Create the Database Service from the environment
	dao = newDBer()
	snsSvc = &common.SNS{Client: sns.New(awsSession)}

	usageService, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	usageSvc = usageService

	lambda.Start(Handler)
}

func newDBer() db.DBer {
	dao, err := db.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize database: %s", err)
		log.Fatal(errorMessage)
	}

	return dao
}

func newAWSSession() *session.Session {
	awsSession, err := session.NewSession()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create AWS session: %s", err)
		log.Fatal(errorMessage)
	}
	return awsSession
}
