package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-lambda-go/events"
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
		WithAccountService().
		Build()
	if err != nil {
		panic(err)
	}

	services = svcBldr
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	var lease lease.Lease

	for _, record := range snsEvent.Records {
		snsRecord := record.SNS

		err := json.Unmarshal([]byte(snsRecord.Message), &lease)
		if err != nil {
			log.Printf("Failed to read SNS message %s: %s", snsRecord.Message, err.Error())
			return errors.NewInternalServer("unexpected error parsing SNS message", err)
		}

		acct, err := services.AccountService().Get(*lease.AccountID)
		if err != nil {
			return err
		}

		err = services.AccountService().UpsertPrincipalAccess(acct)
		if err != nil {
			log.Printf("Failed to update principal access for account %s: %s\n", *acct.ID, err.Error())
			return err
		}

	}
	return nil
}
