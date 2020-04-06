package usage_test

import (
	errors2 "errors"
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/pkg/usage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrFloat64(f float64) *float64 {
	ptrF := f
	return &ptrF
}

func TestUpsertLeaseUsage(t *testing.T) {

	now := time.Now()

	tests := []struct {
		name string
		req  usage.Lease
		ret  error
		exp  error
	}{
		{
			name: "should upsert usage information",
			req: usage.Lease{
				Date:         &now,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(1.0),
				CostCurrency: ptrString("USD"),
			},
			ret: nil,
			exp: nil,
		},
		{
			name: "should get failure",
			req: usage.Lease{
				Date:         &now,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(1.0),
				CostCurrency: ptrString("USD"),
			},
			ret: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			exp: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
		},
		{
			name: "should fail validation",
			req: usage.Lease{
				Date:        &now,
				PrincipalID: ptrString("test"),
				LeaseID:     ptrString("id-1"),
				CostAmount:  ptrFloat64(1.0),
			},
			exp: errors.NewValidation("usage", fmt.Errorf("costCurrency: must be a valid cost currency.")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.LeaseReaderWriter{}

			mocksRwd.On("Write", mock.AnythingOfType("*usage.Lease")).Return(tt.ret)

			usageSvc := usage.NewService(usage.NewServiceInput{
				DataLeaseSvc: mocksRwd,
			})

			err := usageSvc.UpsertLeaseUsage(&tt.req)
			assert.True(t, errors.Is(err, tt.exp), "actual error %q doesn't match expected error %q", err, tt.exp)
		})
	}
}

func TestGetLease(t *testing.T) {

	tests := []struct {
		name             string
		leaseID          string
		mockLeaseUsage   *usage.Lease
		dataServiceError error
		expectedReturn   *usage.Lease
		expectedError    error
	}{
		{
			name:           "should return a lease from the data service",
			leaseID:        "mock-lease-id",
			mockLeaseUsage: &usage.Lease{LeaseID: ptrString("mock-lease-id")},
			expectedReturn: &usage.Lease{LeaseID: ptrString("mock-lease-id")},
		},
		{
			name:             "should return an error from the data service",
			leaseID:          "mock-lease-id",
			dataServiceError: errors2.New("data service error"),
			expectedReturn:   nil,
			expectedError:    errors2.New("data service error"),
		},
	}

	for _, tt := range tests {
		mockDataSvc := &mocks.LeaseReaderWriter{}

		// Mock the lease record from the DB
		mockDataSvc.
			On("Get", tt.leaseID).
			Return(tt.mockLeaseUsage, tt.dataServiceError)

		usageSvc := usage.NewService(usage.NewServiceInput{
			DataLeaseSvc: mockDataSvc,
		})

		res, err := usageSvc.GetLease(tt.leaseID)
		assert.Equal(t, tt.expectedReturn, res)
		assert.Equal(t, tt.expectedError, err)
	}
}

func TestGetPrincipal(t *testing.T) {

	tests := []struct {
		name                     string
		principalID              string
		principalBudgetStartDate time.Time
		mockPrincipalUsage       *usage.Principal
		dataServiceError         error
		expectedReturn           *usage.Principal
		expectedError            error
	}{
		{
			name:                     "should return a principal usage record from the data service",
			principalID:              "mock-principal-id",
			principalBudgetStartDate: time.Unix(100, 0),
			mockPrincipalUsage:       &usage.Principal{PrincipalID: ptrString("mock-principal-id")},
			expectedReturn:           &usage.Principal{PrincipalID: ptrString("mock-principal-id")},
		},
		{
			name:                     "should return an error from the data service",
			principalID:              "mock-principal-id",
			principalBudgetStartDate: time.Unix(100, 0),
			dataServiceError:         errors2.New("data service error"),
			expectedReturn:           nil,
			expectedError:            errors2.New("data service error"),
		},
	}

	for _, tt := range tests {
		mockDataSvc := &mocks.PrincipalReader{}

		// Mock the lease record from the DB
		mockDataSvc.
			On("Get", tt.principalID, tt.principalBudgetStartDate).
			Return(tt.mockPrincipalUsage, tt.dataServiceError)

		usageSvc := usage.NewService(usage.NewServiceInput{
			DataPrincipalSvc: mockDataSvc,
		})

		res, err := usageSvc.GetPrincipal(tt.principalID, tt.principalBudgetStartDate)
		assert.Equal(t, tt.expectedReturn, res)
		assert.Equal(t, tt.expectedError, err)
	}
}

func TestListPrincipal(t *testing.T) {

	tests := []struct {
		name                string
		mockPrincipalUsages usage.Principals
		dataServiceError    error
		expectedReturn      usage.Principals
		expectedError       error
	}{
		{
			name: "should return principal usage records from the data service",
			mockPrincipalUsages: usage.Principals(
				[]usage.Principal{
					{PrincipalID: ptrString("mock-principal-id-1")},
					{PrincipalID: ptrString("mock-principal-id-2")},
				},
			),
			expectedReturn: usage.Principals(
				[]usage.Principal{
					{PrincipalID: ptrString("mock-principal-id-1")},
					{PrincipalID: ptrString("mock-principal-id-2")},
				},
			),
		},
		{
			name:             "should return an error from the data service",
			dataServiceError: errors2.New("data service error"),
			expectedReturn:   nil,
			expectedError:    errors2.New("data service error"),
		},
	}

	for _, tt := range tests {
		mockDataSvc := &mocks.PrincipalReader{}

		query := &usage.Principal{PrincipalID: ptrString("mock-id")}

		// Mock the lease record from the DB
		mockDataSvc.
			On("List", query).
			Return(&tt.mockPrincipalUsages, tt.dataServiceError)

		usageSvc := usage.NewService(usage.NewServiceInput{
			DataPrincipalSvc: mockDataSvc,
		})

		res, err := usageSvc.ListPrincipal(query)
		if tt.expectedError != nil {
			assert.Equal(t, tt.expectedError, err)
		} else {
			assert.Equal(t, &tt.expectedReturn, res)
		}
	}
}

func TestLeases(t *testing.T) {

	tests := []struct {
		name             string
		mockLeaseUsages  usage.Leases
		dataServiceError error
		expectedReturn   usage.Leases
		expectedError    error
	}{
		{
			name: "should return lease usage records from the data service",
			mockLeaseUsages: usage.Leases([]usage.Lease{
				{LeaseID: ptrString("lease-1")},
				{LeaseID: ptrString("lease-2")},
			}),
			expectedReturn: usage.Leases([]usage.Lease{
				{LeaseID: ptrString("lease-1")},
				{LeaseID: ptrString("lease-2")},
			}),
		},
		{
			name:             "should return an error from the data service",
			dataServiceError: errors2.New("data service error"),
			expectedReturn:   nil,
			expectedError:    errors2.New("data service error"),
		},
	}

	for _, tt := range tests {
		mockDataSvc := &mocks.LeaseReaderWriter{}

		query := &usage.Lease{LeaseID: ptrString("mock-id")}

		// Mock the lease record from the DB
		mockDataSvc.
			On("List", query).
			Return(&tt.mockLeaseUsages, tt.dataServiceError)

		usageSvc := usage.NewService(usage.NewServiceInput{
			DataLeaseSvc: mockDataSvc,
		})

		res, err := usageSvc.ListLease(query)
		if tt.expectedError != nil {
			assert.Equal(t, tt.expectedError, err)
		} else {
			assert.Equal(t, &tt.expectedReturn, res)
		}
	}
}
