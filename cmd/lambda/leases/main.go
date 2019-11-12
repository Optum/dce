package main

import (
	"context"
	"fmt"
	"net/http"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
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
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	// Config - The configuration client
	Config common.DefaultEnvConfig
	// AWSSession - The AWS session
	AWSSession *session.Session
	// Dao - Database service
	Dao db.DBer
	// SnsSvc - SNS service
	SnsSvc common.Notificationer
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
	log.Println("Cold start; creating router for /leases")

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
	return muxLambda.ProxyWithContext(ctx, req)
}

// buildBaseURL returns a base API url from the request properties.
func buildBaseURL(r *http.Request) string {
	return r.URL.String()
}

func main() {

	AWSSession = newAWSSession()
	// Create the Database Service from the environment
	Dao = newDBer()
	SnsSvc = &common.SNS{Client: sns.New(AWSSession)}

	usageSvc, err := usage.NewFromEnv()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize usage service: %s", err)
		log.Fatal(errorMessage)
	}

	UsageSvc = usageSvc

	// provisionLeaseTopicARN := common.RequireEnv("PROVISION_TOPIC")

	// router := &api.Router{
	// 	ResourceName: "/leases",
	// 	GetController: GetController{
	// 		Dao: dao,
	// 	},
	// 	ListController: ListController{
	// 		Dao: dao,
	// 	},
	// 	DeleteController: DeleteController{
	// 		Dao:                    dao,
	// 		SNS:                    snsSvc,
	// 		AccountDeletedTopicArn: accountDeletedTopicArn,
	// 		ResetQueueURL:          resetQueueURL,
	// 		Queue:                  queue,
	// 	},
	// 	CreateController: CreateController{
	// 		Dao:                   dao,
	// 		Provisioner:           prov,
	// 		SNS:                   snsSvc,
	// 		LeaseTopicARN:         &provisionLeaseTopicARN,
	// 		UsageSvc:              usageSvc,
	// 		PrincipalBudgetAmount: &principalBudgetAmount,
	// 		PrincipalBudgetPeriod: &principalBudgetPeriod,
	// 		MaxLeaseBudgetAmount:  &maxLeaseBudgetAmount,
	// 		MaxLeasePeriod:        &maxLeasePeriod,
	// 	},
	// 	UserDetails: api.UserDetails{
	// 		CognitoUserPoolID:        common.RequireEnv("COGNITO_USER_POOL_ID"),
	// 		RolesAttributesAdminName: common.RequireEnv("COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME"),
	// 		CognitoClient:            cognitoidentityprovider.New(awsSession),
	// 	},
	// }

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
