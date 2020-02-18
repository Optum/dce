package data

import (
	"fmt"
	"strconv"

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
	TableName      string `env:"USAGE_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
	Limit          int64  `env:"LIMIT" envDefault:"25"`
}

// Write the Lease record in DynamoDB
// This is an upsert operation in which the record will either
// be inserted or updated
// prevLastModifiedOn parameter is the original lastModifiedOn
func (a *Usage) Write(usg *usage.Usage) error {

	var expr expression.Expression
	var err error
	returnValue := "NONE"

	putMap, _ := dynamodbattribute.Marshal(usg)
	input := &dynamodb.PutItemInput{
		TableName:                 aws.String(a.TableName),
		Item:                      putMap.M,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              aws.String(returnValue),
	}
	err = putItem(input, a.DynamoDB)

	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("update failed for usage with Start Date \"%d\" and PrincipalID %q", *usg.StartDate, *usg.PrincipalID),
			err,
		)
	}

	return nil

}

// Get gets the Usage record by StartDate and PrincipalID
func (a *Usage) Get(startDate int64, principalID string) (*usage.Usage, error) {

	input := &dynamodb.GetItemInput{
		// Query in Lease Table
		TableName: aws.String(a.TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"StartDate": {
				N: aws.String(strconv.FormatInt(startDate, 10)),
			},
			"PrincipalId": {
				S: aws.String(principalID),
			},
		},
		ConsistentRead: aws.Bool(a.ConsistentRead),
	}

	res, err := getItem(input, a.DynamoDB)

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get usage failed for start date \"%d\" and principal %q", startDate, principalID),
			err,
		)
	}

	if len(res.Item) == 0 {
		return nil, errors.NewNotFound("usage", fmt.Sprintf("%d-%s", startDate, principalID))
	}

	usg := &usage.Usage{}
	err = dynamodbattribute.UnmarshalMap(res.Item, usg)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling usage with start date \"%d\" and princiapl %q", startDate, principalID),
			err,
		)
	}
	return usg, nil
}
