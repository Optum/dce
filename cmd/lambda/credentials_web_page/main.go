package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	sitePathPrefix       string
	apigwDeploymentName  string
	awsCurrentRegion     string
	// Config - The configuration client
	Config common.DefaultEnvConfig
)

var identityPoolID 		 string
var userPoolProviderName string
var userPoolClientID     string
var userPoolAppWebDomain string
var userPoolID           string

func init() {
	initConfig()

	log.Println("Cold start; creating router for /auth")
	authRoutes := api.Routes{
		api.Route{
			Name:        "GetAuthPage",
			Method:      "GET",
			Pattern:     "/auth",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetAuthPage,
		},
		api.Route{
			Name:        "GetAuthPageAssets",
			Method:      "GET",
			Pattern:     "/auth/public/{file}",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetAuthPageAssets,
		},
	}
	r := api.NewRouter(authRoutes)
	muxLambda = gorillamux.New(r)
}

func initConfig() {
	sitePathPrefix = Config.GetEnvVar("SITE_PATH_PREFIX", "sitePathPrefix")
	apigwDeploymentName = Config.GetEnvVar("APIGW_DEPLOYMENT_NAME", "apigwDeploymentName")
	awsCurrentRegion = Config.GetEnvVar("AWS_CURRENT_REGION", "awsCurrentRegion")

	a := "identityPoolID"
	identityPoolID = a
	b := "userPoolProviderName"
	userPoolProviderName = b
	c := "userPoolClientID"
	userPoolClientID = c
	d := "userPoolAppWebDomain"
	userPoolAppWebDomain = d
	e := "userPoolID"
	userPoolID = e


	GetParamStoreVars(
		&GetPSVarsInput {
			EnvironmentVariable: "PS_IDENTITY_POOL_ID",
			LocalVariable: &identityPoolID,
			Default: "identityPoolID",
		},
		&GetPSVarsInput {
			EnvironmentVariable: "PS_USER_POOL_PROVIDER_NAME",
			LocalVariable: &userPoolProviderName,
			Default: "userPoolProviderName",
		},
		&GetPSVarsInput {
			EnvironmentVariable: "PS_USER_POOL_CLIENT_ID",
			LocalVariable: &userPoolClientID,
			Default: "userPoolClientID",
		},
		&GetPSVarsInput {
			EnvironmentVariable: "PS_USER_POOL_APP_WEB_DOMAIN",
			LocalVariable: &userPoolAppWebDomain,
			Default: "userPoolAppWebDomain",
		},
		&GetPSVarsInput {
			EnvironmentVariable: "PS_USER_POOL_ID",
			LocalVariable: &userPoolID,
			Default: "userPoolID",
		},
	)
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Send Lambda requests to the router
	lambda.Start(Handler)
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

type GetPSVarsInput struct {
	EnvironmentVariable string
	LocalVariable *string
	Default string
}


func GetParamStoreVars(inputs ...*GetPSVarsInput) {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            aws.Config{Region: aws.String(awsCurrentRegion)},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatal(err)
	}
	ssmsvc := ssm.New(sess, aws.NewConfig().WithRegion(awsCurrentRegion))
	withDecryption := false

	PSNamesToLocalVars := map[string]*string{}
	for _, input := range inputs {
		psName := Config.GetEnvVar(input.EnvironmentVariable, input.Default)
		PSNamesToLocalVars[psName] = input.LocalVariable
		log.Println("PSNamesToLocalVars[psName]: ", *PSNamesToLocalVars[psName])
	}

	PSNames := getKeys(PSNamesToLocalVars)
	getParametersInput := &ssm.GetParametersInput{
		Names: PSNames,
		WithDecryption: &withDecryption,
	}
	getParametersOutput, err := ssmsvc.GetParameters(getParametersInput)
	if err != nil {
		log.Fatal(err)
	}

	params := getParametersOutput.Parameters
	for _, param := range params {
		log.Print("Retrieved SSM Parameter: ", param.GoString())
		*PSNamesToLocalVars[*param.Name] = *param.Value
	}

	invalidParams := getParametersOutput.InvalidParameters
	for _, invalidParam := range invalidParams {
		log.Print("Invalid SSM Parameter: ", invalidParam)
	}

	log.Print("Local Vars: \n")
	log.Print("identityPoolID: ", identityPoolID)
	log.Print("userPoolProviderName: ", userPoolProviderName)
	log.Print("userPoolClientID: ", userPoolClientID)
	log.Print("userPoolAppWebDomain: ", userPoolAppWebDomain)
	log.Print("userPoolID: ", userPoolID)
}

func getKeys(aMap map[string]*string) []*string{
	keys := []*string{}
	for k := range aMap {
		newK := k
		keys = append(keys, &newK)
	}
	return keys
}