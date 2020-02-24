package main

import (
	"context"
	"encoding/json"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/config"
	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"regexp"
)

var (
	Config common.DefaultEnvConfig
	principalBudgetAmount float64
)

type lambdaConfig struct {}

var (
	// Services handles the configuration of the AWS services
	Services *config.ServiceBuilder
	// Settings - the configuration settings for the controller
	Settings *lambdaConfig
)

func init() {
	cfgBldr := &config.ConfigurationBuilder{}
	Settings = &lambdaConfig{}
	if err := cfgBldr.Unmarshal(Settings); err != nil {
		log.Fatalf("Could not load configuration: %s", err.Error())
	}

	// load up the values into the various settings...
	err := cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()
	if err != nil {
		log.Printf("Error: %+v", err)
	}
	svcBldr := &config.ServiceBuilder{Config: cfgBldr}

	_, err = svcBldr.
		WithLeaseService().
		Build()
	if err != nil {
		panic(err)
	}
	principalBudgetAmount = Config.GetEnvFloatVar("PRINCIPAL_BUDGET_AMOUNT", 1000.00)
}

type leaseSummaryRecord struct {
	CostAmount   float64
	Budget       float64
	CostCurrency string
	Date         int64
	LeaseId      string
	PrincipalId  string
	SK           string
	TimeToLive   string
}

type principalSummaryRecord struct {
	CostAmount   float64
	CostCurrency string
	Date         int64
	PrincipalId  string
	SK           string
	TimeToLive   string
}

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	// Defer errors for later
	deferredErrors := []error{}

	// We get a stream of DynDB records, representing changes to the table
	for _, record := range event.Records {
		input := handleRecordInput{
			record:                record,
		}
		err := handleRecord(&input)
		if err != nil {
			deferredErrors = append(deferredErrors, err)
		}
	}

	if len(deferredErrors) > 0 {
		return errors2.NewMultiError("Failed to handle DynDB Event", deferredErrors)
	}

	return nil
}

type handleRecordInput struct {
	record                events.DynamoDBEventRecord
}

func handleRecord(input *handleRecordInput) error {
	record := input.record
	if record.EventName != "INSERT" && record.EventName != "MODIFY"{
		return nil
	}

	sortKey := record.Change.NewImage["SK"].String()
	leaseSummaryRegex := regexp.MustCompile(`(Usage-Lease)[-\w]+(-Summary)`)
	principalRegex := regexp.MustCompile(`(Usage-Principal)[-\w]+`)
	switch {
	case leaseSummaryRegex.MatchString(sortKey):
		leaseSummary := leaseSummaryRecord{}
		err := UnmarshalStreamImage(record.Change.NewImage, &leaseSummary)
		if err != nil {
			log.Printf("ERROR: Failed to unmarshal stream image")
			return err
		}
		if isLeaseOverBudget(&leaseSummary) {
			log.Printf("Lease ID %s over budget", leaseSummary.LeaseId)
			_, err := Services.LeaseService().Delete(leaseSummary.LeaseId)
			if err != nil {
				log.Printf("ERROR: failed to delete lease for leaseID %s")
				return err
			}
		}
	case principalRegex.MatchString(sortKey):
		principalSummary := principalSummaryRecord{}
		err := UnmarshalStreamImage(record.Change.NewImage, &principalSummary)
		if err != nil {
			log.Printf("ERROR: Failed to unmarshal stream image")
			return err
		}
		if isPrincipalOverBudget(&principalSummary) {
			log.Printf("Principal ID %s over budget", principalSummary.PrincipalId)
			query := lease.Lease{
				PrincipalID: &principalSummary.PrincipalId,
				Status: lease.StatusActive.StatusPtr(),
			}
			deferredErrors := []error{}
			deleteLeases :=  func(leases *lease.Leases) bool {
				for _, _lease := range *leases {
					_, err := Services.LeaseService().Delete(*_lease.ID)
					if err != nil {
						deferredErrors = append(deferredErrors, err)
					}
				}
				return true
			}
			err := Services.LeaseService().ListPages(&query, deleteLeases)
			if err != nil {
				log.Printf("ERROR: Failed to delete one or more leases for principalID %s", principalSummary.PrincipalId)
				return err
			}
			if len(deferredErrors) > 0 {
				log.Printf("ERROR: Failed to delete one or more leases %v", deferredErrors)
				return errors2.NewMultiError("Failed to handle DynDB Event", deferredErrors)
			}
		}
	default:
	}

	return nil
}

func isLeaseOverBudget(leaseSummary *leaseSummaryRecord) bool {
	log.Printf("Lease usage was %6.2f out of a %6.2f budget", leaseSummary.CostAmount, leaseSummary.Budget)
	return leaseSummary.CostAmount >= leaseSummary.Budget
}

func isPrincipalOverBudget(principalSummary *principalSummaryRecord) bool {
	log.Printf("Principal usage was %6.2f out of a %6.2f budget", principalSummary.CostAmount, principalBudgetAmount)
	return principalSummary.CostAmount >= principalBudgetAmount
}

// https://stackoverflow.com/questions/49129534/unmarshal-mapstringdynamodbattributevalue-into-a-struct
// UnmarshalStreamImage converts events.DynamoDBAttributeValue to struct
func UnmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue, out interface{}) error {

	dbAttrMap := make(map[string]*dynamodb.AttributeValue)

	for k, v := range attribute {

		var dbAttr dynamodb.AttributeValue

		bytes, err := v.MarshalJSON(); if err != nil {
			return err
		}

		err = json.Unmarshal(bytes, &dbAttr)
		if err != nil {
			return err
		}
		dbAttrMap[k] = &dbAttr
	}

	return dynamodbattribute.UnmarshalMap(dbAttrMap, out)

}