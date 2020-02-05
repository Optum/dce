package lease

import (
	"time"

	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Writer put an item into the data store
type Writer interface {
	Write(input *Lease, lastModifiedOn *int64) error
}

// Deleter Deletes an item from the data store
type Deleter interface {
	Delete(input *Lease) error
}

// SingleReader Reads an item information from the data store
type SingleReader interface {
	Get(leaseID string) (*Lease, error)
}

// MultipleReader reads multiple items from the data store
type MultipleReader interface {
	List(*Lease) (*Leases, error)
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

// Service is a type corresponding to a Lease table record
type Service struct {
	dataSvc  ReaderWriterDeleter
	eventSvc Eventer
}

// Get returns a lease from ID
func (a *Service) Get(ID string) (*Lease, error) {

	new, err := a.dataSvc.Get(ID)
	if err != nil {
		return nil, err
	}

	return new, err
}

// Save writes the record to the dataSvc
func (a *Service) Save(data *Lease) error {
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

// Delete finds a given lease and deletes it if it is not of status `Active`. Returns the lease.
func (a *Service) Delete(data *Lease) error {

	err := a.dataSvc.Delete(data)
	if err != nil {
		return err
	}

	return nil
}

// List Get a list of leases based on Principal ID
func (a *Service) List(query *Lease) (*Leases, error) {
	err := validation.ValidateStruct(query,
		// ID has to be empty
		validation.Field(&query.ID, validation.NilOrNotEmpty, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("lease", err)
	}

	leases, err := a.dataSvc.List(query)
	if err != nil {
		return nil, err
	}

	return leases, nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc  ReaderWriterDeleter
	EventSvc Eventer
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:  input.DataSvc,
		eventSvc: input.EventSvc,
	}
}
