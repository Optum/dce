package account

import (
	"time"

	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
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
	Publish() error
}

// Manager manages all the actions against an account
type Manager interface {
	Setup(adminRole string) error
}

// Service is a type corresponding to a Account table record
type Service struct {
	dataSvc    ReaderWriterDeleter
	managerSvc Manager
	eventSvc   Eventer
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
		validation.Field(&data.ID, validation.By(isNil)),
		validation.Field(&data.LastModifiedOn, validation.By(isNil)),
		validation.Field(&data.Status, validation.By(isNil)),
		validation.Field(&data.CreatedOn, validation.By(isNil)),
		validation.Field(&data.PrincipalRoleArn, validation.By(isNil)),
		validation.Field(&data.PrincipalPolicyHash, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("account", err)
	}

	account, err := a.dataSvc.Get(ID)
	if err != nil {
		return nil, err
	}

	if data.AdminRoleArn != nil {
		account.AdminRoleArn = data.AdminRoleArn
	}
	if data.Metadata != nil {
		account.Metadata = data.Metadata
	}

	err = a.Save(account)
	if err != nil {
		return nil, err
	}
	return account, nil
}

// Delete finds a given account and deletes it if it is not of status `Leased`. Returns the account.
func (a *Service) Delete(data *Account) error {

	err := validation.ValidateStruct(data,
		validation.Field(&data.Status, validation.NotNil, validation.By(isAccountNotLeased)),
	)
	if err != nil {
		return errors.NewConflict("account", *data.ID, err)
	}

	err = a.dataSvc.Delete(data)
	if err != nil {
		return err
	}

	return nil
}

// List Get a list of accounts based on Principal ID
func (a *Service) List(query *Account) (*Accounts, error) {

	accounts, err := a.dataSvc.List(query)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc    ReaderWriterDeleter
	ManagerSvc Manager
	EventSvc   Eventer
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:    input.DataSvc,
		eventSvc:   input.EventSvc,
		managerSvc: input.ManagerSvc,
	}
}
