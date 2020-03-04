package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/data"
	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"regexp"
)

type lambdaConfig struct {
	PrincipalBudgetAmount float64 `env:"PRINCIPAL_BUDGET_AMOUNT" envDefault:"100"`
}

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

	Services = svcBldr
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
			record: record,
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
	record events.DynamoDBEventRecord
}

func handleRecord(input *handleRecordInput) error {
	record := input.record
	if record.EventName != "INSERT" && record.EventName != "MODIFY" {
		return nil
	}

	sortKey := record.Change.NewImage["SK"].String()
	leaseSummaryRegex := regexp.MustCompile((data.UsageLeaseSkSummaryPrefix) + `[-\w]+`)
	principalRegex := regexp.MustCompile(data.UsagePrincipalSkPrefix + `[-\w]+`)
	switch {
	case leaseSummaryRegex.MatchString(sortKey):
		leaseSummary := usage.Lease{}
		err := UnmarshalStreamImage(record.Change.NewImage, &leaseSummary)
		if err != nil {
			return errors2.NewInternalServer("Failed to unmarshal stream image", err)
		}

		if isLeaseOverBudget(&leaseSummary) {
			leaseID := leaseSummary.LeaseID
			log.Printf("lease id %s is over budget", *leaseID)
			_, err := Services.LeaseService().Delete(*leaseID)
			if err != nil {
				return errors2.NewInternalServer(fmt.Sprintf("Failed to delete lease for leaseID %s", *leaseID), err)
			}
			log.Printf("ended lease id %s", *leaseID)
		}
	case principalRegex.MatchString(sortKey):
		principalSummary := usage.Principal{}
		err := UnmarshalStreamImage(record.Change.NewImage, &principalSummary)
		if err != nil {
			return errors2.NewInternalServer("Failed to unmarshal stream image", err)
		}

		if isPrincipalOverBudget(&principalSummary) {
			log.Printf("principal id %s is over budget", *principalSummary.PrincipalID)
			query := lease.Lease{
				PrincipalID: principalSummary.PrincipalID,
				Status:      lease.StatusActive.StatusPtr(),
			}
			deferredErrors := []error{}
			deleteLeases := func(leases *lease.Leases) bool {
				for _, _lease := range *leases {
					_, err := Services.LeaseService().Delete(*_lease.ID)
					if err != nil {
						deferredErrors = append(deferredErrors, err)
					}
					log.Printf("ended lease id %s because principal id %s is over budget", *_lease.ID, *principalSummary.PrincipalID)
				}
				return true
			}
			err := Services.LeaseService().ListPages(&query, deleteLeases)
			if err != nil {
				return errors2.NewInternalServer(fmt.Sprintf("Failed to delete one or more leases for principalID %s", *principalSummary.PrincipalID), err)
			}
			if len(deferredErrors) > 0 {
				return errors2.NewMultiError("Failed to delete one or more leases", deferredErrors)
			}
		}
	default:
	}

	return nil
}

func isLeaseOverBudget(leaseSummary *usage.Lease) bool {
	log.Printf("lease id %s usage is %6.2f out of a %6.2f budget", *leaseSummary.LeaseID, *leaseSummary.CostAmount, *leaseSummary.BudgetAmount)
	return *leaseSummary.CostAmount >= *leaseSummary.BudgetAmount
}

func isPrincipalOverBudget(principalSummary *usage.Principal) bool {
	log.Printf("principal id %s usage is %6.2f out of a %6.2f budget", *principalSummary.PrincipalID, *principalSummary.CostAmount, Settings.PrincipalBudgetAmount)
	return *principalSummary.CostAmount >= Settings.PrincipalBudgetAmount
}

// https://stackoverflow.com/questions/49129534/unmarshal-mapstringdynamodbattributevalue-into-a-struct
// UnmarshalStreamImage converts events.DynamoDBAttributeValue to struct
func UnmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue, out interface{}) error {
	dbAttrMap := make(map[string]*dynamodb.AttributeValue)

	for k, v := range attribute {

		var dbAttr dynamodb.AttributeValue

		bytes, err := v.MarshalJSON()
		if err != nil {
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
