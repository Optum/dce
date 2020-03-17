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

// Service is a type corresponding to a Lease table record
type Service struct {
	dataSvc                  ReaderWriter
	eventSvc                 Eventer
	defaultLeaseLengthInDays int
	principalBudgetAmount    float64
	principalBudgetPeriod    string
	maxLeaseBudgetAmount     float64
	maxLeasePeriod           int64
}

// Weekly
const (
	Weekly = "WEEKLY"
)

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
		data.StatusModifiedOn = &now
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

// Create creates a new lease using the data provided. Returns the lease record
func (a *Service) Create(data *Lease, principalSpentAmount float64) (*Lease, error) {

	// Set default expiresOn
	if data.ExpiresOn == nil {
		leaseExpires := time.Now().AddDate(0, 0, a.defaultLeaseLengthInDays).Unix()
		data.ExpiresOn = &leaseExpires
	}

	// Set default metadata (empty object)
	if data.Metadata == nil {
		data.Metadata = map[string]interface{}{}
	}

	// Set default budget amount
	if data.BudgetAmount == nil {
		data.BudgetAmount = &a.maxLeaseBudgetAmount
	}

	// Set default budget currency
	if data.BudgetCurrency == nil {
		currency := ""
		data.BudgetCurrency = &currency
	}

	// Set default budget notification emails
	if data.BudgetNotificationEmails == nil {
		notificationEmails := []string{""}
		data.BudgetNotificationEmails = &notificationEmails
	}

	// Validate the incoming record doesn't have unneeded fields
	err := validation.ValidateStruct(data,
		validation.Field(&data.AccountID, validateAccountID...),
		validation.Field(&data.PrincipalID, validatePrincipalID...),
		validation.Field(&data.ID, validation.By(isNil)),
		validation.Field(&data.Status, validation.By(isNil)),
		validation.Field(&data.StatusReason, validation.By(isNil)),
		validation.Field(&data.ExpiresOn, validation.NotNil, validation.By(isExpiresOnValid(a))),
	)
	if err != nil {
		return nil, errors.NewValidation("lease", err)
	}

	err = validation.ValidateStruct(data,
		validation.Field(&data.BudgetAmount, validation.By(isBudgetAmountValid(a, *data.PrincipalID, principalSpentAmount))),
	)
	if err != nil {
		return nil, errors.NewValidation("lease", err)
	}

	// Check if principal already has an active lease
	query := &Lease{
		PrincipalID: data.PrincipalID,
		Status:      StatusActive.StatusPtr(),
	}

	existingLeases, err := a.List(query)
	if err != nil {
		return nil, errors.NewInternalServer("lease", err)
	}
	if existingLeases != nil && len(*existingLeases) > 0 {
		return nil, errors.NewAlreadyExists("lease", *data.PrincipalID)
	}

	newLeaseRecord := NewLease(NewLeaseInput{
		AccountID:                *data.AccountID,
		PrincipalID:              *data.PrincipalID,
		BudgetAmount:             *data.BudgetAmount,
		BudgetCurrency:           *data.BudgetCurrency,
		BudgetNotificationEmails: *data.BudgetNotificationEmails,
		ExpiresOn:                *data.ExpiresOn,
		LastModifiedOn:           *data.LastModifiedOn,
		CreatedOn:                *data.CreatedOn,
	})
	if err != nil {
		return nil, err
	}

	err = a.Save(newLeaseRecord)
	if err != nil {
		return nil, err
	}

	err = a.eventSvc.LeaseCreate(newLeaseRecord)
	if err != nil {
		return nil, err
	}

	return newLeaseRecord, nil
}

// ListPages runs a function on each page in a list
func (a *Service) ListPages(query *Lease, fn func(*Leases) bool) error {

	for {
		records, err := a.dataSvc.List(query)
		if err != nil {
			return err
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
	DataSvc                  ReaderWriter
	EventSvc                 Eventer
	DefaultLeaseLengthInDays int     `env:"DEFAULT_LEASE_LENGTH_IN_DAYS" envDefault:"7"`
	PrincipalBudgetAmount    float64 `env:"PRINCIPAL_BUDGET_AMOUNT" envDefault:"1000.00"`
	PrincipalBudgetPeriod    string  `env:"PRINCIPAL_BUDGET_PERIOD" envDefault:"Weekly"`
	MaxLeaseBudgetAmount     float64 `env:"MAX_LEASE_BUDGET_AMOUNT" envDefault:"1000.00"`
	MaxLeasePeriod           int64   `env:"MAX_LEASE_PERIOD" envDefault:"704800"`
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:                  input.DataSvc,
		eventSvc:                 input.EventSvc,
		defaultLeaseLengthInDays: input.DefaultLeaseLengthInDays,
		principalBudgetAmount:    input.PrincipalBudgetAmount,
		principalBudgetPeriod:    input.PrincipalBudgetPeriod,
		maxLeaseBudgetAmount:     input.MaxLeaseBudgetAmount,
		maxLeasePeriod:           input.MaxLeasePeriod,
	}
}
