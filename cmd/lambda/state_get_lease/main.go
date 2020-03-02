// Get the Lease and format it for state machine usage
package main

import (
	"context"
	"log"
	"time"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	// Cost Explorer takes up to 24 hours to fully resolve
	// so we continue running the usage check for up to 30 hours after
	// given the 6 hours between executions
	usageContinuation = 108000
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

type eventOut struct {
	*lease.Lease
	TTL int64
}

func handler(ctx context.Context, event lease.Lease) (eventOut, error) {
	updLease, err := services.LeaseService().Get(*event.ID)
	if err != nil {
		return eventOut{}, err
	}

	var leaseEndDate int64
	if updLease.Status.String() == lease.StatusInactive.String() {
		leaseEndDate = *updLease.StatusModifiedOn
	} else {
		leaseEndDate = *updLease.ExpiresOn
	}

	leaseEndDate = leaseEndDate + usageContinuation - time.Now().Unix()
	return eventOut{
		updLease,
		leaseEndDate,
	}, nil
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}
