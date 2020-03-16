package lease_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrFloat(s float64) *float64 {
	ptrS := s
	return &ptrS
}
func ptrArrayString(s []string) *[]string {
	ptrS := s
	return &ptrS
}

func TestGetLeaseByID(t *testing.T) {

	type response struct {
		data *lease.Lease
		err  error
	}

	tests := []struct {
		name string
		ID   string
		ret  response
		exp  response
	}{
		{
			name: "should get an lease by ID",
			ID:   "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			ret: response{
				data: &lease.Lease{
					ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status: lease.StatusActive.StatusPtr(),
				},
				err: nil,
			},
			exp: response{
				data: &lease.Lease{
					ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status: lease.StatusActive.StatusPtr(),
				},
				err: nil,
			},
		},
		{
			name: "should get failure",
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriter{}

			mocksRwd.On("Get", tt.ID).Return(tt.ret.data, tt.ret.err)

			leaseSvc := lease.NewService(lease.NewServiceInput{
				DataSvc: mocksRwd,
			})

			getLease, err := leaseSvc.Get(tt.ID)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, getLease)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		ID        string
		expErr    error
		returnErr error
		expLease  *lease.Lease
	}{
		{
			name: "should delete a lease",
			ID:   "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			expLease: &lease.Lease{
				ID:           ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:       lease.StatusActive.StatusPtr(),
				StatusReason: lease.StatusReasonDestroyed.StatusReasonPtr(),
			},
			returnErr: nil,
		},
		{
			name:      "should error when delete fails",
			ID:        "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			expLease:  nil,
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriter{}
			mocksRwd.On("Get", tt.ID).
				Return(tt.expLease, tt.returnErr)
			mocksRwd.On("Write", mock.Anything, mock.Anything).
				Return(tt.returnErr)

			mocksEvents := &mocks.Eventer{}
			mocksEvents.On("LeaseEnd", mock.AnythingOfType("*lease.Lease")).Return(nil)

			leaseSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc:  mocksRwd,
					EventSvc: mocksEvents,
				},
			)
			actualLease, err := leaseSvc.Delete(tt.ID)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.Equal(t, tt.expLease, actualLease)

		})
	}
}

func TestSave(t *testing.T) {
	now := time.Now().Unix()

	type response struct {
		data *lease.Lease
		err  error
	}

	tests := []struct {
		name      string
		returnErr error
		lease     *lease.Lease
		exp       response
	}{
		{
			name: "should save lease with timestamps",
			lease: &lease.Lease{
				ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:         lease.StatusActive.StatusPtr(),
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("test:arn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			exp: response{
				data: &lease.Lease{
					ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:         lease.StatusActive.StatusPtr(),
					AccountID:      ptrString("123456789012"),
					PrincipalID:    ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should save with new created on",
			lease: &lease.Lease{
				ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:      lease.StatusActive.StatusPtr(),
				PrincipalID: ptrString("test:arn"),
				AccountID:   ptrString("123456789012"),
			},
			exp: response{
				data: &lease.Lease{
					ID:               ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:           lease.StatusActive.StatusPtr(),
					PrincipalID:      ptrString("test:arn"),
					AccountID:        ptrString("123456789012"),
					LastModifiedOn:   &now,
					CreatedOn:        &now,
					StatusModifiedOn: &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should fail on return err",
			lease: &lease.Lease{
				ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:      lease.StatusActive.StatusPtr(),
				PrincipalID: ptrString("test:arn"),
				AccountID:   ptrString("123456789012"),
			},
			exp: response{
				data: &lease.Lease{
					ID:               ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:           lease.StatusActive.StatusPtr(),
					PrincipalID:      ptrString("test:arn"),
					AccountID:        ptrString("123456789012"),
					LastModifiedOn:   &now,
					CreatedOn:        &now,
					StatusModifiedOn: &now,
				},
				err: errors.NewInternalServer("failure", nil),
			},
			returnErr: errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriter{}

			mocksRwd.On("Write", mock.AnythingOfType("*lease.Lease"), mock.AnythingOfType("*int64")).Return(tt.returnErr)

			leaseSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc: mocksRwd,
				},
			)

			err := leaseSvc.Save(tt.lease)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, tt.lease)

		})
	}
}

func TestGetLeases(t *testing.T) {

	type response struct {
		data *lease.Leases
		err  error
	}

	tests := []struct {
		name      string
		inputData lease.Lease
		ret       response
		exp       response
	}{
		{
			name: "standard",
			inputData: lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			ret: response{
				data: &lease.Leases{
					lease.Lease{
						ID:     aws.String("1"),
						Status: lease.StatusActive.StatusPtr(),
					},
					lease.Lease{
						ID:     aws.String("2"),
						Status: lease.StatusActive.StatusPtr(),
					},
				},
				err: nil,
			},
			exp: response{
				data: &lease.Leases{
					lease.Lease{
						ID:     ptrString("1"),
						Status: lease.StatusActive.StatusPtr(),
					},
					lease.Lease{
						ID:     ptrString("2"),
						Status: lease.StatusActive.StatusPtr(),
					},
				},
				err: nil,
			},
		},
		{
			name: "internal error",
			inputData: lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			},
		},
		{
			name: "validation error",
			inputData: lease.Lease{
				ID: ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
			},
			ret: response{
				data: nil,
				err:  nil,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("id: must be empty.")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRWD := &mocks.ReaderWriter{}
			mocksRWD.On("List", mock.AnythingOfType("*lease.Lease")).Return(tt.ret.data, tt.ret.err)

			leasesSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc: mocksRWD,
				},
			)

			leases, err := leasesSvc.List(&tt.inputData)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, leases)
		})
	}

}

