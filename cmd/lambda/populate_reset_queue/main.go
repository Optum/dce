package main

import (
	"log"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/config"
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

	for {
		accounts, err := services.Config.AccountService().List(query)
		if err != nil {
			return err
		}
		for _, acct := range *accounts {
			// Send Message
			err = services.Config.AccountService().Reset(&acct)
			if err != nil {
				return err
			}
		}
		if query.NextID == nil {
			break
		}
	}

	return nil
}

// Main
func main() {
	lambda.Start(Handler)
}
