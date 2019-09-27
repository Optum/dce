package db

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
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
	// Name of the RedboxLease table
	LeaseTableName string
}

// The DBer interface includes all methods used by the DB struct to interact with
// DynamoDB. This is useful if we want to mock the DB service.
type DBer interface {
	GetAccount(accountID string) (*RedboxAccount, error)
	GetReadyAccount() (*RedboxAccount, error)
	GetAccountsForReset() ([]*RedboxAccount, error)
	GetAccounts() ([]*RedboxAccount, error)
	FindAccountsByStatus(status AccountStatus) ([]*RedboxAccount, error)
	FindAccountsByPrincipalID(principalID string) ([]*RedboxAccount, error)
	PutAccount(account RedboxAccount) error
	DeleteAccount(accountID string) (*RedboxAccount, error)
	PutLease(account RedboxLease) (*RedboxLease, error)
	TransitionAccountStatus(accountID string, prevStatus AccountStatus, nextStatus AccountStatus) (*RedboxAccount, error)
	TransitionLeaseStatus(accountID string, principalID string, prevStatus LeaseStatus, nextStatus LeaseStatus, leaseStatusReason string) (*RedboxLease, error)
	FindLeasesByAccount(accountID string) ([]*RedboxLease, error)
	FindLeasesByPrincipal(principalID string) ([]*RedboxLease, error)
	FindLeasesByStatus(status LeaseStatus) ([]*RedboxLease, error)
	UpdateMetadata(accountID string, metadata map[string]interface{}) error
	UpdateAccountPrincipalPolicyHash(accountID string, prevHash string, nextHash string) (*RedboxAccount, error)
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

// GetAccounts returns a list of accounts from the table
// TODO implement pagination and query support
func (db *DB) GetAccounts() ([]*RedboxAccount, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(db.AccountTableName),
	}

	// Execute and verify the query
	resp, err := db.Client.Scan(input)
	if err != nil {
		return make([]*RedboxAccount, 0), err
	}

	// Return the Redbox Account
	accounts := []*RedboxAccount{}
	for _, r := range resp.Items {
		n, err := unmarshalAccount(r)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, n)
	}
	return accounts, nil
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

func (db *DB) FindAccountsByStatus(status AccountStatus) ([]*RedboxAccount, error) {
	res, err := db.Client.Query(&dynamodb.QueryInput{
		TableName: aws.String(db.AccountTableName),
		IndexName: aws.String("AccountStatus"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(string(status)),
			},
		},
		KeyConditionExpression: aws.String("AccountStatus = :status"),
	})

	accounts := []*RedboxAccount{}

	if err != nil {
		return accounts, err
	}

	for _, item := range res.Items {
		acct, err := unmarshalAccount(item)
		if err != nil {
			return accounts, err
		}
		accounts = append(accounts, acct)
	}

	return accounts, nil
}
func (db *DB) FindAccountsByPrincipalID(principalID string) ([]*RedboxAccount, error) {
	res, err := db.Client.Query(&dynamodb.QueryInput{
		TableName: aws.String(db.AccountTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pid": {
				S: aws.String(string(principalID)),
			},
		},
		KeyConditionExpression: aws.String("PrincipalId = :pid"),
	})

	accounts := []*RedboxAccount{}

	if err != nil {
		return accounts, err
	}

	for _, item := range res.Items {
		acct, err := unmarshalAccount(item)
		if err != nil {
			return accounts, err
		}
		accounts = append(accounts, acct)
	}

	return accounts, nil
}

// GetLease retrieves a Lease for the
// given accountID and principalID
func (db *DB) GetLease(accountID string, principalID string) (*RedboxLease, error) {
	result, err := db.Client.GetItem(
		&dynamodb.GetItemInput{
			TableName: aws.String(db.LeaseTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {

					S: aws.String(accountID),
				},
				"PrincipalId": {
					S: aws.String(principalID),
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

	return unmarshalLease(result.Item)
}

// FindLeasesByAccount finds lease values for a given accountID
func (db *DB) FindLeasesByAccount(accountID string) ([]*RedboxLease, error) {
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":a1": {
				S: aws.String(accountID),
			},
		},
		KeyConditionExpression: aws.String("AccountId = :a1"),
		TableName:              aws.String(db.LeaseTableName),
	}

	resp, err := db.Client.Query(input)
	if err != nil {
		return nil, err
	}

	var redboxes []*RedboxLease
	for _, r := range resp.Items {
		n, err := unmarshalLease(r)
		if err != nil {
			return nil, err
		}
		redboxes = append(redboxes, n)
	}

	return redboxes, nil
}

//FindLeasesByPrincipal finds leased accounts for a given principalID
func (db *DB) FindLeasesByPrincipal(principalID string) ([]*RedboxLease, error) {
	input := &dynamodb.QueryInput{
		IndexName: aws.String("PrincipalId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":u1": {
				S: aws.String(principalID),
			},
		},
		KeyConditionExpression: aws.String("PrincipalId = :u1"),
		TableName:              aws.String(db.LeaseTableName),
	}

	resp, err := db.Client.Query(input)
	if err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, nil
	}

	fmt.Println(resp)

	var redboxes []*RedboxLease
	for _, r := range resp.Items {
		n, err := unmarshalLease(r)
		if err != nil {
			return nil, err
		}
		redboxes = append(redboxes, n)
	}

	return redboxes, nil
}

