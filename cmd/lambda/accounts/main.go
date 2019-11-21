package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	// CurrentAccountID - The ID of the AWS Account this is running in
	CurrentAccountID *string
	// AWSSession - The AWS session
	AWSSession *session.Session
	// RoleManager - Manages the roles
	RoleManager rolemanager.RoleManager
	// Dao - Database service
	Dao db.DBer
	// SnsSvc - SNS service
	SnsSvc common.Notificationer
	// Queue - SQS Queue client
	Queue common.Queue
	// TokenSvc - Token service client
	TokenSvc common.TokenService
	// StorageSvc - Storage service client
	StorageSvc common.Storager
	// Config - The configuration client
	Config common.DefaultEnvConfig
)

var (
	accountCreatedTopicArn      string
	policyName                  string
	artifactsBucket             string
	principalPolicyS3Key        string
	principalRoleName           string
	principalIAMDenyTags        []string
	principalMaxSessionDuration int64
	tags                        []*iam.Tag
	resetQueueURL               string
	allowedRegions              []string
)

func init() {
	initConfig()

	log.Println("Cold start; creating router for /accounts")
	accountRoutes := api.Routes{
		// Routes with query strings always go first,
		// because the matcher will stop on the first match
		api.Route{
			"GetAccountByStatus",
			"GET",
			"/accounts",
			[]string{"accountStatus"},
			GetAccountByStatus,
		},

		// Routes without query strings go after all of the
		// routes that use query strings for matchers.
		api.Route{
			"GetAllAccounts",
			"GET",
			"/accounts",
			api.EmptyQueryString,
			GetAllAccounts,
		},
		api.Route{
			"GetAccountByID",
			"GET",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			GetAccountByID,
		},
		api.Route{
			"UpdateAccountByID",
			"PUT",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			UpdateAccountByID,
		},
		api.Route{
			"DeleteAccount",
			"DELETE",
			"/accounts/{accountId}",
			api.EmptyQueryString,
			DeleteAccount,
		},
		api.Route{
			"CreateAccount",
			"POST",
			"/accounts",
			api.EmptyQueryString,
			CreateAccount,
		},
	}
	r := api.NewRouter(accountRoutes)
	muxLambda = gorillamux.New(r)
}

// initConfig configures package-level variables
// loaded from env vars.
func initConfig() {
	policyName = Config.GetEnvVar("PRINCIPAL_POLICY_NAME", "DCEPrincipalDefaultPolicy")
	artifactsBucket = Config.GetEnvVar("ARTIFACTS_BUCKET", "DefaultArtifactBucket")
	principalPolicyS3Key = Config.GetEnvVar("PRINCIPAL_POLICY_S3_KEY", "DefaultPrincipalPolicyS3Key")
	principalRoleName = Config.GetEnvVar("PRINCIPAL_ROLE_NAME", "DCEPrincipal")
	principalIAMDenyTags = strings.Split(Config.GetEnvVar("PRINCIPAL_IAM_DENY_TAGS", "DefaultPrincipalIamDenyTags"), ",")
	principalMaxSessionDuration = int64(Config.GetEnvIntVar("PRINCIPAL_MAX_SESSION_DURATION", 100))
	tags = []*iam.Tag{
		{Key: aws.String("Terraform"), Value: aws.String("False")},
		{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
		{Key: aws.String("Environment"), Value: aws.String(Config.GetEnvVar("TAG_ENVIRONMENT", "DefaultTagEnvironment"))},
		{Key: aws.String("Contact"), Value: aws.String(Config.GetEnvVar("TAG_CONTACT", "DefaultTagContact"))},
		{Key: aws.String("AppName"), Value: aws.String(Config.GetEnvVar("TAG_APP_NAME", "DefaultTagAppName"))},
	}
	accountCreatedTopicArn = Config.GetEnvVar("ACCOUNT_CREATED_TOPIC_ARN", "DefaultAccountCreatedTopicArn")
	resetQueueURL = Config.GetEnvVar("RESET_SQS_URL", "DefaultResetSQSUrl")
	allowedRegions = strings.Split(Config.GetEnvVar("ALLOWED_REGIONS", "us-east-1"), ",")
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	CurrentAccountID = &req.RequestContext.AccountID
	// If no name is provided in the HTTP request body, throw an error
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Setup services
	Dao = newDBer()
	AWSSession = newAWSSession()
	Queue = common.SQSQueue{Client: sqs.New(AWSSession)}
	SnsSvc = &common.SNS{Client: sns.New(AWSSession)}
	TokenSvc = common.STS{Client: sts.New(AWSSession)}

	StorageSvc = common.S3{
		Client:  s3.New(AWSSession),
		Manager: s3manager.NewDownloader(AWSSession),
	}

	RoleManager = &rolemanager.IAMRoleManager{}

	// Send Lambda requests to the router
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

// WriteServerErrorWithResponse - Writes a server error with the specific message.
func WriteServerErrorWithResponse(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusInternalServerError,
		"ServerError",
		message,
	)
}

// WriteAPIErrorResponse - Writes the error response out to the provided ResponseWriter
func WriteAPIErrorResponse(w http.ResponseWriter, responseCode int,
	errCode string, errMessage string) {
	// Create the Error Response
	errResp := response.CreateErrorResponse(errCode, errMessage)
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		WriteAPIResponse(w, http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Write an error
	WriteAPIResponse(w, responseCode, string(apiResponse))
}

// WriteAPIResponse - Writes the response out to the provided ResponseWriter
func WriteAPIResponse(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// WriteAlreadyExistsError - Writes the already exists error.
func WriteAlreadyExistsError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusConflict,
		"AlreadyExistsError",
		"The requested resource cannot be created, as it conflicts with an existing resource",
	)
}

// WriteRequestValidationError - Writes a request validate error with the given message.
func WriteRequestValidationError(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusBadRequest,
		"RequestValidationError",
		message,
	)
}

// WriteNotFoundError - Writes a request validate error with the given message.
func WriteNotFoundError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusNotFound,
		"NotFound",
		"The requested resource could not be found.",
	)
}
