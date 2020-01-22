package data

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// query for doing a query against dynamodb
func (a *Account) query(query *account.Account, keyName string, index string) (*account.Accounts, error) {
	var expr expression.Expression
	var bldr expression.Builder
	var err error
	var res *dynamodb.QueryOutput

	keyCondition, filters := getFiltersFromStruct(query, &keyName)
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

	accounts := &account.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of accounts", err)
	}
	return accounts, nil
}

// scan for doing a scan against dynamodb
func (a *Account) scan(query *account.Account) (*account.Accounts, error) {
	var expr expression.Expression
	var err error
	var res *dynamodb.ScanOutput

	_, filters := getFiltersFromStruct(query, nil)
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

	accounts := &account.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of accounts", err)
	}
	return accounts, err
}

// List Get a list of accounts
func (a *Account) List(query *account.Account) (*account.Accounts, error) {

	if query.Status != nil {
		return a.query(query, "AccountStatus", "AccountStatus")
	}
	return a.scan(query)
}
