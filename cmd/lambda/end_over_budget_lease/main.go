package main

import (
	"context"
	"encoding/json"
	"github.com/Optum/dce/pkg/common"
	errors2 "github.com/Optum/dce/pkg/errors"
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

func init() {
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
	leaseRegex := regexp.MustCompile(`(Usage-Lease)[-\w]+(-Summary)`)
	principalRegex := regexp.MustCompile(`(Usage-Principal)[-\w]+`)
	switch {
	case leaseRegex.MatchString(sortKey):
		leaseSummary := leaseSummaryRecord{}
		if err := UnmarshalStreamImage(record.Change.NewImage, &leaseSummary); err != nil {
			log.Fatalln(err)
		}
		if isLeaseOverBudget(&leaseSummary) {
			log.Println("TODO: lease over budget, end lease")
		}
	case principalRegex.MatchString(sortKey):
		principalSummary := principalSummaryRecord{}
		if err := UnmarshalStreamImage(record.Change.NewImage, &principalSummary); err != nil {
			log.Fatalln(err)
		}
		if isPrincipalOverBudget(&principalSummary) {
			log.Println("TODO: principal over budget, end lease")
		}
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

		bytes, marshalErr := v.MarshalJSON(); if marshalErr != nil {
			return marshalErr
		}

		err := json.Unmarshal(bytes, &dbAttr)
		if err != nil {
			log.Fatalln(err)
		}
		dbAttrMap[k] = &dbAttr
	}

	return dynamodbattribute.UnmarshalMap(dbAttrMap, out)

}