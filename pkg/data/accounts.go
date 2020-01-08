package data

import (
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// queryAccounts for doing a query against dynamodb
func (a *Account) queryAccounts(q *model.Account, keyName string, index string) (*model.Accounts, error) {
	var expr expression.Expression
	var bldr expression.Builder
	var err error
	var res *dynamodb.QueryOutput

	keyCondition, filters := getFiltersFromStruct(q, &keyName)
	bldr = expression.NewBuilder().WithKeyCondition(*keyCondition)
	if filters != nil {
		bldr = bldr.WithFilter(*filters)
	}

	expr, err = bldr.Build()
	if err != nil {
		return nil, errors.NewInternalServer("unable to build query", err)
	}

	res, err = a.DynamoDB.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(a.TableName),
		IndexName:                 aws.String(index),
		KeyConditionExpression:    expr.KeyCondition(),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return nil, errors.NewInternalServer(
			"failed to query accounts",
			err,
		)
	}

	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of accounts", err)
	}
	return accounts, nil
}

// scanAccounts for doing a scan against dynamodb
func (a *Account) scanAccounts(q *model.Account) (*model.Accounts, error) {
	var expr expression.Expression
	var err error
	var res *dynamodb.ScanOutput

	_, filters := getFiltersFromStruct(q, nil)
	if filters != nil {
		expr, err = expression.NewBuilder().WithFilter(*filters).Build()
		if err != nil {
			return nil, errors.NewInternalServer("unable to build query", err)
		}
	}
	res, err = a.DynamoDB.Scan(&dynamodb.ScanInput{
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

// GetAccounts Get a list of accounts
func (a *Account) GetAccounts(q *model.Account) (*model.Accounts, error) {

	if q.Status != nil {
		return a.queryAccounts(q, "AccountStatus", "AccountStatus")
	}
	return a.scanAccounts(q)
}
