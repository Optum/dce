package account

import (
	"encoding/json"
	"fmt"
	"time"

	gErrors "errors"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Updater put an item into the data store
type Updater interface {
	Update(input *model.Account, lastModifiedOn *int64) error
}

// Deleter Deletes an Account from the data store
type Deleter interface {
	Delete(inptut *model.Account) error
}

// SingleReader Reads Account information from the data store
type SingleReader interface {
	GetAccountByID(accountID string, account *model.Account) error
}

// MultipleReader reads multiple accounts from the data store
type MultipleReader interface {
	GetAccounts(*model.Account) (*model.Accounts, error)
	GetAccountsByStatus(status string) (*model.Accounts, error)
}

// Reader data Layer
type Reader interface {
	SingleReader
	MultipleReader
}

// Writer data layer
type Writer interface {
	Updater
	Deleter
}

// ReaderUpdater includes Reader and Writer interfaces
type ReaderUpdater interface {
	Reader
	Updater
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
	data model.Account
}

// ID Returns the Account ID
func (a *Account) ID() *string {
	return a.data.ID
}

// Status Returns the Account ID
func (a *Account) Status() *model.AccountStatus {
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

func (a *Account) updateStatus(nextStatus model.AccountStatus) {
	*a.data.Status = nextStatus
}

// UpdateStatus updates account status for a given accountID and
// returns the updated record on success
func (a *Account) UpdateStatus(nextStatus model.AccountStatus, u Updater) error {

	a.updateStatus(nextStatus)
	err := a.save(u)
	if err != nil {
		return fmt.Errorf(
			"unable to update account status from \"%v\" to \"%v\" on account %s: %w",
			*a.data.Status,
			nextStatus,
			*a.data.ID,
			err,
		)
	}

	return nil
}

func (a *Account) save(u Updater) error {
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
	err = u.Update(&a.data, lastModifiedOn)
	if err != nil {
		return err
	}
	return nil
}

// Validate the account data
func (a *Account) Validate() error {
	err := validation.ValidateStruct(&a.data,
		validation.Field(&a.data.AdminRoleArn, validation.NotNil),
		validation.Field(&a.data.ID, validation.NotNil),
		validation.Field(&a.data.LastModifiedOn, validation.NotNil),
		validation.Field(&a.data.Status, validation.NotNil),
		validation.Field(&a.data.CreatedOn, validation.NotNil),
		validation.Field(&a.data.PrincipalRoleArn, validation.NilOrNotEmpty),
		validation.Field(&a.data.PrincipalPolicyHash, validation.NilOrNotEmpty),
	)
	if err != nil {
		return &errors.ErrValidation{
			Message: fmt.Sprintf("update validation error: %v", err),
			Err:     err,
		}
	}
	return nil
}

// Update the Account record in DynamoDB
func (a *Account) Update(d model.Account, u Updater, am Manager) error {
	err := validation.ValidateStruct(&d,
		// ID has to be empty
		validation.Field(&d.ID, validation.NilOrNotEmpty, validation.In(*a.data.ID)),
		validation.Field(&d.AdminRoleArn, validation.By(isNilOrUsableAdminRole(am))),
		//validation.Field(&d.AdminRoleArn, validation.By(isNil)),
		validation.Field(&d.ID, validation.By(isNil)),
		validation.Field(&d.LastModifiedOn, validation.By(isNil)),
		validation.Field(&d.Status, validation.By(isNil)),
		validation.Field(&d.CreatedOn, validation.By(isNil)),
		validation.Field(&d.PrincipalRoleArn, validation.By(isNil)),
		validation.Field(&d.PrincipalPolicyHash, validation.By(isNil)),
	)
	if err != nil {
		return &errors.ErrValidation{
			Message: fmt.Sprintf("update validation error: %s", err.Error()),
			Err:     err,
		}
	}

	if d.AdminRoleArn != nil {
		a.data.AdminRoleArn = d.AdminRoleArn
	}
	if d.Metadata != nil {
		a.data.Metadata = d.Metadata
	}

	err = a.save(u)
	if err != nil {
		return fmt.Errorf("unable to update account %s: %w", *a.data.ID, err)
	}
	return nil
}

// Delete finds a given account and deletes it if it is not of status `Leased`. Returns the account.
func (a *Account) Delete(d Deleter) error {

	err := validation.ValidateStruct(&a.data,
		validation.Field(&a.data.Status, validation.NotNil, validation.By(isAccountLeased)),
	)
	if err != nil {
		return fmt.Errorf("unable to delete account \"%s\": %w", *a.data.ID, errors.ErrAccountIsLeased)
	}

	err = d.Delete(&a.data)
	if err != nil {
		return fmt.Errorf("unable to delete account \"%s\": %w", *a.data.ID, err)
	}

	return nil
}

// OrphanAccount - Orpahn an account
func (a *Account) OrphanAccount() error {
	return nil
}

// GetAccountByID returns an account from ID
func GetAccountByID(ID string, d SingleReader) (*Account, error) {

	newAccount := Account{}
	err := d.GetAccountByID(ID, &newAccount.data)

	return &newAccount, err
}

// New returns an account from ID
func New(data model.Account) *Account {
	now := time.Now().Unix()
	account := &Account{
		data: data,
	}
	account.data.CreatedOn = &now
	account.data.LastModifiedOn = &now
	return account
}

// MarshalJSON Marshals the data inside the account
func (a *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.data)
}

// GetReadyAccount returns an available account record with a
// corresponding status of 'Ready'
func GetReadyAccount(d Reader) (*Account, error) {
	accounts, err := GetAccountsByStatus(model.AccountStatus("Ready"), d)
	if err != nil {
		return nil, err
	}
	if len(*accounts) < 1 {
		return nil, errors.ErrNotFound
	}

	return &(*accounts)[0], err
}

// CreateAccount creates a new account
func CreateAccount(ID string, AdminRoleArn string, Metadata map[string]interface{}, d ReaderUpdater) (*Account, error) {
	// Check if the account already exists
	existingAccount, err := GetAccountByID(ID, d)
	if !gErrors.Is(err, errors.ErrNotFound) {
		if err != nil {
			return nil, fmt.Errorf("Failed to add account %s to pool: %w", ID, err)
		}
		if existingAccount.data.ID != nil {
			return nil, fmt.Errorf("account %s already exists: %w", ID, errors.ErrConflict)
		}
	}

	// Verify that we can assume role in the account,
	// using the `adminRoleArn`
	//_, err = TokenSvc.AssumeRole(&sts.AssumeRoleInput{
	//	RoleArn:         aws.String(request.AdminRoleArn),
	//	RoleSessionName: aws.String("MasterAssumeRoleVerification"),
	//})

	//if err != nil {
	//	return nil, fmt.Errorf("unable to add account %s to pool: adminRole is not assumable by the master account: %w", ID, errors.ErrValidation)
	//}

	// Prepare the account record
	accountStatus := model.NotReady
	account := &Account{
		data: model.Account{
			ID:           &ID,
			Status:       &accountStatus,
			AdminRoleArn: &AdminRoleArn,
			Metadata:     Metadata,
		},
	}

	// Create an IAM Role for the principal (end-user) to login to
	//masterAccountID := *CurrentAccountID
	//createRolRes, policyHash, err := createPrincipalRole(account, masterAccountID)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create principal role for %s: %w", ID, errors.ErrInternalServer)
	//}
	//account.PrincipalRoleArn = createRolRes.RoleArn
	//account.PrincipalPolicyHash = policyHash

	// Write the Account to the DB
	err = account.save(d)
	if err != nil {
		return nil, fmt.Errorf("failed to add account %s to pool with error: %s: %w", ID, err, errors.ErrInternalServer)
	}

	// Add Account to Reset Queue
	//err = Queue.SendMessage(&resetQueueURL, &account.ID)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to add account %s to reset Queue: %w", ID, errors.ErrInternalServer)
	//}

	// Publish the Account to an "account-created" topic
	//accountResponse := response.AccountResponse(account)
	//snsMessage, err := common.PrepareSNSMessageJSON(accountResponse)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create SNS account-created message for %s: %w", ID, errors.ErrInternalServer)
	//}

	// TODO: Initialize these in a better spot.

	//_, err = SnsSvc.PublishMessage(&accountCreatedTopicArn, &snsMessage, true)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to publish SNS account-created message for %s: %w", ID, errors.ErrInternalServer)
	//}

	//accountResponseJSON, err := json.Marshal(accountResponse)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to marshal account response for %s: %w", ID, errors.ErrInternalServer)
	//}
	return account, nil
}