func (db *DB) FindLeasesByStatus(status LeaseStatus) ([]*RedboxLease, error) {
	res, err := db.Client.Query(&dynamodb.QueryInput{
		IndexName: aws.String("LeaseStatus"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {
				S: aws.String(string(status)),
			},
		},
		KeyConditionExpression: aws.String("LeaseStatus = :status"),
		TableName:              aws.String(db.LeaseTableName),
	})

	leases := []*RedboxLease{}

	if err != nil {
		return leases, err
	}

	for _, item := range res.Items {
		lease, err := unmarshalLease(item)
		if err != nil {
			return leases, err
		}
		leases = append(leases, lease)
	}

	return leases, nil
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

// PutLease writes an Lease to DynamoDB
// Returns the previous AccountsLease if there is one - does not return
// the lease that was added
func (db *DB) PutLease(lease RedboxLease) (
	*RedboxLease, error) {
	item, err := dynamodbattribute.MarshalMap(lease)
	if err != nil {
		return nil, err
	}

	result, err := db.Client.PutItem(
		&dynamodb.PutItemInput{
			TableName: aws.String(db.LeaseTableName),
			Item:      item,
		},
	)
	if err != nil {
		return nil, err
	}
	return unmarshalLease(result.Attributes)
}

// TransitionLeaseStatus updates a lease's status from prevStatus to nextStatus.
// Will fail if the Lease was not previously set to `prevStatus`
//
// For example, to set a ResetLock on an account, you could call:
//		db.TransitionLeaseStatus(accountId, principalID, Active, ResetLock)
//
// And to unlock the account:
//		db.TransitionLeaseStatus(accountId, principalID, ResetLock, Active)
func (db *DB) TransitionLeaseStatus(accountID string, principalID string, prevStatus LeaseStatus, nextStatus LeaseStatus, leaseStatusReason string) (*RedboxLease, error) {
	result, err := db.Client.UpdateItem(
		&dynamodb.UpdateItemInput{
			// Query in Lease Table
			TableName: aws.String(db.LeaseTableName),
			// Find Lease for the requested accountId
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: aws.String(accountID),
				},
				"PrincipalId": {
					S: aws.String(principalID),
				},
			},
			// Set Status="Active"
			UpdateExpression: aws.String("set LeaseStatus=:nextStatus, " +
				"LastModifiedOn=:lastModifiedOn, " + "LeaseStatusModifiedOn=:leaseStatusModifiedOn"),
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
				":leaseStatusModifiedOn": {
					N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
				},
			},
			// Only update locked records
			ConditionExpression: aws.String("LeaseStatus = :prevStatus"),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
		},
	)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "ConditionalCheckFailedException" {
				return nil, &StatusTransitionError{
					fmt.Sprintf(
						"unable to update lease status from \"%v\" to \"%v\" for %v/%v: no lease exists with Status=\"%v\"",
						prevStatus,
						nextStatus,
						accountID,
						principalID,
						prevStatus,
					),
				}
			}
		}
		return nil, err
	}

	return unmarshalLease(result.Attributes)
}

