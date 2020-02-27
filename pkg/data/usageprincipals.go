package data

import (
	"strconv"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// query for doing a query against dynamodb
func (a *UsagePrincipal) query(query *usage.Principal) (*queryScanOutput, error) {
	var expr expression.Expression
	var bldr expression.Builder
	var err error
	var res *dynamodb.QueryOutput

	keyCondition, filters := getFiltersFromStruct(query, aws.String("LeaseId"), &sortKey{
		keyName:    "Date",
		typeSearch: "BeginsWith",
	})
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
		IndexName:                 aws.String(usageLeaseIndex),
		KeyConditionExpression:    expr.KeyCondition(),
		ConsistentRead:            aws.Bool(a.ConsistentRead),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	queryInput.SetLimit(*query.Limit)
	if query.NextPrincipalID != nil {
		// Should be more dynamic
		queryInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"PrincipalId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
			"LeaseId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
			"Date": &dynamodb.AttributeValue{
				N: aws.String(strconv.FormatInt(*query.NextDate, 10)),
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

// scan for doing a scan against dynamodb
func (a *UsagePrincipal) scan(query *usage.Principal) (*queryScanOutput, error) {
	var expr expression.Expression
	var err error
	var res *dynamodb.ScanOutput

	_, filters := getFiltersFromStruct(query, nil, nil)
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
	if query.NextPrincipalID != nil {
		// Should be more dynamic
		scanInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"PrincipalId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
			"LeaseId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
			"Date": &dynamodb.AttributeValue{
				N: aws.String(strconv.FormatInt(*query.NextDate, 10)),
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

// List Get a list of accounts
func (a *UsagePrincipal) List(query *usage.Principal) (*usage.Principals, error) {

	var outputs *queryScanOutput
	var err error

	if query.Limit == nil {
		query.Limit = &a.Limit
	}

	if query.PrincipalID != nil && query.Date != nil {
		outputs, err = a.query(query)
	} else {
		outputs, err = a.scan(query)
	}
	if err != nil {
		return nil, err
	}

	query.NextPrincipalID = nil
	query.NextDate = nil
	for k, v := range outputs.lastEvaluatedKey {
		switch k {
		case "NextPrincipalId":
			query.NextPrincipalID = v.S
		case "NextDate":
			n, _ := strconv.ParseInt(*v.S, 10, 64)
			query.NextDate = &n
		}
	}

	usgs := &usage.Principals{}
	err = dynamodbattribute.UnmarshalListOfMaps(outputs.items, usgs)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of usage", err)
	}

	return usgs, nil
}
