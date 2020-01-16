package lease

import (
	"encoding/json"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Writer put an item into the data store
type Writer interface {
	WriteLease(input *model.Lease, lastModifiedOn *int64) error
}

// Deleter Deletes an item from the data store
type Deleter interface {
	DeleteLease(input *model.Lease) error
}

// SingleReader Reads an item information from the data store
type SingleReader interface {
	GetLeaseByID(leaseID string) (*model.Lease, error)
}

// MultipleReader reads multiple items from the data store
type MultipleReader interface {
	GetLeases(*model.Lease) (*model.Leases, error)
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

// Manager manages all the actions against a lease
type Manager interface {
	Setup(adminRole string) error
}

// Account is a type corresponding to a Account table record
type Lease struct {
	writer WriterDeleter
	data   model.Lease
}

// ID Returns the Lease ID
func (a *Lease) ID() *string {
	return a.data.ID
}

// LeaseStatus Returns the Lease status
func (a *Lease) LeaseStatus() *model.LeaseStatus {
	return a.data.LeaseStatus
}

// AccountID Returns the Lease's AccountID
func (a *Lease) AccountID() *string {
	return a.data.AccountID
}

// PrincipalID Returns the Lease's PrincipalID
func (a *Lease) PrincipalID() *string {
	return a.data.PrincipalID
}

func (a *Lease) save() error {
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
	err = a.writer.WriteLease(&a.data, lastModifiedOn)
	if err != nil {
		return err
	}
	return nil
}

// Validate the lease data
func (a *Lease) Validate() error {
	err := validation.ValidateStruct(&a.data,
		validation.Field(&a.data.ID, validateID...),
		validation.Field(&a.data.LastModifiedOn, validateInt64...),
		validation.Field(&a.data.LeaseStatus, validateStatus...),
		validation.Field(&a.data.CreatedOn, validateInt64...),
		validation.Field(&a.data.PrincipalID, validatePrincipalID...),
		validation.Field(&a.data.AccountID, validateAccountID...),
	)
	if err != nil {
		return errors.NewValidation("lease", err)
	}
	return nil
}

// Update the Lease record in DynamoDB
func (a *Lease) Update(d model.Lease, am Manager) error {
	err := validation.ValidateStruct(&d,
		// ID has to be empty
		validation.Field(&d.ID, validation.NilOrNotEmpty, validation.In(*a.data.ID)),
		validation.Field(&d.ID, validation.By(isNil)),
		validation.Field(&d.LastModifiedOn, validation.By(isNil)),
		validation.Field(&d.LeaseStatus, validation.By(isNil)),
		validation.Field(&d.CreatedOn, validation.By(isNil)),
		validation.Field(&d.PrincipalID, validation.By(isNil)),
		validation.Field(&d.AccountID, validation.By(isNil)),
	)
	if err != nil {
		return errors.NewValidation("lease", err)
	}

	err = a.save()
	if err != nil {
		return err
	}
	return nil
}

// Delete finds a given lease and deletes it if it is not of status `Active`. Returns the lease.
func (a *Lease) Delete() error {

	err := validation.ValidateStruct(&a.data,
		validation.Field(&a.data.LeaseStatus, validation.NotNil, validation.By(isLeaseNotActive)),
	)
	if err != nil {
		return errors.NewConflict("lease", *a.data.ID, err)
	}

	err = a.writer.DeleteLease(&a.data)
	if err != nil {
		return err
	}

	return nil
}

// GetLeaseByID returns a lease from ID
func GetLeaseByID(ID string, d SingleReader, wd WriterDeleter) (*Lease, error) {

	newLease := Lease{
		writer: wd,
	}
	data, err := d.GetLeaseByID(ID)
	if err != nil {
		return nil, err
	}
	newLease.data = *data

	return &newLease, err
}

// New returns a lease from ID
func New(wd WriterDeleter, data model.Lease) *Lease {
	now := time.Now().Unix()
	lease := &Lease{
		writer: wd,
		data:   data,
	}
	lease.data.CreatedOn = &now
	lease.data.LastModifiedOn = &now
	return lease
}

// MarshalJSON Marshals the data inside the lease
func (a *Lease) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.data)
}
