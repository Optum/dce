package account

import (
	"encoding/json"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Data - Handles importing and exporting Accounts and non-exported Properties
type Data struct {
	ID                  *string                `json:"id,omitempty" dynamodbav:"Id"`                                             // AWS Account ID
	Status              *Status                `json:"accountStatus,omitempty" dynamodbav:"AccountStatus,omitempty"`             // Status of the AWS Account
	LastModifiedOn      *int64                 `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn"`                     // Last Modified Epoch Timestamp
	CreatedOn           *int64                 `json:"createdOn,omitempty"  dynamodbav:"CreatedOn,omitempty"`                    // Account CreatedOn
	AdminRoleArn        *string                `json:"adminRoleArn,omitempty"  dynamodbav:"AdminRoleArn"`                        // Assumed by the master account, to manage this user account
	PrincipalRoleArn    *string                `json:"principalRoleArn,omitempty"  dynamodbav:"PrincipalRoleArn,omitempty"`      // Assumed by principal users
	PrincipalPolicyHash *string                `json:"principalPolicyHash,omitempty" dynamodbav:"PrincipalPolicyHash,omitempty"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"metadata,omitempty"  dynamodbav:"Metadata,omitempty"`                      // Any org specific metadata pertaining to the account
}

// Status is an account status type
type Status string

const (
	// AccountStatusNone status
	AccountStatusNone Status = "None"
	// AccountStatusReady status
	AccountStatusReady Status = "Ready"
	// AccountStatusNotReady status
	AccountStatusNotReady Status = "NotReady"
	// AccountStatusLeased status
	AccountStatusLeased Status = "Leased"
	// AccountStatusOrphaned status
	AccountStatusOrphaned Status = "Orphaned"
)

// String returns the string value of AccountStatus
func (c Status) String() string {
	return string(c)
}

// StringPtr returns a pointer to the string value of AccountStatus
func (c Status) StringPtr() *string {
	v := string(c)
	return &v
}

// StatusPtr returns a pointer to the string value of AccountStatus
func (c Status) StatusPtr() *Status {
	v := c
	return &v
}

// Writer put an item into the data store
type Writer interface {
	WriteAccount(i *Account, lastModifiedOn *int64) error
}

// Deleter Deletes an Account from the data store
type Deleter interface {
	DeleteAccount(i *Account) error
}

// SingleReader Reads Account information from the data store
type SingleReader interface {
	GetAccountByID(ID string, account *Account) error
}

// MultipleReader reads multiple accounts from the data store
type MultipleReader interface {
	GetAccounts(query *Account, accounts *Accounts) error
}

// Reader data Layer
type Reader interface {
	SingleReader
	MultipleReader
}

// WriterDeleter data layer
type WriterDeleter interface {
	Writer
	Deleter
}

// ReaderWriterDeleter includes Reader and Writer interfaces
type ReaderWriterDeleter interface {
	Reader
	WriterDeleter
}

// Eventer for publishing events
type Eventer interface {
	Publish() error
}

// Manager manages all the actions against an account
type Manager interface {
	Setup(adminRole string) error
}

// Account is a type corresponding to a Account table record
type Account struct {
	writer WriterDeleter
	data   Data
}

// ID Returns the Account ID
func (a *Account) ID() *string {
	return a.data.ID
}

// Status Returns the Account ID
func (a *Account) Status() *Status {
	return a.data.Status
}

// AdminRoleArn Returns the Admin Role Arn
func (a *Account) AdminRoleArn() *string {
	return a.data.AdminRoleArn
}

// PrincipalRoleArn Returns the Principal Role Arn
func (a *Account) PrincipalRoleArn() *string {
	return a.data.PrincipalRoleArn
}

// PrincipalPolicyHash Returns the Principal Role Hash
func (a *Account) PrincipalPolicyHash() *string {
	return a.data.PrincipalPolicyHash
}

// Metadata Returns the Principal Role Hash
func (a *Account) Metadata() map[string]interface{} {
	return a.data.Metadata
}

func (a *Account) save() error {
	var lastModifiedOn *int64
	now := time.Now().Unix()
	if a.data.LastModifiedOn == nil {
		lastModifiedOn = nil
		a.data.CreatedOn = &now
		a.data.LastModifiedOn = &now
	} else {
		lastModifiedOn = a.data.LastModifiedOn
		a.data.LastModifiedOn = &now
	}

	err := a.Validate()
	if err != nil {
		return err
	}
	err = a.writer.WriteAccount(a, lastModifiedOn)
	if err != nil {
		return err
	}
	return nil
}

// Validate the account data
func (a *Account) Validate() error {
	err := validation.ValidateStruct(&a.data,
		validation.Field(&a.data.AdminRoleArn, validateAdminRoleArn...),
		validation.Field(&a.data.ID, validateID...),
		validation.Field(&a.data.LastModifiedOn, validateInt64...),
		validation.Field(&a.data.Status, validateStatus...),
		validation.Field(&a.data.CreatedOn, validateInt64...),
		validation.Field(&a.data.PrincipalRoleArn, validatePrincipalRoleArn...),
		validation.Field(&a.data.PrincipalPolicyHash, validatePrincipalPolicyHash...),
	)
	if err != nil {
		return errors.NewValidation("account", err)
	}
	return nil
}

// Update the Account record in DynamoDB
func (a *Account) Update(d Account, am Manager) error {
	err := validation.ValidateStruct(&d.data,
		// ID has to be empty
		validation.Field(&d.data.ID, validation.NilOrNotEmpty, validation.In(*a.data.ID)),
		validation.Field(&d.data.AdminRoleArn, validation.By(isNilOrUsableAdminRole(am))),
		validation.Field(&d.data.ID, validation.By(isNil)),
		validation.Field(&d.data.LastModifiedOn, validation.By(isNil)),
		validation.Field(&d.data.Status, validation.By(isNil)),
		validation.Field(&d.data.CreatedOn, validation.By(isNil)),
		validation.Field(&d.data.PrincipalRoleArn, validation.By(isNil)),
		validation.Field(&d.data.PrincipalPolicyHash, validation.By(isNil)),
	)
	if err != nil {
		return errors.NewValidation("account", err)
	}

	if d.data.AdminRoleArn != nil {
		a.data.AdminRoleArn = d.data.AdminRoleArn
	}
	if d.data.Metadata != nil {
		a.data.Metadata = d.data.Metadata
	}

	err = a.save()
	if err != nil {
		return err
	}
	return nil
}

// Delete finds a given account and deletes it if it is not of status `Leased`. Returns the account.
func (a *Account) Delete() error {

	err := validation.ValidateStruct(&a.data,
		validation.Field(&a.data.Status, validation.NotNil, validation.By(isAccountNotLeased)),
	)
	if err != nil {
		return errors.NewConflict("account", *a.data.ID, err)
	}

	err = a.writer.DeleteAccount(a)
	if err != nil {
		return err
	}

	return nil
}

// GetAccountByID returns an account from ID
func GetAccountByID(ID string, d SingleReader, wd WriterDeleter) (*Account, error) {

	account := &Account{}
	err := d.GetAccountByID(ID, account)
	if err != nil {
		return nil, err
	}

	account.writer = wd

	return account, err
}

// New returns an account from ID
func New(wd WriterDeleter, data Data) *Account {
	now := time.Now().Unix()

	new := &Account{
		writer: wd,
		data:   data,
	}
	if new.data.CreatedOn == nil {
		new.data.CreatedOn = &now
	}
	if new.data.LastModifiedOn == nil {
		new.data.LastModifiedOn = &now
	}

	return new
}

// MarshalJSON Marshals the data inside the account
func (a *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.data)
}

// UnmarshalJSON Unmarshals the data inside the account
func (a *Account) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &a.data); err != nil {
		return errors.NewInternalServer("unable to unmarshal account", err)
	}
	return nil
}

// UnmarshalDynamoDBAttributeValue Unmarshals the data inside the account
func (a *Account) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if err := dynamodbattribute.Unmarshal(av, &a.data); err != nil {
		return errors.NewInternalServer("unable to unmarshal account", err)
	}
	return nil
}

// MarshalDynamoDBAttributeValue Marshals the data inside the account
func (a *Account) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	data := a.data
	newAv, err := dynamodbattribute.Marshal(&data)
	if err != nil {
		return errors.NewInternalServer("unable to unmarshal account", err)
	}

	*av = *newAv
	return nil
}
