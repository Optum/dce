package main

import (
	"context"
	"net/url"

	"log"

	"github.com/Optum/dce/pkg/api"
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
	conf        *leasesConfig
	baseRequest url.URL
)

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for lease creation/destruction
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

func init() {
	log.Println("Cold start; creating router for /leases")

	var err error
	conf, err = initConfig()
	if err != nil {
		log.Fatalf("Failed to initialize: %s", err)
	}

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
	lambda.Start(Handler)
}