func TestCreate(t *testing.T) {

	type response struct {
		data *lease.Lease
		err  error
	}

	leaseExpiresAfterAWeek := time.Now().AddDate(0, 0, 7).Unix()
	leaseExpiresAfterAYear := time.Now().AddDate(1, 0, 0).Unix()
	leaseExpiresYesterday := time.Now().AddDate(0, 0, -1).Unix()
	timeNow := time.Now().Unix()

	tests := []struct {
		name                 string
		req                  *lease.Lease
		exp                  response
		getResponse          *lease.Leases
		writeErr             error
		leaseCreateErr       error
		principalSpentAmount float64
	}{
		{
			name: "should create",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
			},
			exp: response{
				data: &lease.Lease{
					ID:                       ptrString("6d666a28-4f2c-43af-8c94-1b715ca079ae"),
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					Status:                   lease.StatusActive.StatusPtr(),
					BudgetAmount:             ptrFloat(200.00),
					BudgetCurrency:           ptrString("USD"),
					BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
					CreatedOn:                &timeNow,
					LastModifiedOn:           &timeNow,
					StatusModifiedOn:         &timeNow,
					ExpiresOn:                &leaseExpiresAfterAWeek,
				},
				err: nil,
			},
			getResponse:          nil,
			writeErr:             nil,
			leaseCreateErr:       nil,
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on lease validation error caused by budget amount greater than max lease budget amount",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(2000.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("budgetAmount: Requested lease has a budget amount of 2000.000000, which is greater than max lease budget amount of 1000.000000.")),
			},
			getResponse: &lease.Leases{
				lease.Lease{
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					BudgetAmount:             ptrFloat(200.00),
					BudgetCurrency:           ptrString("USD"),
					BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
					Metadata:                 map[string]interface{}{},
				},
			},
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on lease validation error caused by user already over principal budget amount",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("budgetAmount: Unable to create lease: User principal User1 has already spent 2000.00 of their 1000.00 principal budget.")),
			},
			getResponse: &lease.Leases{
				lease.Lease{
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					BudgetAmount:             ptrFloat(200.00),
					BudgetCurrency:           ptrString("USD"),
					BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
					Metadata:                 map[string]interface{}{},
				},
			},
			principalSpentAmount: 2000.0,
		},
		{
			name: "should fail on lease expires yesterday",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresYesterday,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("expiresOn: Requested lease has a desired expiry date less than today: %d.", leaseExpiresYesterday)),
			},
			getResponse: &lease.Leases{
				lease.Lease{
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					BudgetAmount:             ptrFloat(200.00),
					BudgetCurrency:           ptrString("USD"),
					BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
					Metadata:                 map[string]interface{}{},
				},
			},
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on lease expires after a year",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresAfterAYear,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("expiresOn: Requested lease has a budget expires on of %d, which is greater than max lease period of 704800.", leaseExpiresAfterAYear)),
			},
			getResponse: &lease.Leases{
				lease.Lease{
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					BudgetAmount:             ptrFloat(200.00),
					BudgetCurrency:           ptrString("USD"),
					BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
					Metadata:                 map[string]interface{}{},
				},
			},
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on principalId missing",
			req: &lease.Lease{
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresAfterAWeek,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("principalId: must be a string.")),
			},
			getResponse:          nil,
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on accountId missing",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresAfterAWeek,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("accountId: must be a string.")),
			},
			getResponse:          nil,
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on leaseId must be empty",
			req: &lease.Lease{
				ID:                       ptrString(""),
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresAfterAWeek,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("id: must be empty.")),
			},
			getResponse:          nil,
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on status and statusReason must be empty",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				Status:                   lease.StatusActive.StatusPtr(),
				StatusReason:             lease.StatusReasonExpired.StatusReasonPtr(),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresAfterAWeek,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("leaseStatus: must be empty; leaseStatusReason: must be empty.")),
			},
			getResponse:          nil,
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on createdOn must be empty",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				CreatedOn:                &timeNow,
				LastModifiedOn:           &timeNow,
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
				ExpiresOn:                &leaseExpiresAfterAWeek,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("createdOn: must be empty.")),
			},
			getResponse:          nil,
			principalSpentAmount: 0.0,
		},
		{
			name: "should fail on lease already exists",
			req: &lease.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
			},
			exp: response{
				data: nil,
				err:  errors.NewAlreadyExists("lease", "User1"),
			},
			getResponse: &lease.Leases{
				lease.Lease{
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					Status:                   lease.StatusActive.StatusPtr(),
					BudgetAmount:             ptrFloat(200.00),
					BudgetCurrency:           ptrString("USD"),
					BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
					Metadata:                 map[string]interface{}{},
				},
			},
			principalSpentAmount: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mocksRwd := &mocks.ReaderWriter{}
			mocksEventer := &mocks.Eventer{}

			mocksRwd.On("List", mock.AnythingOfType("*lease.Lease")).Return(tt.getResponse, nil)
			mocksRwd.On("Write", mock.AnythingOfType("*lease.Lease"), mock.AnythingOfType("*int64")).Return(tt.writeErr)
			mocksEventer.On("LeaseCreate", mock.AnythingOfType("*lease.Lease")).Return(nil)

			leaseSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc:                  mocksRwd,
					EventSvc:                 mocksEventer,
					DefaultLeaseLengthInDays: 7,
					PrincipalBudgetAmount:    1000.00,
					PrincipalBudgetPeriod:    "Weekly",
					MaxLeaseBudgetAmount:     1000.00,
					MaxLeasePeriod:           704800,
				},
			)

			result, err := leaseSvc.Create(tt.req, tt.principalSpentAmount)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			if result != nil {
				result.ID = tt.exp.data.ID
			}
			assert.Equal(t, tt.exp.data, result)
		})
	}
}
