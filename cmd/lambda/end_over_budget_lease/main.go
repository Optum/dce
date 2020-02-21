package main

import (
	"context"
	"github.com/Optum/dce/pkg/config"
	"log"

	"github.com/Optum/dce/pkg/db"
	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type lambdaConfiguration struct {
	PrincipalBudgetAmount    float64 `env:"PRINCIPAL_BUDGET_AMOUNT" defaultEnv:"1000.00"`

}

var (
	Services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	Settings *lambdaConfiguration
)

func init() {
	cfgBldr := &config.ConfigurationBuilder{}
	Settings = &lambdaConfiguration{}
	if err := cfgBldr.Unmarshal(Settings); err != nil {
		log.Fatalf("Could not load configuration: %s", err.Error())
	}
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	// Defer errors for later
	deferredErrors := []error{}

	dbSvc, err := db.NewFromEnv()
	if err != nil {
		log.Fatalf("Failed to configure DB service %s", err)
	}

	// We get a stream of DynDB records, representing changes to the table
	for _, record := range event.Records {

		input := handleRecordInput{
			record:                record,
			dbSvc:                 dbSvc,
		}
		err := handleRecord(&input)
		if err != nil {
			deferredErrors = append(deferredErrors, err)
		}
	}

	if len(deferredErrors) > 0 {
		return errors2.NewMultiError("Failed to handle DynDB Event", deferredErrors)
	}

	return nil
}

type handleRecordInput struct {
	record                events.DynamoDBEventRecord
	dbSvc                 db.DBer
}

func handleRecord(input *handleRecordInput) error {
	record := input.record

	switch record.EventName {
	// We only care about modified records
	case "MODIFY":
		log.Printf("RECORD: ", record)
	default:
	}

	return nil
}