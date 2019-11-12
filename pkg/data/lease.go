package data

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// Lease - Data Layer Struct
type Lease struct {
	AwsDynamoDB    dynamodbiface.DynamoDBAPI
	TableName      string
	ConsistentRead bool
}

// Update the Lease record in DynamoDB
func (l *Lease) Update(input interface{}, lastModifiedOn int64) error {

	modExpr := expression.Name("LastModifiedOn").Equal(expression.Value(lastModifiedOn))
	expr, err := expression.NewBuilder().WithCondition(modExpr).Build()

	putMap, _ := dynamodbattribute.Marshal(input)

	res, err := l.AwsDynamoDB.PutItem(
		&dynamodb.PutItemInput{
			// Query in Lease Table
			TableName: aws.String(l.TableName),
			// Find Account for the requested accountId
			Item: putMap.M,
			// Condition Expression
			ConditionExpression: expr.Condition(),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
		},
	)

	if err != nil {
		log.Printf("Failed to update account: %s", err)
		return err
	}

	return dynamodbattribute.UnmarshalMap(res.Attributes, &input)
}

// GetByID Get the Lease by ID
func (l *Lease) GetByID(accountID string, input interface{}) error {

	res, err := l.AwsDynamoDB.GetItem(
		&dynamodb.GetItemInput{
			// Query in Lease Table
			TableName: aws.String(l.TableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			ConsistentRead: aws.Bool(l.ConsistentRead),
		},
	)

	if err != nil {
		log.Printf("Failed to update account: %s", err)
		return err
	}

	return dynamodbattribute.UnmarshalMap(res.Item, &input)
}
