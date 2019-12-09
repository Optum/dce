package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// Account - Data Layer Struct
type Account struct {
	AwsDynamoDB    dynamodbiface.DynamoDBAPI
	TableName      string `env:"ACCOUNT_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
}

// Update the Account record in DynamoDB
func (a *Account) Update(account *model.Account, lastModifiedOn *int64) error {

	var expr expression.Expression
	var err error
	var returnValue string
	// lastModifiedOn is 0 on a create
	if lastModifiedOn != nil {
		modExpr := expression.Name("LastModifiedOn").Equal(expression.Value(lastModifiedOn))
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		returnValue = "ALL_NEW"
	} else {
		modExpr := expression.Name("LastModifiedOn").AttributeNotExists()
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		returnValue = "NONE"
	}

	putMap, _ := dynamodbattribute.Marshal(account)

	res, err := a.AwsDynamoDB.PutItem(
		&dynamodb.PutItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			// Find Account for the requested accountId
			Item: putMap.M,
			// Condition Expression
			ConditionExpression:      expr.Condition(),
			ExpressionAttributeNames: expr.Names(),
			// Return the updated record
			ReturnValues: aws.String(returnValue),
		},
	)

	if err != nil {
		return fmt.Errorf("update failed for account %s: %s: %w", account.ID, err, errors.ErrInternalServer)
	}

	return dynamodbattribute.UnmarshalMap(res.Attributes, &account)
}

// Delete the Account record in DynamoDB
func (a *Account) Delete(account *model.Account) error {

	res, err := a.AwsDynamoDB.DeleteItem(
		&dynamodb.DeleteItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(account.ID),
				},
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to delete account %s: %s: %w", account.ID, err, errors.ErrInternalServer)
	}

	return dynamodbattribute.UnmarshalMap(res.Attributes, &account)
}

// GetAccountByID the Account record by ID
func (a *Account) GetAccountByID(accountID string, account *model.Account) error {

	res, err := a.AwsDynamoDB.GetItem(
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
		return fmt.Errorf("failed to get account %s: %s: %w", accountID, err, errors.ErrInternalServer)
	}

	if len(res.Item) == 0 {
		return fmt.Errorf("account %s not found: %w", accountID, errors.ErrNotFound)
	}

	return dynamodbattribute.UnmarshalMap(res.Item, &account)
}

// GetAccountsByStatus - Returns the accounts by status
func (a *Account) GetAccountsByStatus(status string) (*model.Accounts, error) {
	res, err := a.AwsDynamoDB.Query(&dynamodb.QueryInput{
		TableName: aws.String(a.TableName),
		IndexName: aws.String("Status"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(string(status)),
			},
		},
		KeyConditionExpression: aws.String("Status = :status"),
		ConsistentRead:         aws.Bool(a.ConsistentRead),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting accounts by status %s: %s: %w", status, err, errors.ErrInternalServer)
	}

	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	return accounts, err
}

// GetAccountsByPrincipalID Get a list of accounts based on Principal ID
func (a *Account) GetAccountsByPrincipalID(principalID string) (*model.Accounts, error) {
	res, err := a.AwsDynamoDB.Query(&dynamodb.QueryInput{
		TableName: aws.String(a.TableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pid": {
				S: aws.String(string(principalID)),
			},
		},
		KeyConditionExpression: aws.String("PrincipalId = :pid"),
		ConsistentRead:         aws.Bool(a.ConsistentRead),
	})
	if err != nil {
		return nil, fmt.Errorf("error gettings accounts by principal ID %s: %s: %w", principalID, err, errors.ErrInternalServer)
	}

	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	return accounts, err
}

// GetAccounts Get a list of accounts based on Principal ID
func (a *Account) GetAccounts() (*model.Accounts, error) {
	res, err := a.AwsDynamoDB.Scan(&dynamodb.ScanInput{
		TableName:      aws.String(a.TableName),
		ConsistentRead: aws.Bool(a.ConsistentRead),
	})
	if err != nil {
		return nil, fmt.Errorf("error gettings accounts: %s: %w", err, errors.ErrInternalServer)
	}
	accounts := &model.Accounts{}
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, accounts)
	return accounts, err
}
