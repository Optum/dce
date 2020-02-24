package main

import (
	"context"
	"testing"
	"time"

	"github.com/Optum/dce/internal/types"
	"github.com/Optum/dce/pkg/account"
	accountMocks "github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/usage"
	usageMocks "github.com/Optum/dce/pkg/usage/usageiface/mocks"
	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptr64(i int64) *int64 {
	ptrI := i
	return &ptrI
}

func ptrFloat64(f float64) *float64 {
	ptrF := f
	return &ptrF
}
func TestGetLeaseUsage(t *testing.T) {

	type getUsage struct {
		lease     *lease.Lease
		startDate time.Time
		endDate   time.Time
		retData   types.Usages
		retErr    error
	}

	now := time.Now()
	beginningOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		input         lease.Lease
		getAccount    *account.Account
		getAccountErr error
		getUsage      getUsage
		putUsages     usage.Leases
		puttErr       error
		expErr        error
		expOut        lease.Lease
	}{
		{
			name: "when valid lease provided get current lease with TTL",
			input: lease.Lease{
				PrincipalID:    ptrString("user"),
				ID:             ptrString("id-1"),
				AccountID:      ptrString("123456789012"),
				CreatedOn:      ptr64(now.AddDate(0, 0, -3).Unix()),
				Status:         lease.StatusActive.StatusPtr(),
				BudgetCurrency: ptrString("USD"),
			},
			getAccount: &account.Account{
				ID: ptrString("123456789012"),
			},
			getUsage: getUsage{
				startDate: beginningOfDay.AddDate(0, 0, -2),
				endDate:   beginningOfDay,
				retData: types.Usages{
					{
						TimePeriod: beginningOfDay,
						Amount:     10.0,
					},
				},
				retErr: nil,
			},
			putUsages: usage.Leases{
				{
					PrincipalID:  ptrString("user"),
					Date:         &beginningOfDay,
					LeaseID:      ptrString("id-1"),
					CostAmount:   ptrFloat64(10.0),
					CostCurrency: ptrString("USD"),
				},
			},
			expOut: lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("user"),
				ID:             ptrString("id-1"),
				Status:         lease.StatusActive.StatusPtr(),
				CreatedOn:      ptr64(now.AddDate(0, 0, -3).Unix()),
				BudgetCurrency: ptrString("USD"),
			},
		},
		{
			name: "when inactive lease within 30 hours",
			input: lease.Lease{
				PrincipalID:      ptrString("user"),
				ID:               ptrString("id-1"),
				AccountID:        ptrString("123456789012"),
				CreatedOn:        ptr64(now.AddDate(0, 0, -3).Unix()),
				Status:           lease.StatusInactive.StatusPtr(),
				StatusModifiedOn: ptr64(now.Unix()),
				BudgetCurrency:   ptrString("USD"),
			},
			getAccount: &account.Account{
				ID: ptrString("123456789012"),
			},
			getUsage: getUsage{
				startDate: beginningOfDay.AddDate(0, 0, -2),
				endDate:   beginningOfDay,
				retData: types.Usages{
					{
						TimePeriod: beginningOfDay,
						Amount:     10.0,
					},
				},
				retErr: nil,
			},
			putUsages: usage.Leases{
				{
					PrincipalID:  ptrString("user"),
					Date:         &beginningOfDay,
					LeaseID:      ptrString("id-1"),
					CostAmount:   ptrFloat64(10.0),
					CostCurrency: ptrString("USD"),
				},
			},
			expOut: lease.Lease{
				AccountID:        ptrString("123456789012"),
				PrincipalID:      ptrString("user"),
				ID:               ptrString("id-1"),
				Status:           lease.StatusInactive.StatusPtr(),
				StatusModifiedOn: ptr64(now.Unix()),
				CreatedOn:        ptr64(now.AddDate(0, 0, -3).Unix()),
				BudgetCurrency:   ptrString("USD"),
			},
		},
		{
			name: "when created today don't query previous days",
			input: lease.Lease{
				PrincipalID:      ptrString("user"),
				ID:               ptrString("id-1"),
				AccountID:        ptrString("123456789012"),
				CreatedOn:        ptr64(now.Unix()),
				Status:           lease.StatusActive.StatusPtr(),
				StatusModifiedOn: ptr64(now.Unix()),
				BudgetCurrency:   ptrString("USD"),
			},
			getAccount: &account.Account{
				ID: ptrString("123456789012"),
			},
			getUsage: getUsage{
				startDate: beginningOfDay,
				endDate:   beginningOfDay,
				retData: types.Usages{
					{
						TimePeriod: beginningOfDay,
						Amount:     10.0,
					},
				},
				retErr: nil,
			},
			putUsages: usage.Leases{
				{
					PrincipalID:  ptrString("user"),
					Date:         &beginningOfDay,
					LeaseID:      ptrString("id-1"),
					CostAmount:   ptrFloat64(10.0),
					CostCurrency: ptrString("USD"),
				},
			},
			expOut: lease.Lease{
				AccountID:        ptrString("123456789012"),
				PrincipalID:      ptrString("user"),
				ID:               ptrString("id-1"),
				Status:           lease.StatusActive.StatusPtr(),
				StatusModifiedOn: ptr64(now.Unix()),
				CreatedOn:        ptr64(now.Unix()),
				BudgetCurrency:   ptrString("USD"),
			},
		},
	}

	// Iterate through each test in the list
	for _, tt := range tests {
		cfgBldr := &config.ConfigurationBuilder{}
		svcBldr := &config.ServiceBuilder{Config: cfgBldr}
		// Setup mocks

		accountSvcMock := accountMocks.Servicer{}
		usageSvcMocks := usageMocks.Servicer{}
		accountSvcMock.On("Get", *tt.input.AccountID).Return(tt.getAccount, tt.getAccountErr)
		accountSvcMock.On("GetUsageBetweenDates", tt.getAccount, tt.getUsage.startDate, tt.getUsage.endDate).Return(tt.getUsage.retData, tt.getUsage.retErr)

		for _, usg := range tt.putUsages {
			usageSvcMocks.On("UpsertLeaseUsage", &usg).Return(nil)
		}

		svcBldr.Config.WithService(&accountSvcMock).WithService(&usageSvcMocks)
		_, err := svcBldr.Build()
		assert.Nil(t, err)
		if err == nil {
			services = svcBldr
		}

		out, err := handler(context.TODO(), tt.input)
		assert.True(t, errors.Is(err, tt.expErr))
		assert.Equal(t, tt.expOut, out)
	}
}
