package data

import (
	"strconv"
	"strings"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// query for doing a query against dynamodb
func (a *Usage) query(query *usage.Usage, keyName string, index string) (*queryScanOutput, error) {
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
	if query.NextStartDate != nil && query.NextPrincipalID != nil {
		// Should be more dynamic
		queryInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"StartDate": &dynamodb.AttributeValue{
				N: aws.String(strconv.FormatInt(*query.StartDate, 10)),
			},
			"PrincipalId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
		})
	}

	res, err = a.DynamoDB.Query(queryInput)
	if err != nil {
		return nil, errors.NewInternalServer(
			"failed to query usages",
			err,
		)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}

// scan for doing a scan against dynamodb
func (a *Usage) scan(query *usage.Usage) (*queryScanOutput, error) {
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
	if query.NextStartDate != nil && query.NextPrincipalID != nil {
		// Should be more dynamic
		scanInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"StartDate": &dynamodb.AttributeValue{
				N: aws.String(strconv.FormatInt(*query.StartDate, 10)),
			},
			"PrincipalId": &dynamodb.AttributeValue{
				S: query.NextPrincipalID,
			},
		})
	}

	res, err = a.DynamoDB.Scan(scanInput)

	if err != nil {
		return nil, errors.NewInternalServer("error getting usages", err)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}

// List Get a list of usage information
func (a *Usage) List(query *usage.Usage) (*usage.Usages, error) {

	var outputs *queryScanOutput
	var err error

	if query.Limit == nil {
		query.Limit = &a.Limit
	}

	outputs, err = a.scan(query)
	if err != nil {
		return nil, err
	}

	query.NextStartDate = nil
	query.NextPrincipalID = nil
	for k, v := range outputs.lastEvaluatedKey {
		if strings.Contains(k, "StartDate") {
			if n, err := strconv.ParseInt(*v.N, 10, 64); err == nil {
				query.NextStartDate = &n
			} else {
				return nil, errors.NewInternalServer("unexpected error translating start date to int64", err)
			}

		}
		if strings.Contains(k, "PrincipalId") {
			query.NextPrincipalID = v.S
		}
	}

	usgs := &usage.Usages{}
	err = dynamodbattribute.UnmarshalListOfMaps(outputs.items, usgs)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshal of usages", err)
	}

	return usgs, nil
}
