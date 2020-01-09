package data

import (
	"fmt"

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
// This is an upsert operation in which the record will either
// be inserted or updated
// prevLastModifiedOn parameter is the original lastModifiedOn
func (a *Account) WriteAccount(account *model.Account, prevLastModifiedOn *int64) error {

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
func (a *Account) GetAccountByID(accountID string) (*model.Account, error) {

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
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get failed for account %q", accountID),
			err,
		)
	}

	if len(res.Item) == 0 {
		return nil, errors.NewNotFound("account", accountID)
	}

	account := model.Account{}
	err = dynamodbattribute.UnmarshalMap(res.Item, &account)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling account %q", accountID),
			err,
		)
	}
	return &account, nil
}
