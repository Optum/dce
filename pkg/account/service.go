package account

import (
	"log"
	"time"

	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/imdario/mergo"
)

// Writer put an item into the data store
type Writer interface {
	Write(i *Account, lastModifiedOn *int64) error
}

// Deleter Deletes an Account from the data store
type Deleter interface {
	Delete(i *Account) error
}

// SingleReader Reads Account information from the data store
type SingleReader interface {
	Get(ID string) (*Account, error)
}

// MultipleReader reads multiple accounts from the data store
type MultipleReader interface {
	List(query *Account) (*Accounts, error)
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
	AccountCreate(account *Account) error
	AccountDelete(account *Account) error
	AccountUpdate(account *Account) error
	AccountReset(account *Account) error
}

// Manager manages all the actions against an account
type Manager interface {
	ValidateAccess(role *arn.ARN) error
	UpsertPrincipalAccess(account *Account) error
	DeletePrincipalAccess(account *Account) error
}

// Service is a type corresponding to a Account table record
type Service struct {
	dataSvc           ReaderWriterDeleter
	managerSvc        Manager
	eventSvc          Eventer
	principalRoleName string
}

// Get returns an account from ID
func (a *Service) Get(ID string) (*Account, error) {

	new, err := a.dataSvc.Get(ID)
	if err != nil {
		return nil, err
	}

	return new, err
}

// Save writes the record to the dataSvc
func (a *Service) Save(data *Account) error {
	var lastModifiedOn *int64
	now := time.Now().Unix()
	if data.LastModifiedOn == nil {
		lastModifiedOn = nil
		data.CreatedOn = &now
		data.LastModifiedOn = &now
	} else {
		lastModifiedOn = data.LastModifiedOn
		data.LastModifiedOn = &now
	}

	err := data.Validate()
	if err != nil {
		return err
	}
	err = a.dataSvc.Write(data, lastModifiedOn)
	if err != nil {
		return err
	}
	return nil
}

// Update the Account record in DynamoDB
func (a *Service) Update(ID string, data *Account) (*Account, error) {
	err := validation.ValidateStruct(data,
		// ID has to be empty
		validation.Field(&data.ID, validation.NilOrNotEmpty, validation.In(ID)),
		validation.Field(&data.AdminRoleArn, validation.By(isNilOrUsableAdminRole(a.managerSvc))),
	)
	if err != nil {
		return nil, errors.NewValidation("account", err)
	}

	account, err := a.dataSvc.Get(ID)
	if err != nil {
		return nil, err
	}

	err = mergo.Merge(account, *data, mergo.WithOverride)
	if err != nil {
		return nil, errors.NewInternalServer("unexpected error updating account", err)
	}

	err = a.Save(account)
	if err != nil {
		return nil, err
	}
	return account, nil
}

// Create creates a new account using the data provided. Returns the account record
func (a *Service) Create(data *Account) (*Account, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := validation.ValidateStruct(data,
		// This may be considered double validation but we are going to need the ID
		// so need to make sure its set
		validation.Field(&data.ID, validateID...),
		validation.Field(&data.AdminRoleArn, validateAdminRoleArn...),
		validation.Field(&data.LastModifiedOn, validation.By(isNil)),
		validation.Field(&data.Status, validation.By(isNil)),
		validation.Field(&data.CreatedOn, validation.By(isNil)),
		validation.Field(&data.PrincipalRoleArn, validation.By(isNil)),
		validation.Field(&data.PrincipalPolicyHash, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("account", err)
	}

	// Check if account already exists
	existingAccount, err := a.Get(*data.ID)
	if existingAccount != nil {
		return nil, errors.NewAlreadyExists("account", *data.ID)
	}
	if err != nil {
		if !errors.Is(err, errors.NewNotFound("account", *data.ID)) {
			return nil, err
		}
	}

	new, err := NewAccount(NewAccountInput{
		ID:                *data.ID,
		AdminRoleArn:      *data.AdminRoleArn,
		Metadata:          data.Metadata,
		PrincipalRoleName: a.principalRoleName,
	})
	if err != nil {
		return nil, err
	}

	err = a.UpsertPrincipalAccess(new)
	if err != nil {
		return nil, err
	}

	err = a.Save(new)
	if err != nil {
		return nil, err
	}

	err = a.eventSvc.AccountCreate(new)
	if err != nil {
		return nil, err
	}

	err = a.eventSvc.AccountReset(new)
	if err != nil {
		return nil, err
	}

	return new, nil
}

// Delete finds a given account and deletes it if it is not of status `Leased`. Returns the account.
func (a *Service) Delete(data *Account) error {

	err := validation.ValidateStruct(data,
		validation.Field(&data.Status, validation.NotNil, validation.By(isAccountNotLeased)),
		validation.Field(&data.AdminRoleArn, validation.NotNil),
		validation.Field(&data.PrincipalRoleArn, validation.NotNil),
	)
	if err != nil {
		return errors.NewConflict("account", *data.ID, err)
	}

	err = a.dataSvc.Delete(data)
	if err != nil {
		return err
	}

	err = a.managerSvc.DeletePrincipalAccess(data)
	if err != nil {
		return err
	}

	err = a.eventSvc.AccountDelete(data)
	if err != nil {
		return err
	}

	err = a.Reset(data)
	if err != nil {
		return err
	}

	return nil
}

// List Get a list of accounts based on a query
func (a *Service) List(query *Account) (*Accounts, error) {

	accounts, err := a.dataSvc.List(query)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

// ListPages Execute a function per page of accounts
func (a *Service) ListPages(query *Account, fn func(*Accounts) bool) error {

	for {
		records, err := a.dataSvc.List(query)
		if err != nil {
			return err
		}
		if !fn(records) {
			break
		}
		if query.NextID == nil {
			break
		}
	}

	return nil
}

// Reset initiates the Reset account process.  It will not change the status as there may
// be many reasons why a reset is called.  Delete, Lease Ending, etc.
func (a *Service) Reset(data *Account) error {
	err := validation.ValidateStruct(data,
		validation.Field(&data.Status, validation.NotNil, validation.By(isAccountNotLeased)),
		validation.Field(&data.AdminRoleArn, validation.NotNil),
		validation.Field(&data.PrincipalRoleArn, validation.NotNil),
	)
	if err != nil {
		return errors.NewConflict("account", *data.ID, err)
	}

	err = a.eventSvc.AccountReset(data)
	if err != nil {
		return err
	}
	log.Printf("Added account %q to Reset Queue\n", *data.ID)

	return nil
}

// UpsertPrincipalAccess merges principal access to make sure its in sync with expectations
func (a *Service) UpsertPrincipalAccess(data *Account) error {
	err := validation.ValidateStruct(data,
		validation.Field(&data.AdminRoleArn, validation.NotNil),
		validation.Field(&data.PrincipalRoleArn, validation.NotNil),
	)
	if err != nil {
		return errors.NewConflict("account", *data.ID, err)
	}

	oldHash := data.PrincipalPolicyHash

	err = a.managerSvc.UpsertPrincipalAccess(data)
	if err != nil {
		return err
	}
	if oldHash != data.PrincipalPolicyHash {
		err = a.Save(data)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	PrincipalRoleName string `env:"PRINCIPAL_ROLE_NAME" envDefault:"DCEPrincipal"`
	DataSvc           ReaderWriterDeleter
	ManagerSvc        Manager
	EventSvc          Eventer
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:           input.DataSvc,
		eventSvc:          input.EventSvc,
		managerSvc:        input.ManagerSvc,
		principalRoleName: input.PrincipalRoleName,
	}
}
