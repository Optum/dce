package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// Account - Data Layer Struct
type Account struct {
	DynamoDB       dynamodbiface.DynamoDBAPI
	TableName      string `env:"ACCOUNT_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
	Limit          int64  `env:"LIMIT" envDefault:"25"`
}

// Write the Account record in DynamoDB
// This is an upsert operation in which the record will either
// be inserted or updated
// prevLastModifiedOn parameter is the original lastModifiedOn
func (a *Account) Write(account *account.Account, prevLastModifiedOn *int64) error {

	var expr expression.Expression
	var err error
	returnValue := "NONE"
	// lastModifiedOn is nil on a create
	if prevLastModifiedOn != nil {
		modExpr := expression.Name("LastModifiedOn").Equal(expression.Value(prevLastModifiedOn))
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		if err != nil {
			return errors.NewInternalServer("error building query", err)
		}
	} else {
		modExpr := expression.Name("LastModifiedOn").AttributeNotExists()
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		if err != nil {
			return errors.NewInternalServer("error building query", err)
		}
	}

	putMap, _ := dynamodbattribute.Marshal(account)
	input := &dynamodb.PutItemInput{
		// Query in Lease Table
		TableName: aws.String(a.TableName),
		// Find Account for the requested accountId
		Item: putMap.M,
		// Condition Expression
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		// Return the updated record
		ReturnValues: aws.String(returnValue),
	}
	err = putItem(input, a.DynamoDB)
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		if awsErr.Code() == "ConditionalCheckFailedException" {
			return errors.NewConflict(
				"account",
				*account.ID,
				fmt.Errorf("unable to update account: accounts has been modified since request was made"))
		}
	}
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("update failed for account %q", *account.ID),
			err,
		)
	}

	return nil
}

// Delete the Account record in DynamoDB
func (a *Account) Delete(account *account.Account) error {

	_, err := a.DynamoDB.DeleteItem(
		&dynamodb.DeleteItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			// Return the updated record
			ReturnValues: aws.String("NONE"),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: account.ID,
				},
			},
		},
	)

	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("delete failed for account %q", *account.ID),
			err,
		)
	}

	return nil
}

// Get the Account record by ID
func (a *Account) Get(ID string) (*account.Account, error) {
	res, err := a.DynamoDB.GetItem(
		&dynamodb.GetItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(ID),
				},
			},
			ConsistentRead: aws.Bool(a.ConsistentRead),
		},
	)

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get failed for account %q", ID),
			err,
		)
	}

	if len(res.Item) == 0 {
		return nil, errors.NewNotFound("account", ID)
	}

	account := &account.Account{}
	err = dynamodbattribute.UnmarshalMap(res.Item, account)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling account %q", ID),
			err,
		)
	}
	return account, nil
}
