package data

import (
	gErrors "errors"
	"fmt"
	"strconv"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"

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
}

// WriteAccount the Account record in DynamoDB
func (a *Account) WriteAccount(account *model.Account, lastModifiedOn *int64) error {

	var expr expression.Expression
	var err error
	var returnValue string
	// lastModifiedOn is nil on a create
	if lastModifiedOn != nil {
		modExpr := expression.Name("LastModifiedOn").Equal(expression.Value(lastModifiedOn))
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		returnValue = "NONE"
	} else {
		modExpr := expression.Name("LastModifiedOn").AttributeNotExists()
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		returnValue = "NONE"
	}

	putMap, _ := dynamodbattribute.Marshal(account)
	_, err = a.DynamoDB.PutItem(
		&dynamodb.PutItemInput{
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
		},
	)
	var awsErr awserr.Error
	if gErrors.As(err, &awsErr) {
		if awsErr.Code() == "ConditionalCheckFailedException" {
			return errors.NewConflict(
				"account",
				*account.ID,
				fmt.Errorf("unable to update account with LastModifiedOn=%q", strconv.FormatInt(*account.LastModifiedOn, 10)))
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

// DeleteAccount the Account record in DynamoDB
func (a *Account) DeleteAccount(account *model.Account) error {

	_, err := a.DynamoDB.DeleteItem(
		&dynamodb.DeleteItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
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

// GetAccountByID the Account record by ID
func (a *Account) GetAccountByID(accountID string, account *model.Account) error {

	res, err := a.DynamoDB.GetItem(
		&dynamodb.GetItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			ConsistentRead: aws.Bool(a.ConsistentRead),
		},
	)

	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("get failed for account %q", accountID),
			err,
		)
	}

	if len(res.Item) == 0 {
		return errors.NewNotFound("account", accountID)
	}

	return dynamodbattribute.UnmarshalMap(res.Item, &account)
}
