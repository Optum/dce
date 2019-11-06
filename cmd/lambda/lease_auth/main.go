package main

import (
	"fmt"

	"log"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	consoleURL    = "https://console.aws.amazon.com/"
	federationURL = "https://signin.aws.amazon.com/federation"
)

// buildBaseURL returns a base API url from the request properties.
func buildBaseURL(req *events.APIGatewayProxyRequest) string {
	return fmt.Sprintf("https://%s/%s", req.Headers["Host"], req.RequestContext.Stage)
}

func main() {

	// Create the Database Service from the environment
	dao := newDBer()

	// Create the Token Service
	awsSession := newAWSSession()
	tokenSvc := common.STS{Client: sts.New(awsSession)}
	cognitoSvc := cognitoidentityprovider.New(awsSession)

	router := &api.Router{
		ResourceName: "/auth",
		CreateController: CreateController{
			Dao:           dao,
			TokenService:  tokenSvc,
			FederationURL: federationURL,
			ConsoleURL:    consoleURL,
			UserDetailer: &api.UserDetails{
				CognitoUserPoolID:        common.RequireEnv("COGNITO_USER_POOL_ID"),
				RolesAttributesAdminName: common.RequireEnv("COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME"),
				CognitoClient:            cognitoSvc,
			},
		},
	}

	lambda.Start(router.Route)
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
