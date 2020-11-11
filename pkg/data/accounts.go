package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/errors"
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
func (a *Account) queryAccounts(query *account.Account, keyName string, index string) (*queryScanOutput, error) {
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
	if query.NextID != nil {
		attributeValues := make(map[string]*dynamodb.AttributeValue)

		attributeValues["Id"] = &dynamodb.AttributeValue{
			S: &query.NextID.ID,
		}

		attributeValues["AccountStatus"] = &dynamodb.AttributeValue{
			S: &query.NextID.AccountStatus,
		}

		queryInput.SetExclusiveStartKey(attributeValues)

		// Should be more dynamic
		/*queryInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"Id": &dynamodb.AttributeValue{
				S: query.NextID,
			},
			"AccountStatus": &dynamodb.AttributeValue{
				S: aws.String(query.Status.String()),
			},
		})*/
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
func (a *Account) scanAccounts(query *account.Account) (*queryScanOutput, error) {
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
	if query.NextID != nil {
		attributeValues := make(map[string]*dynamodb.AttributeValue)

		attributeValues["Id"] = &dynamodb.AttributeValue{
			S: &query.NextID.ID,
		}

		attributeValues["AccountStatus"] = &dynamodb.AttributeValue{
			S: &query.NextID.AccountStatus,
		}

		scanInput.SetExclusiveStartKey(attributeValues)
		// Should be more dynamic
		/*scanInput.SetExclusiveStartKey(map[string]*dynamodb.AttributeValue{
			"Id": &dynamodb.AttributeValue{
				S: query.NextID,
			},
			"AccountStatus": &dynamodb.AttributeValue{
				S: aws.String(query.Status.String()),
			},
		})*/
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
func (a *Account) List(query *account.Account) (*account.Accounts, error) {

	var outputs *queryScanOutput
	var err error

	if query.Limit == nil {
		query.Limit = &a.Limit
	}

	if query.Status != nil {
		outputs, err = a.queryAccounts(query, "AccountStatus", "AccountStatus")
	} else {
		outputs, err = a.scanAccounts(query)
	}
	if err != nil {
		return nil, err
	}

	if outputs.lastEvaluatedKey != nil {
		query.NextID = &account.NextID{}
		fmt.Printf("Last Evaluated key: %v\n", outputs.lastEvaluatedKey)
		/*fmt.Printf("Last Evaluated key ID: %v\n", *outputs.lastEvaluatedKey["Id"])
		fmt.Printf("Last Evaluated key AccountStatus: %v\n", *outputs.lastEvaluatedKey["AccountStatus"])*/

		query.NextID = &account.NextID{}

		if outputs.lastEvaluatedKey["Id"] != nil {
			query.NextID.ID = *outputs.lastEvaluatedKey["Id"].S
		}
		if outputs.lastEvaluatedKey["AccountStatus"] != nil {
			query.NextID.AccountStatus = *outputs.lastEvaluatedKey["AccountStatus"].S
		}

		/*query.NextID = &account.NextID{
			ID:            *outputs.lastEvaluatedKey["Id"].S,
			AccountStatus: *outputs.lastEvaluatedKey["AccountStatus"].S,
		}*/
		//query.NextID.ID = *outputs.lastEvaluatedKey["Id"].S
		//query.NextID.AccountStatus = *outputs.lastEvaluatedKey["AccountStatus"].S
		fmt.Println("Has more pages")
	} else {
		query.NextID = nil
		fmt.Println("No more pages")
	}

	accounts := &account.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(outputs.items, accounts)
	if err != nil {
		return nil, errors.NewInternalServer("failed unmarshaling of accounts", err)
	}

	return accounts, nil
}
