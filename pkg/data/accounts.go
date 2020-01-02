package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// GetAccountsByStatus - Returns the accounts by status
func (a *Account) GetAccountsByStatus(status string) (*model.Accounts, error) {

	res, err := a.AwsDynamoDB.Query(&dynamodb.QueryInput{
		TableName: aws.String(a.TableName),
		IndexName: aws.String("AccountStatus"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(string(status)),
			},
		},
		KeyConditionExpression: aws.String("AccountStatus = :status"),
		ConsistentRead:         aws.Bool(a.ConsistentRead),
	})

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failed to get accounts by status %q", status),
			err,
		)
	}

	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	return accounts, err
}

// GetAccounts Get a list of accounts based on Principal ID
func (a *Account) GetAccounts(q *model.Account) (*model.Accounts, error) {
	var expr expression.Expression
	var err error
	var res *dynamodb.ScanOutput

	filters := getFiltersFromStruct(q)
	if filters != nil {
		expr, err = expression.NewBuilder().WithFilter(*filters).Build()
		if err != nil {
			return nil, errors.NewInternalServer("unabled to build query", err)
		}
	}
	res, err = a.AwsDynamoDB.Scan(&dynamodb.ScanInput{
		TableName:                 aws.String(a.TableName),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, errors.NewInternalServer("error getting accounts", err)
	}

	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of accounts", err)
	}
	return accounts, err
}
