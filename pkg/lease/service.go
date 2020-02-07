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

// ReaderWriterDeleter includes Reader and Writer interfaces
type ReaderWriter interface {
	Reader
	Writer
}

// Eventer for publishing events
type Eventer interface {
	Publish() error
}

// Service is a type corresponding to a Lease table record
type Service struct {
	dataSvc  ReaderWriter
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

// Delete finds a given lease and checks if it's active and then updates it to status `Inactive`. Returns the lease.
func (a *Service) Delete(ID string) (*Lease, error) {

	data, err := a.dataSvc.Get(ID)
	if err != nil {
		return nil, err
	}

	err = validation.ValidateStruct(data,
		validation.Field(&data.Status, validation.NotNil, validation.By(isLeaseActive)),
	)
	if err != nil {
		return nil, errors.NewConflict("lease", *data.ID, err)
	}

	data.Status = StatusInactive.StatusPtr()
	err = a.dataSvc.Write(data, data.LastModifiedOn)
	if err != nil {
		return nil, err
	}

	return data, nil
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

// Create creates a new lease using the data provided. Returns the lease record
func (a *Service) Create(data *Lease) (*Lease, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := validation.ValidateStruct(data,
		validation.Field(&data.AccountID, validateAccountID...),
		validation.Field(&data.PrincipalID, validatePrincipalID...),
		validation.Field(&data.ID, validation.By(isNil)),
		validation.Field(&data.Status, validation.By(isNil)),
		validation.Field(&data.LastModifiedOn, validation.By(isNil)),
		validation.Field(&data.CreatedOn, validation.By(isNil)),
		validation.Field(&data.StatusReason, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("lease", err)
	}

	// Check if principal already has an active lease
	existingLeases, err := a.List(data)
	if err != nil {
		return nil, errors.NewInternalServer("lease", err)
	}
	if len(*existingLeases) > 0 {
		return nil, errors.NewAlreadyExists("lease", *data.ID)
	}

	data.Status = StatusActive.StatusPtr()
	err = a.Save(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc  ReaderWriter
	EventSvc Eventer
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:  input.DataSvc,
		eventSvc: input.EventSvc,
	}
}
