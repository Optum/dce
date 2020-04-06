package usage

import "time"

// PrincipalReader reads principal usage information form the data store
type PrincipalReader interface {
	Get(principalID string, principalBudgetStartDate time.Time) (*Principal, error)
	List(query *Principal) (*Principals, error)
}

// LeaseWriter put an item into the data store
type LeaseWriter interface {
	Write(i *Lease) error
}

// LeaseReader data Layer
type LeaseReader interface {
	Get(id string) (*Lease, error)
	List(query *Lease) (*Leases, error)
}

// LeaseReaderWriter includes Reader and Writer interfaces
type LeaseReaderWriter interface {
	LeaseReader
	LeaseWriter
}

// Service is a type corresponding to a Usage table record
type Service struct {
	dataLeaseSvc     LeaseReaderWriter
	dataPrincipalSvc PrincipalReader
	budgetPeriod     string
}

// UpsertLeaseUsage creates a new lease usage record
func (a *Service) UpsertLeaseUsage(data *Lease) error {
	// Validate the incoming record doesn't have unneeded fields
	err := data.Validate()
	if err != nil {
		return err
	}

	err = a.dataLeaseSvc.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// GetLease gets a single Lease Usage record
func (a *Service) GetLease(id string) (*Lease, error) {

	usg, err := a.dataLeaseSvc.Get(id)
	if err != nil {
		return nil, err
	}

	return usg, nil
}

// GetPrincipal gets a single Principal Usage record
func (a *Service) GetPrincipal(principalID string, principalBudgetStartDate time.Time) (*Principal, error) {
	usg, err := a.dataPrincipalSvc.Get(principalID, principalBudgetStartDate)
	if err != nil {
		return nil, err
	}

	return usg, nil
}

// ListPrincipal list Principal Usage records
func (a *Service) ListPrincipal(data *Principal) (*Principals, error) {
	usgs, err := a.dataPrincipalSvc.List(data)
	if err != nil {
		return nil, err
	}

	return usgs, nil
}

// ListLease list Lease Usage records
func (a *Service) ListLease(data *Lease) (*Leases, error) {

	usgs, err := a.dataLeaseSvc.List(data)
	if err != nil {
		return nil, err
	}

	return usgs, nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataLeaseSvc     LeaseReaderWriter
	DataPrincipalSvc PrincipalReader
	BudgetPeriod     string
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataLeaseSvc:     input.DataLeaseSvc,
		dataPrincipalSvc: input.DataPrincipalSvc,
		budgetPeriod:     input.BudgetPeriod,
	}
}
