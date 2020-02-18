package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// Usage - Data Layer Struct
type Usage struct {
	DynamoDB       dynamodbiface.DynamoDBAPI
	TableName      string `env:"PRINCIPAL_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
	Limit          int64  `env:"LIMIT" envDefault:"25"`
}

// Write the Usage record in DynamoDB
// This is an upsert operation in which the record will either
// be inserted or updated
// Returns the old record
func (a *Usage) Write(usg *usage.Usage) (*usage.Usage, error) {

	var err error
	returnValue := "ALL_OLD"

	putMap, _ := dynamodbattribute.Marshal(usg)
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(a.TableName),
		Item:         putMap.M,
		ReturnValues: aws.String(returnValue),
	}

	old, err := a.DynamoDB.PutItem(input)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("update failed for usage with PrincipalID %q and SK %s", *usg.PrincipalID, *usg.SK),
			err,
		)
	}

	oldUsg := &usage.Usage{}
	err = dynamodbattribute.UnmarshalMap(old.Attributes, oldUsg)
	if err != nil {
		fmt.Printf("Error: %+v", err)
		return nil, err
	}

	fmt.Printf("Old Usage: %+v\n", oldUsg)

	return oldUsg, nil

}

// Add to CostAmount
// Returns new values
func (a *Usage) Add(usg *usage.Usage) (*usage.Usage, error) {

	var err error
	returnValue := "ALL_NEW"
	var expr expression.Expression
	var updateBldr expression.UpdateBuilder

	updateBldr = updateBldr.Add(expression.Name("CostAmount"), expression.Value(usg.CostAmount))
	updateBldr = updateBldr.Set(expression.Name("CostCurrency"), expression.Value(usg.CostCurrency))
	updateBldr = updateBldr.Set(expression.Name("Date"), expression.Value(usg.Date.Unix()))
	updateBldr = updateBldr.Set(expression.Name("TimeToLive"), expression.Value(usg.TimeToLive))
	expr, err = expression.NewBuilder().WithUpdate(updateBldr).Build()

	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PrincipalId": {
				S: usg.PrincipalID,
			},
			"SK": {
				S: usg.SK,
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
		return nil, errors.NewInternalServer(
			fmt.Sprintf("update failed for usage with PrincipalID %q and SK %s", *usg.PrincipalID, *usg.SK),
			err,
		)
	}

	newUsg := &usage.Usage{}
	err = dynamodbattribute.UnmarshalMap(old.Attributes, newUsg)
	if err != nil {
		fmt.Printf("Error: %+v", err)
		return nil, err
	}

	return newUsg, nil

}
