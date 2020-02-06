package main

import (
	"fmt"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

var (
	Services *config.ServiceBuilder
)

func initConfig() {
	log.Println("Cold start; initializing lambda")

	// load up the values into the various settings...
	cfgBldr := &config.ConfigurationBuilder{}
	_ = cfgBldr.
		WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").
		Build()

	svcBuilder := &config.ServiceBuilder{Config: cfgBldr}
	_, err := svcBuilder.
		WithAccountService().
		Build()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize account service: %s", err)
		log.Fatal(errorMessage)
	}

	Services = svcBuilder
}

// Handler - Handle the lambda function
func Handler(cloudWatchEvent events.CloudWatchEvent) {
	log.Printf("Initializing account pool metrics lambda")
	initConfig()

	accountSvc := Services.Config.AccountService()
	metrics := NewAccountPoolMetrics(accountSvc)
	metrics.Refresh()
	metrics.Publish()

	log.Print("Account pool metrics lambda complete")
}

func main() {
	lambda.Start(Handler)
}
