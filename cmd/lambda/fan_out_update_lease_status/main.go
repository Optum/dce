package main

import (
	"encoding/json"
	"log"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	lambdaSDK "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

type configuration struct {
	Debug         string `env:"DEBUG" envDefault:"false"`
	LeaseFunction string `env:"UPDATE_LEASE_STATUS_FUNCTION_NAME" envDefault:"UpdateLeaseStatusFunction"`
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
		WithLambda().
		WithLeaseService().
		Build()
	if err != nil {
		panic(err)
	}

	services = svcBldr

}

func handler(cloudWatchEvent events.CloudWatchEvent) error {

	query := &lease.Lease{
		Status: lease.StatusActive.StatusPtr(),
	}

	var lambdaSvc lambdaiface.LambdaAPI
	err := services.Config.GetService(&lambdaSvc)
	if err != nil {
		return err
	}

	var errs []error

	err = services.LeaseService().ListPages(query,
		func(leases *lease.Leases) bool {
			for _, ls := range *leases {
				leaseJSON, err := json.Marshal(&ls)
				// save any errors to handle later
				if err != nil {
					errs = append(errs, err)
					continue
				}
				// Invoke the fan_out_update_lease_status lambda
				log.Printf("Invoking lambda %s with lease %s @ %s",
					settings.LeaseFunction, *ls.PrincipalID, *ls.AccountID)
				_, err = lambdaSvc.Invoke(&lambdaSDK.InvokeInput{
					FunctionName:   aws.String(settings.LeaseFunction),
					InvocationType: aws.String("Event"),
					Payload:        leaseJSON,
				})
				// save any errors to handle later
				if err != nil {
					log.Printf("Failed to invoke lambda %s with lease %s @ %s: %s",
						settings.LeaseFunction, *ls.PrincipalID, *ls.AccountID, err)
					errs = append(errs, err)
					continue
				}
			}
			return true //always continue
		},
	)
	if err != nil {
		return err
	}

	if len(errs) > 0 {
		return errors.NewMultiError("error when processing accounts", errs)
	}
	return nil
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}
