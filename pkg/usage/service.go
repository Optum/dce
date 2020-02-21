package usage

import (
	"github.com/Optum/dce/pkg/account/accountiface"
)

const (
	usagePrefix = "Usage"
)

// LeaseWriter put an item into the data store
type LeaseWriter interface {
	Write(i *Lease) (*Lease, error)
}

// SingleReader Reads Usage information from the data store
type SingleReader interface{}

// MultipleReader reads multiple usages from the data store
type MultipleReader interface{}

// LeaseReader data Layer
type LeaseReader interface {
	SingleReader
	MultipleReader
}

// LeaseReaderWriter includes Reader and Writer interfaces
type LeaseReaderWriter interface {
	LeaseReader
	LeaseWriter
}

// Service is a type corresponding to a Usage table record
type Service struct {
	dataLeaseSvc LeaseReaderWriter
	accountSvc   accountiface.Servicer
	budgetPeriod string
}

// UpsertLeaseUsage creates a new lease usage record
func (a *Service) UpsertLeaseUsage(data *Lease) (*Lease, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := data.Validate()
	if err != nil {
		return nil, err
	}

	old, err := a.dataLeaseSvc.Write(data)
	if err != nil {
		return nil, err
	}

	return old, nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc      LeaseReaderWriter
	BudgetPeriod string
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataLeaseSvc: input.DataSvc,
		budgetPeriod: input.BudgetPeriod,
	}
}
