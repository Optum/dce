// Get the Lease and format it for state machine usage
package main

import (
	"context"
	"log"
	"time"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/usage"

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
		WithAccountService().
		WithUsageService().
		Build()
	if err != nil {
		panic(err)
	}

	services = svcBldr

}

func handler(ctx context.Context, event lease.Lease) (lease.Lease, error) {
	acct, err := services.AccountService().Get(*event.AccountID)
	if err != nil {
		log.Printf("%+v", err)
		return lease.Lease{}, err
	}

	year, month, day := time.Now().UTC().Date()
	endDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	startDate := endDate.AddDate(0, 0, -2)
	leaseCreatedOn := time.Unix(*event.CreatedOn, 0)

	if event.Status.String() == lease.StatusInactive.String() {
		year, month, day := time.Unix(*event.StatusModifiedOn, 0).Date()
		endDate = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	if leaseCreatedOn.After(startDate) {
		startDate = time.Date(leaseCreatedOn.Year(), leaseCreatedOn.Month(), leaseCreatedOn.Day(), 0, 0, 0, 0, time.UTC)
	}
	usages, err := services.AccountService().GetUsageBetweenDates(acct, startDate, endDate)
	if err != nil {
		log.Printf("%+v", err)
		return lease.Lease{}, err
	}

	log.Printf("Got Usage for account %q: %+v", *acct.ID, usages)
	for _, usg := range usages {
		newUsg, err := usage.NewLease(
			usage.NewLeaseInput{
				PrincipalID:  *event.PrincipalID,
				LeaseID:      *event.ID,
				Date:         usg.TimePeriod,
				CostAmount:   usg.Amount,
				CostCurrency: *event.BudgetCurrency,
				BudgetAmount: *event.BudgetAmount,
			},
		)
		if err != nil {
			log.Printf("Error: %+v", err)
			return lease.Lease{}, err
		}
		err = services.UsageService().UpsertLeaseUsage(newUsg)
		if err != nil {
			log.Printf("Error: %+v", err)
			return lease.Lease{}, err
		}
	}
	return event, nil
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}
