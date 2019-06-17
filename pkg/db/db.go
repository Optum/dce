package db

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/Optum/Redbox/pkg/common"
)

/*
The `DB` service abstracts all interactions
with the Redbox DynamoDB tables
*/

// DB contains DynamoDB client and table names
type DB struct {
	// DynamoDB Client
	Client *dynamodb.DynamoDB
	// Name of the RedboxAccount table
	AccountTableName string
	// Name of the RedboxAccountAssignment table
	AccountAssignmentTableName string
}

// The DBer interface includes all methods used by the DB struct to interact with
// DynamoDB. This is useful if we want to mock the DB service.
type DBer interface {
	GetAccount(accountID string) (*RedboxAccount, error)
	GetReadyAccount() (*RedboxAccount, error)
	GetAccountsForReset() ([]*RedboxAccount, error)
	PutAccount(account RedboxAccount) error
	PutAccountAssignment(account RedboxAccountAssignment) error
	TransitionAccountStatus(accountID string, prevStatus AccountStatus, nextStatus AccountStatus) (*RedboxAccount, error)
	TransitionAssignmentStatus(accountID string, userID string, prevStatus AssignmentStatus, nextStatus AssignmentStatus) (*RedboxAccountAssignment, error)
	FindAssignmentsByAccount(accountID string) ([]*RedboxAccountAssignment, error)
	FindAssignmentByUser(userID string) ([]*RedboxAccountAssignment, error)
}

// GetAccount returns a Redbox account record corresponding to an accountID
// string.
func (db *DB) GetAccount(accountID string) (*RedboxAccount, error) {
	result, err := db.Client.GetItem(
		&dynamodb.GetItemInput{
			TableName: aws.String(db.AccountTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	return unmarshalAccount(result.Item)
}

// GetReadyAccount returns an available Redbox account record with a
// corresponding status of 'Ready'
func (db *DB) GetReadyAccount() (*RedboxAccount, error) {
	// Construct the query to only grab the first available account
	input := &dynamodb.ScanInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":acctstatus": {
				S: aws.String("Ready"),
			},
		},
		FilterExpression: aws.String("AccountStatus = :acctstatus"),
		TableName:        aws.String(db.AccountTableName),
	}

	// Make and verify the query
	resp, err := db.Client.Scan(input)
	if err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, nil
	}

	// Return the Redbox Account
	return unmarshalAccount(resp.Items[0])
}

// GetAccountsForReset returns an array of Redbox account records available to
// be Reset
func (db *DB) GetAccountsForReset() ([]*RedboxAccount, error) {
	// Build the query input parameters
	params := &dynamodb.ScanInput{
		TableName: aws.String(db.AccountTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":acctstatus": {
				S: aws.String("Ready"),
			},
		},
		FilterExpression: aws.String("AccountStatus <> :acctstatus"),
	}

	// Make the DynamoDB Query API call
	// Warning: this could potentially be an expensive operation if the
	// database size becomes too big for the key/value nature of DynamoDB!
	resp, err := db.Client.Scan(params)
	if err != nil {
		return nil, err
	}

	// Create the array of RedboxAccounts
	redboxes := []*RedboxAccount{}
	for _, r := range resp.Items {
		n, err := unmarshalAccount(r)
		if err != nil {
			return nil, err
		}
		redboxes = append(redboxes, n)
	}
	return redboxes, nil
}

// GetAssignment retrieves a AccountAssignment for the
// given accountID and userID
func (db *DB) GetAssignment(accountID string, userID string) (*RedboxAccountAssignment, error) {
	result, err := db.Client.GetItem(
		&dynamodb.GetItemInput{
			TableName: aws.String(db.AccountAssignmentTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {

					S: aws.String(accountID),
				},
				"UserId": {
					S: aws.String(userID),
				},
			},
		},
	)

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	return unmarshalAccountAssignment(result.Item)
}

// FindAssignmentsByAccount finds assignment values for a given accountID
func (db *DB) FindAssignmentsByAccount(accountID string) ([]*RedboxAccountAssignment, error) {
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":a1": {
				S: aws.String(accountID),
			},
		},
		KeyConditionExpression: aws.String("AccountId = :a1"),
		TableName:              aws.String(db.AccountAssignmentTableName),
	}

	resp, err := db.Client.Query(input)
	if err != nil {
		return nil, err
	}

	var redboxes []*RedboxAccountAssignment
	for _, r := range resp.Items {
		n, err := unmarshalAccountAssignment(r)
		if err != nil {
			return nil, err
		}
		redboxes = append(redboxes, n)
	}

	return redboxes, nil
}

//FindAssignmentByUser finds assigned accounts for a given UserID
func (db *DB) FindAssignmentByUser(userID string) ([]*RedboxAccountAssignment, error) {
	input := &dynamodb.QueryInput{
		IndexName: aws.String("UserId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":u1": {
				S: aws.String(userID),
			},
		},
		KeyConditionExpression: aws.String("UserId = :u1"),
		TableName:              aws.String(db.AccountAssignmentTableName),
	}

	resp, err := db.Client.Query(input)
	if err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, nil
	}

	fmt.Println(resp)

	var redboxes []*RedboxAccountAssignment
	for _, r := range resp.Items {
		n, err := unmarshalAccountAssignment(r)
		if err != nil {
			return nil, err
		}
		redboxes = append(redboxes, n)
	}

	return redboxes, nil
}

