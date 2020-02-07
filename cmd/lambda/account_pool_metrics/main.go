package main

import (
	"fmt"
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
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
		WithCloudWatchService().
		Build()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to initialize account service: %s", err)
		log.Fatal(errorMessage)
	}

	Services = svcBuilder
}

// Hardcoding until pagination function is implemented
var QUERY_LIMIT int64 = 20000

func getMetric(status account.Status) CountMetric {
	query := account.Account{
		Status: &status,
		Limit:  &QUERY_LIMIT,
	}
	accounts, err := Services.Config.AccountService().List(&query)
	if err != nil {
		log.Fatal("failed to query accounts by name, ", err)
	}

	return CountMetric{
		name:  status.String(),
		count: len(*accounts),
	}
}

type CountMetric struct {
	name  string
	count int
}

func publishMetrics(namespace string, countMetrics CountMetric) {
	log.Println("Publishing metrics to cloudwatch")

	var cloudWatchSvc cloudwatchiface.CloudWatchAPI
	if err := Services.Config.GetService(&cloudWatchSvc); err != nil {
		panic(err)
	}

	metricData := []*cloudwatch.MetricDatum{
		{
			MetricName: aws.String(string(countMetrics.name) + "Accounts"),
			Unit:       aws.String("Count"),
			Value:      aws.Float64(float64(countMetrics.count)),
		},
	}
	_, err := cloudWatchSvc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: metricData,
	})
	if err != nil {
		log.Fatalln(err)
	}
}

// Handler - Handle the lambda function
func Handler(cloudWatchEvent events.CloudWatchEvent) {
	log.Printf("Initializing account pool metrics lambda")
	initConfig()

	Ready := getMetric(account.StatusReady)
	NotReady := getMetric(account.StatusNotReady)
	Leased := getMetric(account.StatusLeased)
	Orphaned := getMetric(account.StatusOrphaned)

	log.Println("Found ", Ready.count, Ready.name, " accounts")
	log.Println("Found ", NotReady.count, NotReady.name, " accounts")
	log.Println("Found ", Leased.count, Leased.name, " accounts")
	log.Println("Found ", Orphaned.count, Orphaned.name, " accounts")

	publishMetrics("DCE/AccountPool", Ready)
	publishMetrics("DCE/AccountPool", NotReady)
	publishMetrics("DCE/AccountPool", Leased)
	publishMetrics("DCE/AccountPool", Orphaned)

	log.Println("Published ReadyAccount Metric: ", float64(Ready.count))
	log.Println("Published NotReadyAccounts Metric: ", float64(NotReady.count))
	log.Println("Published LeasedAccounts Metric: ", float64(Leased.count))
	log.Println("Published OrphanedAccounts Metric: ", float64(Orphaned.count))

	log.Print("Account pool metrics lambda complete")
}

func main() {
	lambda.Start(Handler)
}
