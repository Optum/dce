package data

import (
	"fmt"
	"log"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

const (
	usageLeaseSkSummaryPrefix string = "Usage-Lease-Summary-"
	usageLeaseSkDailyPrefix   string = "Usage-Lease-Daily-"
	usageSKIndex              string = "SortKey"
)

type usageLeaseData struct {
	usage.Lease
	SK         string `json:"-" dynamodbav:"SK" schema:"-"`
	TimeToLive *int64 `json:"-" dynamodbav:"TimeToLive,omitempty" schema:"-"` // ttl attribute
}

// UsageLease - Data Layer Struct
type UsageLease struct {
	DynamoDB       dynamodbiface.DynamoDBAPI
	TableName      string `env:"PRINCIPAL_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
	Limit          int64  `env:"LIMIT" envDefault:"25"`
	TimeToLive     int    `env:"USAGE_TTL" envDefault:"30"`
	BudgetPeriod   string `env:"PRINCIPAL_BUDGET_PERIOD" envDefault:"WEEKLY"`
}

// Write the Usage record in DynamoDB
// This is an upsert operation in which the record will either
// be inserted or updated
// Returns the old record
func (a *UsageLease) Write(usg *usage.Lease) error {

	var err error
	returnValue := "ALL_OLD"

	usgData := usageLeaseData{
		*usg,
		fmt.Sprintf("%s%s-%d", usageLeaseSkDailyPrefix, *usg.LeaseID, usg.Date.Unix()),
		getTTL(*usg.Date, a.TimeToLive),
	}

	putMap, _ := dynamodbattribute.Marshal(usgData)
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(a.TableName),
		Item:         putMap.M,
		ReturnValues: aws.String(returnValue),
	}

	old, err := a.DynamoDB.PutItem(input)
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("update failed for usage with PrincipalID %q and SK %s", *usgData.PrincipalID, usgData.SK),
			err,
		)
	}

	oldUsg := &usage.Lease{}
	err = dynamodbattribute.UnmarshalMap(old.Attributes, oldUsg)
	if err != nil {
		return err
	}

	diffUsg := usage.Lease{
		PrincipalID:  usg.PrincipalID,
		Date:         usg.Date,
		CostAmount:   usg.CostAmount,
		CostCurrency: usg.CostCurrency,
		LeaseID:      usg.LeaseID,
		BudgetAmount: usg.BudgetAmount,
	}
	if oldUsg.CostAmount != nil {
		diffCost := *diffUsg.CostAmount - *oldUsg.CostAmount
		diffUsg.CostAmount = &diffCost
	}
	if *diffUsg.CostAmount > 0 {
		err = a.addLeaseUsage(diffUsg)
		if err != nil {
			return err
		}

		err = a.addPrincipalUsage(diffUsg)
		if err != nil {
			return err
		}
	}

	return nil

}

// Add to CostAmount
// Returns new values
func (a *UsageLease) addLeaseUsage(usg usage.Lease) error {

	var err error
	returnValue := "ALL_NEW"
	var expr expression.Expression
	var updateBldr expression.UpdateBuilder

	usgData := usageLeaseData{
		usg,
		fmt.Sprintf("%s%s", usageLeaseSkSummaryPrefix, *usg.LeaseID),
		getTTL(*usg.Date, a.TimeToLive),
	}

	updateBldr = updateBldr.Add(expression.Name("CostAmount"), expression.Value(usgData.CostAmount))
	updateBldr = updateBldr.Set(expression.Name("LeaseId"), expression.Value(usgData.LeaseID))
	updateBldr = updateBldr.Set(expression.Name("CostCurrency"), expression.Value(usgData.CostCurrency))
	updateBldr = updateBldr.Set(expression.Name("Date"), expression.Value(usgData.Date.Unix()))
	updateBldr = updateBldr.Set(expression.Name("TimeToLive"), expression.Value(usgData.TimeToLive))
	updateBldr = updateBldr.Set(expression.Name("BudgetAmount"), expression.Value(usgData.BudgetAmount))
	expr, err = expression.NewBuilder().WithUpdate(updateBldr).Build()
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PrincipalId": {
				S: usgData.PrincipalID,
			},
			"SK": {
				S: aws.String(usgData.SK),
			},
		},
		TableName:                 aws.String(a.TableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              aws.String(returnValue),
	}

	old, err := a.DynamoDB.UpdateItem(input)
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("update failed for usage with PrincipalID %q and SK %s", *usgData.PrincipalID, usgData.SK),
			err,
		)
	}

	newUsg := &usage.Lease{}
	err = dynamodbattribute.UnmarshalMap(old.Attributes, newUsg)
	if err != nil {
		fmt.Printf("Error: %+v", err)
		return err
	}

	return nil

}

// Add to CostAmount
// Returns new values
func (a *UsageLease) addPrincipalUsage(usg usage.Lease) error {

	var err error
	returnValue := "ALL_NEW"
	var expr expression.Expression
	var updateBldr expression.UpdateBuilder

	periodStart := getBudgetPeriodTime(*usg.Date, a.BudgetPeriod)

	usgPrincipal := usage.Principal{
		PrincipalID:  usg.PrincipalID,
		Date:         &periodStart,
		CostAmount:   usg.CostAmount,
		CostCurrency: usg.CostCurrency,
	}
	usgData := usagePrincipalData{
		usgPrincipal,
		fmt.Sprintf("%s%d", usagePrincipalSkPrefix, periodStart.Unix()),
		getTTL(*usg.Date, a.TimeToLive),
	}

	updateBldr = updateBldr.Add(expression.Name("CostAmount"), expression.Value(usgData.CostAmount))
	updateBldr = updateBldr.Set(expression.Name("CostCurrency"), expression.Value(usgData.CostCurrency))
	updateBldr = updateBldr.Set(expression.Name("Date"), expression.Value(usgData.Date.Unix()))
	updateBldr = updateBldr.Set(expression.Name("TimeToLive"), expression.Value(usgData.TimeToLive))
	expr, err = expression.NewBuilder().WithUpdate(updateBldr).Build()
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PrincipalId": {
				S: usgData.PrincipalID,
			},
			"SK": {
				S: aws.String(usgData.SK),
			},
		},
		TableName:                 aws.String(a.TableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              aws.String(returnValue),
	}

	old, err := a.DynamoDB.UpdateItem(input)
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("update failed for usage with PrincipalID %q and SK %s", *usgData.PrincipalID, usgData.SK),
			err,
		)
	}

	newUsg := &usage.Lease{}
	err = dynamodbattribute.UnmarshalMap(old.Attributes, newUsg)
	if err != nil {
		fmt.Printf("Error: %+v", err)
		return err
	}

	return nil

}

// Get usage Lease summary
func (a *UsageLease) Get(ID string) (*usage.Lease, error) {
	var expr expression.Expression
	var err error

	keyCondition := expression.Key("SK").Equal(expression.Value(fmt.Sprintf("%s%s", usageLeaseSkSummaryPrefix, ID)))

	expr, err = expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, errors.NewInternalServer("unable to build query", err)
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(a.TableName),
		IndexName:                 aws.String(usageSKIndex),
		KeyConditionExpression:    expr.KeyCondition(),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	fmt.Printf("Query: %+v", queryInput)
	res, err := a.DynamoDB.Query(queryInput)

	if err != nil {
		log.Printf("Error: %+v", err)
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get failed for usage %q", ID),
			err,
		)
	}

	if len(res.Items) != 1 {
		return nil, errors.NewNotFound("usage", ID)
	}

	usg := &usage.Lease{}
	err = dynamodbattribute.UnmarshalMap(res.Items[0], usg)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling usage %q", ID),
			err,
		)
	}
	return usg, nil
}