// PutAccount stores a Redbox account in DynamoDB
func (db *DB) PutAccount(account RedboxAccount) error {
	item, err := dynamodbattribute.MarshalMap(account)
	if err != nil {
		return err
	}

	_, err = db.Client.PutItem(
		&dynamodb.PutItemInput{
			TableName: aws.String(db.AccountTableName),
			Item:      item,
		},
	)
	return err
}

// PutAccountAssignment writes an AccountAssignment to DynamoDB
func (db *DB) PutAccountAssignment(accountAssignment RedboxAccountAssignment) error {
	item, err := dynamodbattribute.MarshalMap(accountAssignment)
	if err != nil {
		return err
	}

	_, err = db.Client.PutItem(
		&dynamodb.PutItemInput{
			TableName: aws.String(db.AccountAssignmentTableName),
			Item:      item,
		},
	)
	return err
}

// TransitionAssignmentStatus updates an Assignment's status from prevStatus to nextStatus.
// Will fail if the Assignment was not previously set to `prevStatus`
//
// For example, to set a ResetLock on an account, you could call:
//		db.TransitionAssignmentStatus(accountId, userId, Active, ResetLock)
//
// And to unlock the account:
//		db.TransitionAssignmentStatus(accountId, userId, ResetLock, Active)
func (db *DB) TransitionAssignmentStatus(accountID string, userID string, prevStatus AssignmentStatus, nextStatus AssignmentStatus) (*RedboxAccountAssignment, error) {
	result, err := db.Client.UpdateItem(
		&dynamodb.UpdateItemInput{
			// Query in Assignment Table
			TableName: aws.String(db.AccountAssignmentTableName),
			// Find Assignment for the requested accountId
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: aws.String(accountID),
				},
				"UserId": {
					S: aws.String(userID),
				},
			},
			// Set Status="Active"
			UpdateExpression: aws.String("set AssignmentStatus=:nextStatus, " +
				"LastModifiedOn=:lastModifiedOn"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":prevStatus": {
					S: aws.String(string(prevStatus)),
				},
				":nextStatus": {
					S: aws.String(string(nextStatus)),
				},
				":lastModifiedOn": {
					N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
				},
			},
			// Only update locked records
			ConditionExpression: aws.String("AssignmentStatus = :prevStatus"),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
		},
	)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "ConditionalCheckFailedException" {
				return nil, &StatusTransitionError{
					fmt.Sprintf(
						"unable to update assignment status from \"%v\" to \"%v\" for %v/%v: no assignment exists with Status=\"%v\"",
						prevStatus,
						nextStatus,
						accountID,
						userID,
						prevStatus,
					),
				}
			}
		}
		return nil, err
	}

	return unmarshalAccountAssignment(result.Attributes)
}

// TransitionAccountStatus updates account status for a given accountID and
// returns the updated record on success
func (db *DB) TransitionAccountStatus(accountID string, prevStatus AccountStatus, nextStatus AccountStatus) (*RedboxAccount, error) {
	result, err := db.Client.UpdateItem(
		&dynamodb.UpdateItemInput{
			// Query in Assignment Table
			TableName: aws.String(db.AccountTableName),
			// Find Account for the requested accountId
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			// Set Status=nextStatus ("READY")
			UpdateExpression: aws.String("set AccountStatus=:nextStatus, " +
				"LastModifiedOn=:lastModifiedOn"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":prevStatus": {
					S: aws.String(string(prevStatus)),
				},
				":nextStatus": {
					S: aws.String(string(nextStatus)),
				},
				":lastModifiedOn": {
					N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
				},
			},
			// Only update locked records
			ConditionExpression: aws.String("AccountStatus = :prevStatus"),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
		},
	)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "ConditionalCheckFailedException" {
				return nil, &StatusTransitionError{
					fmt.Sprintf(
						"unable to update account status from \"%v\" to \"%v\" "+
							"for account %v: no account exists with Status=\"%v\"",
						prevStatus,
						nextStatus,
						accountID,
						prevStatus,
					),
				}
			}
		}
		return nil, err
	}

	return unmarshalAccount(result.Attributes)
}

func unmarshalAccount(dbResult map[string]*dynamodb.AttributeValue) (*RedboxAccount, error) {
	redboxAccount := RedboxAccount{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &redboxAccount)

	if err != nil {
		return nil, err
	}

	return &redboxAccount, nil
}

func unmarshalAccountAssignment(dbResult map[string]*dynamodb.AttributeValue) (*RedboxAccountAssignment, error) {
	redboxAssignment := RedboxAccountAssignment{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &redboxAssignment)
	if err != nil {
		return nil, err
	}

	return &redboxAssignment, nil
}

// New creates a new DB Service struct,
// with all the necessary fields configured.
//
// This method is mostly useful for testing, as it gives
// you fine-grained control over how the service is configured.
//
// Elsewhere, you should generally use `db.NewFromEnv()`
//
func New(client *dynamodb.DynamoDB, accountTableName string, accountAssignmentTableName string) *DB {
	return &DB{
		Client:                     client,
		AccountTableName:           accountTableName,
		AccountAssignmentTableName: accountAssignmentTableName,
	}
}

/*
NewFromEnv creates a DB instance configured from environment variables.
Requires env vars for:

- AWS_CURRENT_REGION
- ACCOUNT_DB
- ASSIGNMENT_DB
*/
func NewFromEnv() (*DB, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(common.RequireEnv("AWS_CURRENT_REGION")),
		),
		common.RequireEnv("ACCOUNT_DB"),
		common.RequireEnv("ASSIGNMENT_DB"),
	), nil
}
