package db

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	guuid "github.com/google/uuid"

	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

/*
The `DB` service abstracts all interactions with the DynamoDB tables
*/

// DB contains DynamoDB client and table names
type DB struct {
	// DynamoDB Client
	Client *dynamodb.DynamoDB
	// Name of the Account table
	AccountTableName string
	// Name of the Lease table
	LeaseTableName string
	// Default expiry time, in days, of the lease
	DefaultLeaseLengthInDays int
	// Use Consistent Reads when scanning or querying when possible.
	ConsistentRead bool
}

// The DBer interface includes all methods used by the DB struct to interact with
// DynamoDB. This is useful if we want to mock the DB service.
type DBer interface {
	GetAccount(accountID string) (*Account, error)
	GetReadyAccount() (*Account, error)
	GetAccounts() ([]*Account, error)
	GetLeases(input GetLeasesInput) (GetLeasesOutput, error)
	GetLeaseByID(leaseID string) (*Lease, error)
	FindAccountsByStatus(status AccountStatus) ([]*Account, error)
	FindAccountsByPrincipalID(principalID string) ([]*Account, error)
	PutAccount(account Account) error
	DeleteAccount(accountID string) (*Account, error)
	PutLease(account Lease) (*Lease, error)
	TransitionAccountStatus(accountID string, prevStatus AccountStatus, nextStatus AccountStatus) (*Account, error)
	TransitionLeaseStatus(accountID string, principalID string, prevStatus LeaseStatus, nextStatus LeaseStatus, leaseStatusReason LeaseStatusReason) (*Lease, error)
	FindLeasesByAccount(accountID string) ([]*Lease, error)
	FindLeasesByPrincipal(principalID string) ([]*Lease, error)
	FindLeasesByStatus(status LeaseStatus) ([]*Lease, error)
	UpdateMetadata(accountID string, metadata map[string]interface{}) error
	UpdateAccountPrincipalPolicyHash(accountID string, prevHash string, nextHash string) (*Account, error)
}

// GetAccount returns an account record corresponding to an accountID
// string.
func (db *DB) GetAccount(accountID string) (*Account, error) {
	result, err := db.Client.GetItem(
		&dynamodb.GetItemInput{
			TableName: aws.String(db.AccountTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			ConsistentRead: aws.Bool(db.ConsistentRead),
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
func (db *DB) GetAccounts() ([]*Account, error) {
	input := &dynamodb.ScanInput{
		TableName:      aws.String(db.AccountTableName),
		ConsistentRead: aws.Bool(db.ConsistentRead),
	}

	// Execute and verify the query
	resp, err := db.Client.Scan(input)
	if err != nil {
		return make([]*Account, 0), err
	}

	// Return the Account
	accounts := []*Account{}
	for _, r := range resp.Items {
		n, err := unmarshalAccount(r)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, n)
	}
	return accounts, nil
}

// GetReadyAccount returns an available account record with a
// corresponding status of 'Ready'
func (db *DB) GetReadyAccount() (*Account, error) {
	accounts, err := db.FindAccountsByStatus(Ready)
	if len(accounts) < 1 {
		return nil, err
	}
	return accounts[0], err
}

func (db *DB) FindAccountsByStatus(status AccountStatus) ([]*Account, error) {
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

	accounts := []*Account{}

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
func (db *DB) FindAccountsByPrincipalID(principalID string) ([]*Account, error) {
	res, err := db.Client.Query(&dynamodb.QueryInput{
		TableName: aws.String(db.AccountTableName),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pid": {
				S: aws.String(string(principalID)),
			},
		},
		KeyConditionExpression: aws.String("PrincipalId = :pid"),
		ConsistentRead:         aws.Bool(db.ConsistentRead),
	})

	accounts := []*Account{}

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

// GetLeaseByID gets a lease by ID
func (db *DB) GetLeaseByID(leaseID string) (*Lease, error) {

	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":a1": {
				S: aws.String(leaseID),
			},
		},
		KeyConditionExpression: aws.String("Id = :a1"),
		TableName:              aws.String(db.LeaseTableName),
		IndexName:              aws.String("LeaseId"),
	}

	resp, err := db.Client.Query(input)
	if err != nil {
		return nil, err
	}

	if len(resp.Items) < 1 {
		return nil, fmt.Errorf("No Lease found with id: %s", leaseID)
	}
	if len(resp.Items) > 1 {
		return nil, fmt.Errorf("Found more than one Lease with id: %s", leaseID)
	}

	return unmarshalLease(resp.Items[0])
}

// GetLease retrieves a Lease for the
// given accountID and principalID
func (db *DB) GetLease(accountID string, principalID string) (*Lease, error) {
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
			ConsistentRead: aws.Bool(db.ConsistentRead),
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
func (db *DB) FindLeasesByAccount(accountID string) ([]*Lease, error) {
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":a1": {
				S: aws.String(accountID),
			},
		},
		KeyConditionExpression: aws.String("AccountId = :a1"),
		TableName:              aws.String(db.LeaseTableName),
		ConsistentRead:         aws.Bool(db.ConsistentRead),
	}

	resp, err := db.Client.Query(input)
	if err != nil {
		return nil, err
	}

	var leases []*Lease
	for _, r := range resp.Items {
		n, err := unmarshalLease(r)
		if err != nil {
			return nil, err
		}
		leases = append(leases, n)
	}

	return leases, nil
}

// FindLeasesByPrincipal finds leased accounts for a given principalID
func (db *DB) FindLeasesByPrincipal(principalID string) ([]*Lease, error) {
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

	var leases []*Lease
	for _, r := range resp.Items {
		n, err := unmarshalLease(r)
		if err != nil {
			return nil, err
		}
		leases = append(leases, n)
	}

	return leases, nil
}

