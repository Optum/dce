// Package main sets the handler for the Trigger Reset AWS Lambda
// Function
package main

import (
	"context"
	"log"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/processresetqueue"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Handler will processes the available entries within the available Reset
// Queue and trigger the respective Code Pipeline builds
func handler(ctx context.Context) (bool, error) {
	// Extract the build project name
	buildName := common.RequireEnv("RESET_BUILD_NAME")

	// Extract the Reset SQS Queue URL, if not provided, exit with failure
	queueURL := common.RequireEnv("RESET_SQS_URL")

	// Set up the AWS Session
	awsSession, err := session.NewSession()
	if err != nil {
		return false, err
	}

	// Construct a Queue
	sqsClient := sqs.New(awsSession)
	queue := common.SQSQueue{
		Client: sqsClient,
	}

	// Construct a Pipeline
	codeBuildClient := codebuild.New(awsSession)
	build := common.CodeBuild{
		Client: codeBuildClient,
	}

	// Construct the ResetInput
	dbSvc, err := db.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	resetInput := processresetqueue.ResetInput{
		ResetQueue:    queue,
		ResetQueueURL: &queueURL,
		ResetBuild:    &build,
		BuildName:     &buildName,
		DbSvc:         dbSvc,
	}

	// Call the Reset and return its values
	resetOutput, err := processresetqueue.Reset(&resetInput)
	log.Printf("\nOverall Reset Results: \n%+v\n", *resetOutput)
	return resetOutput.Success, err
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}