// TransitionAccountStatus updates account status for a given accountID and
// returns the updated record on success
func (db *DB) TransitionAccountStatus(accountID string, prevStatus AccountStatus, nextStatus AccountStatus) (*RedboxAccount, error) {
	result, err := db.Client.UpdateItem(
		&dynamodb.UpdateItemInput{
			// Query in Lease Table
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

// UpdateAccountPrincipalPolicyHash updates hash representing the
// current version of the Principal IAM Policy applied to the acount
func (db *DB) UpdateAccountPrincipalPolicyHash(accountID string, prevHash string, nextHash string) (*RedboxAccount, error) {

	conditionExpression := expression.ConditionBuilder{}
	if prevHash != "" {
		log.Printf("Using Condition where PrincipalPolicyHash equals '%s'", prevHash)
		conditionExpression = expression.Name("PrincipalPolicyHash").Equal(expression.Value(prevHash))
	} else {
		log.Printf("Using Condition where PrincipalPolicyHash does not exists")
		conditionExpression = expression.AttributeNotExists(expression.Name("PrincipalPolicyHash"))
	}
	updateExpression, _ := expression.NewBuilder().WithCondition(
		conditionExpression,
	).WithUpdate(
		expression.Set(
			expression.Name("PrincipalPolicyHash"),
			expression.Value(nextHash),
		).Set(
			expression.Name("LastModifiedOn"),
			expression.Value(time.Now().Unix()),
		),
	).Build()

	result, err := db.Client.UpdateItem(
		&dynamodb.UpdateItemInput{
			// Query in Lease Table
			TableName: aws.String(db.AccountTableName),
			// Find Account for the requested accountId
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			ExpressionAttributeNames:  updateExpression.Names(),
			ExpressionAttributeValues: updateExpression.Values(),
			// Set PrincipalPolicyHash=nextHash
			UpdateExpression: updateExpression.Update(),
			// Only update records where the previousHash matches
			ConditionExpression: updateExpression.Condition(),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
		},
	)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "ConditionalCheckFailedException" {
				return nil, &StatusTransitionError{
					fmt.Sprintf(
						"unable to update Principal Policy hash from \"%v\" to \"%v\" "+
							"for account %v: no account exists with PrincipalPolicyHash=\"%v\"",
						prevHash,
						nextHash,
						accountID,
						prevHash,
					),
				}
			}
			return nil, err
		}
		return nil, err
	}

	return unmarshalAccount(result.Attributes)
}

// DeleteAccount finds a given account and deletes it if it is not of status `Leased`. Returns the account.
func (db *DB) DeleteAccount(accountID string) (*RedboxAccount, error) {
	account, err := db.GetAccount(accountID)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query account \"%s\": %s.", accountID, err)
		log.Print(errorMessage)
		return nil, errors.New(errorMessage)
	}

	if account == nil {
		errorMessage := fmt.Sprintf("No account found with ID \"%s\".", accountID)
		log.Print(errorMessage)
		return nil, &AccountNotFoundError{err: errorMessage}
	}

	if account.AccountStatus == Leased {
		errorMessage := fmt.Sprintf("Unable to delete account \"%s\": account is leased.", accountID)
		log.Print(errorMessage)
		return account, &AccountLeasedError{err: errorMessage}
	}

	input := &dynamodb.DeleteItemInput{
		TableName: &db.AccountTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"Id": {
				S: aws.String(accountID),
			},
		},
	}

	_, err = db.Client.DeleteItem(input)
	return account, err
}

// UpdateMetadata updates the metadata field of an account, overwriting the old value completely with a new one
func (db *DB) UpdateMetadata(accountID string, metadata map[string]interface{}) error {
	serialized, err := dynamodbattribute.Marshal(metadata)

	if err != nil {
		log.Printf("Failed to marshall metadata map for updating account %s: %s", accountID, err)
		return err
	}

	_, err = db.Client.UpdateItem(
		&dynamodb.UpdateItemInput{
			TableName: aws.String(db.AccountTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			UpdateExpression: aws.String("set Metadata=:metadata, " +
				"LastModifiedOn=:lastModifiedOn"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":metadata": serialized,
				":lastModifiedOn": {
					N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
				},
			},
		},
	)

	if err != nil {
		log.Printf("Failed to execute metadata update for account %s: %s", accountID, err)
		return err
	}

	return nil
}

func unmarshalAccount(dbResult map[string]*dynamodb.AttributeValue) (*RedboxAccount, error) {
	redboxAccount := RedboxAccount{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &redboxAccount)

	if err != nil {
		return nil, err
	}

	return &redboxAccount, nil
}

func unmarshalLease(dbResult map[string]*dynamodb.AttributeValue) (*RedboxLease, error) {
	redboxLease := RedboxLease{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &redboxLease)
	if err != nil {
		return nil, err
	}

	return &redboxLease, nil
}

// New creates a new DB Service struct,
// with all the necessary fields configured.
//
// This method is mostly useful for testing, as it gives
// you fine-grained control over how the service is configured.
//
// Elsewhere, you should generally use `db.NewFromEnv()`
//
func New(client *dynamodb.DynamoDB, accountTableName string, leaseTableName string) *DB {
	return &DB{
		Client:           client,
		AccountTableName: accountTableName,
		LeaseTableName:   leaseTableName,
	}
}

/*
NewFromEnv creates a DB instance configured from environment variables.
Requires env vars for:

- AWS_CURRENT_REGION
- ACCOUNT_DB
- LEASE_DB
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
		common.RequireEnv("LEASE_DB"),
	), nil
}
