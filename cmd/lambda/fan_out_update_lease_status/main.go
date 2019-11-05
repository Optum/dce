package main

import (
	"encoding/json"
	"log"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	lambdaSDK "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

/*
This lambda initiates the Redbox budget check process. It:

- Runs on a CloudWatch scheduled event (eg. every 6 hours)
- Grabs all active leases from the DB
- For each lease, invokes the `check_budget` lambda, with the lease object as JSON payload

In this way, it acts as a _fan out_ process, to parallelize
budgeting checking process for each account.
*/
func main() {
	lambda.Start(func(cloudWatchEvent events.CloudWatchEvent) {
		log.Printf("Initializing budget check fan out")

		// Configure the DB
		dbSvc, err := db.NewFromEnv()
		if err != nil {
			log.Fatalf("Failed to configure the DB service: %s", err)
		}

		// Configure the Lambda SDK
		awsSession := session.Must(session.NewSession())
		lambdaSvc := lambdaSDK.New(awsSession)

		// Run our Lambda handler
		err = lambdaHandler(&lambdaHandlerInput{
			dbSvc:                         dbSvc,
			lambdaSvc:                     lambdaSvc,
			updateLeaseStatusFunctionName: common.RequireEnv("UPDATE_LEASE_STATUS_FUNCTION_NAME"),
		})
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Print("Budget check fan out complete")
	})
}

type lambdaHandlerInput struct {
	dbSvc                         db.DBer
	lambdaSvc                     lambdaiface.LambdaAPI
	updateLeaseStatusFunctionName string
}

func lambdaHandler(input *lambdaHandlerInput) error {
	// Grab all Status=Leased accounts from the DB
	log.Printf("Looking up active leases...")
	leases, err := input.dbSvc.FindLeasesByStatus(db.Active)
	if err != nil {
		return err
	}
	log.Printf("Found %d active leases", len(leases))

	// Invoke our `check_bucket` lambda for each lease
	invokeErrors := []error{}
	for _, lease := range leases {
		// Serialize lease as JSON
		leaseJSON, err := json.Marshal(lease)
		// save any errors to handle later
		if err != nil {
			invokeErrors = append(invokeErrors, err)
			continue
		}

		// Invoke the fan_out_update_lease_status lambda
		log.Printf("Invoking lambda %s with lease %s @ %s",
			input.updateLeaseStatusFunctionName, lease.PrincipalID, lease.AccountID)
		_, err = input.lambdaSvc.Invoke(&lambdaSDK.InvokeInput{
			FunctionName:   aws.String(input.updateLeaseStatusFunctionName),
			InvocationType: aws.String("Event"),
			Payload:        leaseJSON,
		})
		// save any errors to handle later
		if err != nil {
			log.Printf("Failed to invoke lambda %s with lease %s @ %s: %s",
				input.updateLeaseStatusFunctionName, lease.PrincipalID, lease.AccountID, err)
			invokeErrors = append(invokeErrors, err)
			continue
		}
	}

	// Combine any invocation errors into a single error response
	if len(invokeErrors) > 0 {
		return errors.NewMultiError(
			"Failed to invoke some check_budget functions",
			invokeErrors,
		)
	}

	log.Printf("Successfully invoked check_budget lambda for %d/%d leases",
		len(leases)-len(invokeErrors), len(leases))

	return nil
}
