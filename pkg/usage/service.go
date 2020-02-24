package usage

import (
	"fmt"
	"strconv"

	"github.com/Optum/dce/pkg/errors"
)

// Writer put an item into the data store
type Writer interface {
	Write(i *Usage) error
}

// SingleReader Reads Usage information from the data store
type SingleReader interface {
	Get(startDate int64, principalID string) (*Usage, error)
}

// MultipleReader reads multiple usages from the data store
type MultipleReader interface {
	List(query *Usage) (*Usages, error)
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

// Service is a type corresponding to a Usage table record
type Service struct {
	dataSvc ReaderWriter
}

// Get returns an usage from startDate and principalID
func (a *Service) Get(startDate int64, principalID string) (*Usage, error) {

	new, err := a.dataSvc.Get(startDate, principalID)
	if err != nil {
		return nil, err
	}

	return new, err
}

// save writes the record to the dataSvc
func (a *Service) save(data *Usage) error {

	err := data.Validate()
	if err != nil {
		return err
	}
	err = a.dataSvc.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// Create creates a new usage record
func (a *Service) Create(data *Usage) (*Usage, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := data.Validate()
	if err != nil {
		return nil, errors.NewValidation("usage", err)
	}

	// Check if usage already exists
	existing, err := a.Get(*data.StartDate, *data.PrincipalID)
	if existing != nil {
		return nil, errors.NewAlreadyExists("usage", fmt.Sprintf("%s-%s", strconv.FormatInt(*data.StartDate, 10), *data.PrincipalID))
	}
	if err != nil {
		if !errors.Is(err, errors.NewNotFound("usage", fmt.Sprintf("%s-%s", strconv.FormatInt(*data.StartDate, 10), *data.PrincipalID))) {
			return nil, err
		}
	}

	new, err := NewUsage(NewUsageInput{
		StartDate:    *data.StartDate,
		PrincipalID:  *data.PrincipalID,
		AccountID:    *data.AccountID,
		EndDate:      *data.EndDate,
		CostAmount:   *data.CostAmount,
		CostCurrency: *data.CostCurrency,
		TimeToLive:   *data.TimeToLive,
	})
	if err != nil {
		return nil, err
	}

	err = a.save(new)
	if err != nil {
		return nil, err
	}

	return new, nil
}

// List Get a list of usages based on a query
func (a *Service) List(query *Usage) (*Usages, error) {

	usages, err := a.dataSvc.List(query)
	if err != nil {
		return nil, err
	}

	return usages, nil
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc ReaderWriter
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc: input.DataSvc,
	}
}
