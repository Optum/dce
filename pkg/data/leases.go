package data

import (
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// queryLeases for doing a query against dynamodb
func (a *Lease) queryLeases(query *lease.Lease, keyName string, index string) (*queryScanOutput, error) {
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

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(a.TableName),
		IndexName:                 aws.String(index),
		KeyConditionExpression:    expr.KeyCondition(),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	queryInput.SetLimit(*query.Limit)
	if query.NextAccountID != nil && query.NextPrincipalID != nil{
		// Should be more dynamic
		queryInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"AccountId": &dynamodb.AttributeValue{
				S: query.NextAccountID,
			},
			"PrincipalId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
		})
	}

	res, err = a.DynamoDB.Query(queryInput)
	if err != nil {
		return nil, errors.NewInternalServer(
			"failed to query leases",
			err,
		)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}

// scanLeases for doing a scan against dynamodb
func (a *Lease) scanLeases(query *lease.Lease) (*queryScanOutput, error) {
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

	scanInput := &dynamodb.ScanInput{
		TableName:                 aws.String(a.TableName),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	scanInput.SetLimit(*query.Limit)
	if query.NextAccountID != nil {
		// Should be more dynamic
		scanInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"AccountId": &dynamodb.AttributeValue{
				S: query.NextAccountID,
			},
			"PrincipalId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
		})
	}

	res, err = a.DynamoDB.Scan(scanInput)

	if err != nil {
		return nil, errors.NewInternalServer("error getting leases", err)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}


// List Get a list of leases
func (a *Lease) List(query *lease.Lease) (*lease.Leases, error) {

	var outputs *queryScanOutput
	var err error

	if query.Limit == nil {
		query.Limit = &a.Limit
	}

	if query.Status != nil {
		outputs, err = a.queryLeases(query, "LeaseStatus", "LeaseStatus")
	} else {
		outputs, err = a.scanLeases(query)
	}
	if err != nil {
		return nil, err
	}

	query.NextAccountID = nil
	for _, v := range outputs.lastEvaluatedKey {
		query.NextAccountID = v.S
	}

	leases := &lease.Leases{}
	err = dynamodbattribute.UnmarshalListOfMaps(outputs.items, leases)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshal of leases", err)
	}

	return leases, nil
}
