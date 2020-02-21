package main

import (
	"context"
	"fmt"
	"log"

	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

//type lambdaConfiguration struct {
//	PrincipalBudgetAmount    float64 `env:"PRINCIPAL_BUDGET_AMOUNT" defaultEnv:"1000.00"`
//}
//
//var (
//	Services *config.ServiceBuilder
//	// Settings - the configuration settings for the controller
//	Settings *lambdaConfiguration
//)
//
//func init() {
//	cfgBldr := &config.ConfigurationBuilder{}
//	Settings = &lambdaConfiguration{}
//	if err := cfgBldr.Unmarshal(Settings); err != nil {
//		log.Fatalf("Could not load configuration: %s", err.Error())
//	}
//}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	log.Println("@@@@ START")
	// Defer errors for later
	deferredErrors := []error{}

	// We get a stream of DynDB records, representing changes to the table
	for i, record := range event.Records {
		log.Println("@@@@ RECORD # ", i)

		input := handleRecordInput{
			record:                record,
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
}

func handleRecord(input *handleRecordInput) error {
	record := input.record
	switch record.EventName {
	case "INSERT":
		fmt.Println("INSERT OLD: ", record.Change.OldImage)
		fmt.Print("INSERT NEW: ", record.Change.NewImage)

	case "MODIFY":
		fmt.Println("MODIFY OLD: ", record.Change.OldImage)
		fmt.Print("MODIFY NEW: ", record.Change.NewImage)
	default:
	}

	return nil
}