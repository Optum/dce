package data

import (
	"fmt"
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
func (a *UsagePrincipal) query(query *usage.Principal) (*queryScanOutput, error) {
	var expr expression.Expression
	var bldr expression.Builder
	var err error
	var res *dynamodb.QueryOutput

	keyCondition, filters := getFiltersFromStruct(query, aws.String("PrincipalId"), nil)
	*keyCondition = keyCondition.And(expression.Key("SK").BeginsWith(UsagePrincipalSkPrefix))
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
			"SK": &dynamodb.AttributeValue{
				S: aws.String(fmt.Sprintf("%s%s", UsagePrincipalSkPrefix, strconv.FormatInt(*query.NextDate, 10))),
			},
		})
	}

	res, err = a.DynamoDB.Query(queryInput)
	if err != nil {
		fmt.Printf("%+v\n", queryInput)
		fmt.Printf("%+v\n", err)
		return nil, errors.NewInternalServer(
			"failed to query usage",
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
		*filters = filters.And(expression.Name("SK").BeginsWith(UsagePrincipalSkPrefix))
	} else {
		expr := expression.Name("SK").BeginsWith(UsagePrincipalSkPrefix)
		filters = &expr
	}

	expr, err = expression.NewBuilder().WithFilter(*filters).Build()
	if err != nil {
		return nil, errors.NewInternalServer("unable to build query", err)
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
			"SK": &dynamodb.AttributeValue{
				S: aws.String(fmt.Sprintf("%s%s", UsagePrincipalSkPrefix, strconv.FormatInt(*query.NextDate, 10))),
			},
		})
	}

	res, err = a.DynamoDB.Scan(scanInput)

	if err != nil {
		return nil, errors.NewInternalServer("error getting usage", err)
	}

	return &queryScanOutput{
		items:            res.Items,
		lastEvaluatedKey: res.LastEvaluatedKey,
	}, nil
}

// List retrieves a list of principal usage record
func (a *UsagePrincipal) List(query *usage.Principal) (*usage.Principals, error) {

	var outputs *queryScanOutput
	var err error

	if query.Limit == nil {
		query.Limit = &a.Limit
	}

	if query.PrincipalID != nil {
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
		case "PrincipalId":
			query.NextPrincipalID = v.S
		case "SK":
			n, _ := strconv.ParseInt(
				strings.Replace(*v.S, UsagePrincipalSkPrefix, "", 1),
				10, 64)
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
