package lease

import (
	"time"

	"github.com/Optum/dce/pkg/account"
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

// ReaderWriter includes Reader and Writer interfaces
type ReaderWriter interface {
	Reader
	Writer
}

// Eventer for publishing events
type Eventer interface {
	LeaseCreate(account *Lease) error
	LeaseEnd(account *Lease) error
	LeaseUpdate(old *Lease, new *Lease) error
}

// AccountServicer is a partial implementation of the
// accountiface.Servicer interface, with only the methods
// needed by the LeaseService
type AccountServicer interface {
	// EndLease indicates that the provided account is no longer leased.
	Reset(id string) (*account.Account, error)
}

// Service is a type corresponding to a Lease table record
type Service struct {
	accountSvc AccountServicer
	dataSvc    ReaderWriter
	eventSvc   Eventer
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
		validation.Field(&data.AccountID, validateAccountID...),
	)
	if err != nil {
		return nil, errors.NewConflict("lease", *data.ID, err)
	}

	data.Status = StatusInactive.StatusPtr()
	err = a.dataSvc.Write(data, data.LastModifiedOn)
	if err != nil {
		return nil, err
	}

	_, err = a.accountSvc.Reset(*data.AccountID)
	if err != nil {
		return nil, err
	}

	err = a.eventSvc.LeaseEnd(data)
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

// ListPages runs a function on each page in a list
func (a *Service) ListPages(query *Lease, fn func(*Leases) bool) error {

	for {
		records, err := a.dataSvc.List(query)
		if err != nil {
			return err
		}
		if records == nil || len(*records) == 0 {
			break
		}
		if !fn(records) {
			break
		}
		if query.PrincipalID == nil {
			break
		}
	}

	return nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc    ReaderWriter
	EventSvc   Eventer
	AccountSvc AccountServicer
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:    input.DataSvc,
		eventSvc:   input.EventSvc,
		accountSvc: input.AccountSvc,
	}
}
