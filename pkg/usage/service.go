package usage

import (
	"fmt"
	"strings"
	"time"

	"github.com/Optum/dce/pkg/account/accountiface"
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

const (
	usagePrefix = "Usage"
)

// Writer put an item into the data store
type Writer interface {
	Write(i *Usage) (*Usage, error)
	Add(i *Usage) (*Usage, error)
}

// SingleReader Reads Usage information from the data store
type SingleReader interface{}

// MultipleReader reads multiple usages from the data store
type MultipleReader interface{}

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
	dataSvc      ReaderWriter
	accountSvc   accountiface.Servicer
	budgetPeriod string
	usageTTL     int
}

// UpsertLeaseUsage creates a new lease usage record
func (a *Service) UpsertLeaseUsage(data *Usage) (*Usage, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := validation.ValidateStruct(data,
		validation.Field(&data.SK, validation.By(isNil)),
		validation.Field(&data.Date, validation.NotNil),
		validation.Field(&data.TimeToLive, validation.By(isNil)),
	)
	if err != nil {
		return nil, errors.NewValidation("usage", err)
	}

	sortKey := fmt.Sprintf("%s-Lease-%s-%d", usagePrefix, *data.LeaseID, data.Date.Unix())
	data.SK = &sortKey
	ttl := data.Date.AddDate(0, 0, a.usageTTL).Unix()
	data.TimeToLive = &ttl

	old, err := a.dataSvc.Write(data)
	if err != nil {
		return data, err
	}

	diffUsg := Usage{
		PrincipalID:  data.PrincipalID,
		Date:         data.Date,
		CostAmount:   data.CostAmount,
		CostCurrency: data.CostCurrency,
		LeaseID:      data.LeaseID,
		TimeToLive:   data.TimeToLive,
	}
	if old.CostAmount != nil {
		diffCost := *diffUsg.CostAmount - *old.CostAmount
		diffUsg.CostAmount = &diffCost
	}

	_, err = a.addLeaseUsage(diffUsg)
	if err != nil {
		return data, err
	}

	_, err = a.addPeriodUsage(diffUsg)
	if err != nil {
		return data, err
	}

	return data, nil
}

// addLeaseUsage addes to the current usage record for the period
func (a *Service) addLeaseUsage(data Usage) (*Usage, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := validation.ValidateStruct(&data,
		validation.Field(&data.SK, validation.By(isNil)),
		validation.Field(&data.Date, validation.NotNil),
		validation.Field(&data.LeaseID, validation.NotNil),
	)
	if err != nil {
		return nil, errors.NewValidation("usage", err)
	}

	ttl := data.Date.AddDate(0, 0, a.usageTTL).Unix()
	data.TimeToLive = &ttl

	sortKey := fmt.Sprintf("%s-Lease-%s-Total", usagePrefix, *data.LeaseID)
	data.SK = &sortKey

	_, err = a.dataSvc.Add(&data)
	if err != nil {
		return &data, err
	}

	return &data, nil
}

// addPeriodUsage addes to the current usage record for the period
func (a *Service) addPeriodUsage(data Usage) (*Usage, error) {
	// Validate the incoming record doesn't have unneeded fields
	err := validation.ValidateStruct(&data,
		validation.Field(&data.SK, validation.By(isNil)),
		validation.Field(&data.Date, validation.NotNil),
	)
	if err != nil {
		return nil, errors.NewValidation("usage", err)
	}

	data.Date = a.getBudgetPeriodTime(data.Date)
	ttl := data.Date.AddDate(0, 0, a.usageTTL).Unix()
	data.TimeToLive = &ttl

	sortKey := fmt.Sprintf(
		"%s-Principal-%s-%d",
		usagePrefix,
		strings.Title(strings.ToLower(a.budgetPeriod)),
		data.Date.Unix())
	data.SK = &sortKey

	_, err = a.dataSvc.Add(&data)
	if err != nil {
		return &data, err
	}

	return &data, nil
}

// budgetPeriod gets the epoch for the start of a period
func (a *Service) getBudgetPeriodTime(date *time.Time) *time.Time {
	if date == nil {
		return date
	}

	var new time.Time
	if a.budgetPeriod == "MONTHLY" {
		new = time.Date(date.Year(), date.Month(), 0, 0, 0, 0, 0, time.UTC)
	} else {
		new = firstDayOfISOWeek(date.ISOWeek())
	}

	return &new
}

func firstDayOfISOWeek(year int, week int) time.Time {
	date := time.Date(year, 0, 0, 0, 0, 0, 0, time.UTC)
	isoYear, isoWeek := date.ISOWeek()

	// iterate back to Monday
	for date.Weekday() != time.Monday {
		date = date.AddDate(0, 0, -1)
		isoYear, isoWeek = date.ISOWeek()
	}

	// iterate forward to the first day of the first week
	for isoYear < year {
		date = date.AddDate(0, 0, 7)
		isoYear, isoWeek = date.ISOWeek()
	}

	// iterate forward to the first day of the given week
	for isoWeek < week {
		date = date.AddDate(0, 0, 7)
		isoYear, isoWeek = date.ISOWeek()
	}

	return date
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc      ReaderWriter
	BudgetPeriod string `env:"PRINCIPAL_BUDGET_PERIOD" envDefault:"WEEKLY"`
	UsageTTL     int    `env:"USAGE_TTL" envDefault:"30"`
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc:      input.DataSvc,
		budgetPeriod: input.BudgetPeriod,
		usageTTL:     input.UsageTTL,
	}
}
