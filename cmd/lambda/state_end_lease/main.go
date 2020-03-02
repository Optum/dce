// Get the Lease and format it for state machine usage
package main

import (
	"context"
	"log"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"

	"github.com/aws/aws-lambda-go/lambda"
)

type configuration struct {
	Debug string `env:"DEBUG" envDefault:"false"`
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
		WithLeaseService().
		Build()
	if err != nil {
		panic(err)
	}

	services = svcBldr

}

func handler(ctx context.Context, event lease.Lease) (lease.Lease, error) {

	getLease, err := services.LeaseService().Get(*event.ID)
	if err != nil {
		log.Printf("Error: %+v", err)
		return lease.Lease{}, err
	}

	if getLease.Status.String() == lease.StatusActive.String() {
		updLease, err := services.LeaseService().Delete(*getLease.ID)
		if err != nil {
			log.Printf("Error: %+v", err)
			return lease.Lease{}, err
		}
		return *updLease, nil
	}

	return *getLease, nil
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}
