package main

import (
	"log"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/event/eventiface"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type configuration struct {
	Debug         string `env:"DEBUG" envDefault:"false"`
	ResetQueueURL string `env:"RESET_SQS_URL" envDefault:"SqsUrl"`
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
		WithEventService().
		WithAccountService().
		Build()
	if err != nil {
		panic(err)
	}

	services = svcBldr

}

// Handler is the base handler function for the lambda
func Handler(cloudWatchEvent events.CloudWatchEvent) error {

	query := &account.Account{
		Status: account.StatusNotReady.StatusPtr(),
	}

	var api eventiface.Servicer
	err := services.Config.GetService(&api)
	if err != nil {
		return err
	}

	var errs []error
	err = services.AccountService().ListPages(query,
		func(accts *account.Accounts) bool {

			for _, acct := range *accts {
				a := acct
				log.Printf("Resetting account: %s\n", *a.ID)
				// Send Message
				err := api.AccountReset(&a)
				if err != nil {
					errs = append(errs, err)
				}
			}
			return true //always continue
		},
	)
	if err != nil {
		log.Printf("Account Reset Error: %v\n", err)

		return err
	}

	if len(errs) > 0 {
		return errors.NewMultiError("error when processing accounts", errs)
	}
	return nil
}

// Main
func main() {
	lambda.Start(Handler)
}
