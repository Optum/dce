package data

import (
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// query for doing a query against dynamodb
func (a *Lease) query(q *lease.Lease, keyName string, index string) (*lease.Leases, error) {
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

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(a.TableName),
		IndexName:                 aws.String(index),
		KeyConditionExpression:    expr.KeyCondition(),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}
	res, err = query(input, a.DynamoDB)

	if err != nil {
		return nil, errors.NewInternalServer(
			"failed to query leases",
			err,
		)
	}

	leases := &lease.Leases{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, leases)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of leases", err)
	}
	return leases, nil
}

// scan for doing a scan against dynamodb
func (a *Lease) scan(q *lease.Lease) (*lease.Leases, error) {
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
	input := &dynamodb.ScanInput{
		TableName:                 aws.String(a.TableName),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}
	res, err = scan(input, a.DynamoDB)
	if err != nil {
		return nil, errors.NewInternalServer("error getting leases", err)
	}

	leases := &lease.Leases{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, leases)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of leases", err)
	}
	return leases, err
}

// List Get a list of leases
func (a *Lease) List(query *lease.Lease) (*lease.Leases, error) {

	if query.Status != nil {
		return a.query(query, "LeaseStatus", "LeaseStatus")
	}
	return a.scan(query)
}