func (db *DB) FindLeasesByStatus(status LeaseStatus) ([]*Lease, error) {
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

	leases := []*Lease{}

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

// PutAccount stores an account in DynamoDB
func (db *DB) PutAccount(account Account) error {
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
func (db *DB) PutLease(lease Lease) (
	*Lease, error) {

	// apply some reasonable DEFAULTS to the lease before saving it.
	if len(lease.ID) == 0 {
		lease.ID = guuid.New().String()
	}

	if lease.ExpiresOn == 0 {
		lease.ExpiresOn = time.Now().AddDate(0, 0, db.DefaultLeaseLengthInDays).Unix()
	}

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
func (db *DB) TransitionLeaseStatus(accountID string, principalID string, prevStatus LeaseStatus, nextStatus LeaseStatus, leaseStatusReason LeaseStatusReason) (*Lease, error) {
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
				"LeaseStatusReason=:nextStatusReason, " +
				"LastModifiedOn=:lastModifiedOn, " + "LeaseStatusModifiedOn=:leaseStatusModifiedOn"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":prevStatus": {
					S: aws.String(string(prevStatus)),
				},
				":nextStatus": {
					S: aws.String(string(nextStatus)),
				},
				":nextStatusReason": {
					S: aws.String(string(leaseStatusReason)),
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
func (db *DB) TransitionAccountStatus(accountID string, prevStatus AccountStatus, nextStatus AccountStatus) (*Account, error) {
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
// current version of the Principal IAM Policy applied to the account
func (db *DB) UpdateAccountPrincipalPolicyHash(accountID string, prevHash string, nextHash string) (*Account, error) {

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
func (db *DB) DeleteAccount(accountID string) (*Account, error) {
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

// GetLeasesInput contains the filtering criteria for the GetLeases scan.
type GetLeasesInput struct {
	StartKeys   map[string]string
	PrincipalID string
	AccountID   string
	Status      string
	Limit       int64
}

// GetLeasesOutput contains the scan results as well as the keys for retrieve the next page of the result set.
type GetLeasesOutput struct {
	Results  []*Lease
	NextKeys map[string]string
}

// GetLeases takes a set of filtering criteria and scans the Leases table for the matching records.
func (db *DB) GetLeases(input GetLeasesInput) (GetLeasesOutput, error) {
	limit := int64(25)
	filters := make([]string, 0)
	filterValues := make(map[string]*dynamodb.AttributeValue)

	if input.Limit > 0 {
		limit = input.Limit
	}

	scanInput := &dynamodb.ScanInput{
		TableName:      aws.String(db.LeaseTableName),
		Limit:          &limit,
		ConsistentRead: aws.Bool(db.ConsistentRead),
	}

	// Build the filter clauses.
	if input.Status != "" {
		filters = append(filters, "LeaseStatus = :status")
		filterValues[":status"] = &dynamodb.AttributeValue{S: aws.String(string(input.Status))}
	}

	if input.PrincipalID != "" {
		filters = append(filters, "PrincipalId = :principalId")
		filterValues[":principalId"] = &dynamodb.AttributeValue{S: aws.String(input.PrincipalID)}
	}

	if input.AccountID != "" {
		filters = append(filters, "AccountId = :accountId")
		filterValues[":accountId"] = &dynamodb.AttributeValue{S: aws.String(input.AccountID)}
	}

	if len(filters) > 0 {
		filterStatement := strings.Join(filters, " and ")
		scanInput.FilterExpression = &filterStatement
		scanInput.ExpressionAttributeValues = filterValues
	}

	if input.StartKeys != nil && len(input.StartKeys) > 0 {
		scanInput.ExclusiveStartKey = make(map[string]*dynamodb.AttributeValue)
		for k, v := range input.StartKeys {
			scanInput.ExclusiveStartKey[k] = &dynamodb.AttributeValue{S: aws.String(v)}
		}
	}

	output, err := db.Client.Scan(scanInput)

	// Parse the results and build the next keys if necessary.
	if err != nil {
		return GetLeasesOutput{}, err
	}

	results := make([]*Lease, 0)

	for _, o := range output.Items {
		lease, err := unmarshalLease(o)
		if err != nil {
			return GetLeasesOutput{}, err
		}
		results = append(results, lease)
	}

	nextKey := make(map[string]string)

	for k, v := range output.LastEvaluatedKey {
		nextKey[k] = *v.S
	}

	return GetLeasesOutput{
		Results:  results,
		NextKeys: nextKey,
	}, nil
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

func unmarshalAccount(dbResult map[string]*dynamodb.AttributeValue) (*Account, error) {
	account := Account{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &account)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func unmarshalLease(dbResult map[string]*dynamodb.AttributeValue) (*Lease, error) {
	lease := Lease{}
	err := dynamodbattribute.UnmarshalMap(dbResult, &lease)
	if err != nil {
		return nil, err
	}

	return &lease, nil
}

// New creates a new DB Service struct,
// with all the necessary fields configured.
//
// This method is mostly useful for testing, as it gives
// you fine-grained control over how the service is configured.
//
// Elsewhere, you should generally use `db.NewFromEnv()`
//
func New(client *dynamodb.DynamoDB, accountTableName string, leaseTableName string, defaultLeaseLengthInDays int) *DB {
	return &DB{
		Client:                   client,
		AccountTableName:         accountTableName,
		LeaseTableName:           leaseTableName,
		DefaultLeaseLengthInDays: defaultLeaseLengthInDays,
		ConsistentRead:           false,
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
		common.GetEnvInt("DEFAULT_LEASE_LENGTH_IN_DAYS", 7),
	), nil
}
