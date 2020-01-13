package data

import (
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type queryScanOutput struct {
	items            []map[string]*dynamodb.AttributeValue
	lastEvaluatedKey map[string]*dynamodb.AttributeValue
}

// queryAccounts for doing a query against dynamodb
func (a *Account) queryAccounts(q *model.Account, keyName string, index string) (*queryScanOutput, error) {
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

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(a.TableName),
		IndexName:                 aws.String(index),
		KeyConditionExpression:    expr.KeyCondition(),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	if q.Limit != nil {
		queryInput.SetLimit(*q.Limit)
	}
	if q.NextID != nil {
		// Should be more dynamic
		queryInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"Id": &dynamodb.AttributeValue{
				S: q.NextID,
			},
		})
	}

	res, err = a.DynamoDB.Query(queryInput)
	if err != nil {
		return nil, errors.NewInternalServer(
			"failed to query accounts",
			err,
		)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}

// scanAccounts for doing a scan against dynamodb
func (a *Account) scanAccounts(q *model.Account) (*queryScanOutput, error) {
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

	scanInput := &dynamodb.ScanInput{
		TableName:                 aws.String(a.TableName),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	if q.Limit != nil {
		scanInput.SetLimit(*q.Limit)
	}
	if q.NextID != nil {
		// Should be more dynamic
		scanInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"Id": &dynamodb.AttributeValue{
				S: q.NextID,
			},
		})
	}

	res, err = a.DynamoDB.Scan(scanInput)

	if err != nil {
		return nil, errors.NewInternalServer("error getting accounts", err)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}

// GetAccounts Get a list of accounts
func (a *Account) GetAccounts(q *model.Account) (*model.Accounts, error) {

	var outputs *queryScanOutput
	var err error

	if q.Status != nil {
		outputs, err = a.queryAccounts(q, "AccountStatus", "AccountStatus")
	} else {
		outputs, err = a.scanAccounts(q)
	}
	if err != nil {
		return nil, err
	}

	q.NextID = nil
	for _, v := range outputs.lastEvaluatedKey {
		q.NextID = v.S
	}

	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(outputs.items, accounts)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of accounts", err)
	}

	return accounts, nil
}
