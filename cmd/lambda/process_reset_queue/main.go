// Package main sets the handler for the Trigger Reset AWS Lambda
// Function
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
)

type configuration struct {
	Debug     string `env:"DEBUG" envDefault:"false"`
	BuildName string `env:"RESET_BUILD_NAME" envDefault:"ResetCodeBuild"`
}

var (
	services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	settings *configuration
)

func init() {
	cfgBldr := &config.ConfigurationBuilder{}
	settings = &configuration{}
	if err := cfgBldr.Unmarshal(settings); err != nil {
		log.Fatalf("Could not load configuration: %s", err.Error())
	}

	// load up the values into the various settings...
	err := cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()
	if err != nil {
		log.Printf("Error: %+v", err)
	}
	svcBldr := &config.ServiceBuilder{Config: cfgBldr}

	_, err = svcBldr.
		// DCE services...
		WithCodeBuild().
		Build()
	if err != nil {
		panic(err)
	}

	services = svcBldr

}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {

	var codeBuildSvc codebuildiface.CodeBuildAPI
	if err := services.Config.GetService(&codeBuildSvc); err != nil {
		panic(err)
	}

	for _, message := range sqsEvent.Records {
		err := processMessage(codeBuildSvc, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func processMessage(codeBuildSvc codebuildiface.CodeBuildAPI, event events.SQSMessage) error {

	acct := account.Account{}
	if err := json.Unmarshal([]byte(event.Body), &acct); err != nil {
		return errors.NewInternalServer("unexpected error unmarshaling sqs message", err)
	}

	log.Printf("Start Account: %s\nMessage ID: %s\n", *acct.ID, event.MessageId)

	buildEnvironmentVars := []*codebuild.EnvironmentVariable{
		{
			Name:  aws.String("RESET_ACCOUNT"),
			Value: acct.ID,
		},
		{
			Name:  aws.String("RESET_ACCOUNT_ADMIN_ROLE_NAME"),
			Value: acct.AdminRoleArn.IAMResourceName(),
		},
		{
			Name:  aws.String("RESET_ACCOUNT_PRINCIPAL_ROLE_NAME"),
			Value: acct.PrincipalRoleArn.IAMResourceName(),
		},
	}

	// Trigger Code Pipeline
	log.Printf("Triggering Reset Build %s for Account %s\n", settings.BuildName, *acct.ID)
	_, err := codeBuildSvc.StartBuild(&codebuild.StartBuildInput{
		EnvironmentVariablesOverride: buildEnvironmentVars,
		ProjectName:                  aws.String(settings.BuildName),
	})
	if err != nil {
		return errors.NewInternalServer("unexpected error starting code build", err)
	}

	return nil
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}
