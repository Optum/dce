package lease_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	leases "github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrInt64(i int64) *int64 {
	ptr := i
	return &ptr
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
		data *leases.Lease
		err  error
	}

	tests := []struct {
		name string
		ID   string
		ret  response
		exp  response
	}{
		{
			name: "should get a lease by ID",
			ID:   "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			ret: response{
				data: &leases.Lease{
					ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status: leases.StatusActive.StatusPtr(),
				},
				err: nil,
			},
			exp: response{
				data: &leases.Lease{
					ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status: leases.StatusActive.StatusPtr(),
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

			leaseSvc := leases.NewService(leases.NewServiceInput{
				DataSvc: mocksRwd,
			})

			getLease, err := leaseSvc.Get(tt.ID)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, getLease)
		})
	}
}

func TestGetByAccountIDAndPrincipalID(t *testing.T) {

	type response struct {
		data *leases.Lease
		err  error
	}

	tests := []struct {
		name        string
		accountId   string
		principalId string
		ret         response
		exp         response
	}{
		{
			name:        "should get a lease by accountId and principalId",
			accountId:   "123456789012",
			principalId: "TestUser",
			ret: response{
				data: &leases.Lease{
					ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("TestUser"),
					Status:      leases.StatusActive.StatusPtr(),
				},
				err: nil,
			},
			exp: response{
				data: &leases.Lease{
					ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("TestUser"),
					Status:      leases.StatusActive.StatusPtr(),
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

			mocksRwd.On("GetByAccountIDAndPrincipalID", tt.accountId, tt.principalId).Return(tt.ret.data, tt.ret.err)

			leaseSvc := leases.NewService(leases.NewServiceInput{
				DataSvc: mocksRwd,
			})

			getLease, err := leaseSvc.GetByAccountIDAndPrincipalID(tt.accountId, tt.principalId)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, getLease)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name              string
		ID                string
		StatusReason      leases.StatusReason
		mockDBWriteError  error
		mockPreviousLease *leases.Lease
		expectedError     error
	}{
		{
			name: "should delete a lease",
			ID:   "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			StatusReason: leases.StatusReasonOverPrincipalBudget,
			mockPreviousLease: &leases.Lease{
				ID:           ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				AccountID:    ptrString("123456789012"),
				Status:       leases.StatusActive.StatusPtr(),
				StatusReason: leases.StatusReasonActive.StatusReasonPtr(),
				LastModifiedOn: ptrInt64(100),
			},
			mockDBWriteError: nil,
		},
		{
			name:              "should error when delete fails",
			ID:                "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			StatusReason:      leases.StatusReasonDestroyed,
			mockPreviousLease: &leases.Lease{
				ID:           ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				AccountID:    ptrString("123456789012"),
				Status:       leases.StatusActive.StatusPtr(),
				StatusReason: leases.StatusReasonActive.StatusReasonPtr(),
				LastModifiedOn: ptrInt64(100),
			},
			mockDBWriteError:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expectedError:     errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDataSvc := &mocks.ReaderWriter{}

			// Mock the DB query for the existing leases record (before writing to it)
			mockDataSvc.On("Get", tt.ID).
				Return(tt.mockPreviousLease, tt.mockDBWriteError)

			// Assert that we write the updated leases to the DB,
			// via the leases data svc
			mockDataSvc.On("Write", mock.MatchedBy(func(lease *leases.Lease) bool {
				assert.Equal(t, tt.StatusReason.StatusReasonPtr(), lease.StatusReason,
					"should update lease with provided StatusReason")
				assert.Equal(t, leases.StatusInactive.StatusPtr(), lease.Status,
					"should mark lease as Status=Inactive")

				now := time.Now().Unix()
				assert.InDelta(t, now, *lease.LastModifiedOn, 30,
					"should update LastModifiedOn timestamp to current time")
				assert.InDelta(t, now, *lease.StatusModifiedOn, 30,
					"should update StatusModifiedOn timestamp to current time")

				return true
			}), tt.mockPreviousLease.LastModifiedOn).
				Return(tt.mockDBWriteError)

			// Mock the AccountSvc.Reset() method
			// which adds our account to the reset SQS queue
			mocksAccountSvc := &mocks.AccountServicer{}
			mocksAccountSvc.On("Reset", mock.AnythingOfType("string")).
				Return(nil, nil)

			// Mock the `EventSvc.LeaseEnd()` method.
			// which triggers an SNS event
			mocksEvents := &mocks.Eventer{}
			mocksEvents.On("LeaseEnd", mock.Anything).Return(nil)

			leaseSvc := leases.NewService(
				leases.NewServiceInput{
					DataSvc:    mockDataSvc,
					EventSvc:   mocksEvents,
					AccountSvc: mocksAccountSvc,
				},
			)
			result, err := leaseSvc.Delete(tt.ID, tt.StatusReason)
			assert.True(t, errors.Is(err, tt.expectedError), "actual error %q doesn't match expected error %q", err, tt.expectedError)

			if tt.expectedError == nil {
				assert.Equal(t, tt.ID, *result.ID,
					"returned lease has lease ID")
				assert.Equal(t, leases.StatusInactive, *result.Status,
					"returned lease is Status=Inactive")
				assert.Equal(t, tt.StatusReason, *result.StatusReason,
					"returned lease has updated status reason")

				mockDataSvc.AssertExpectations(t)
				mocksAccountSvc.AssertExpectations(t)
				mocksEvents.AssertExpectations(t)
			}
		})
	}
}

func TestSave(t *testing.T) {
	now := time.Now().Unix()

	type response struct {
		data *leases.Lease
		err  error
	}

	tests := []struct {
		name      string
		returnErr error
		lease     *leases.Lease
		exp       response
	}{
		{
			name: "should save leases with timestamps",
			lease: &leases.Lease{
				ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:         leases.StatusActive.StatusPtr(),
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("test:arn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			exp: response{
				data: &leases.Lease{
					ID:               ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:           leases.StatusActive.StatusPtr(),
					AccountID:        ptrString("123456789012"),
					PrincipalID:      ptrString("test:arn"),
					LastModifiedOn:   &now,
					CreatedOn:        &now,
					StatusModifiedOn: &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should save with new created on",
			lease: &leases.Lease{
				ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:      leases.StatusActive.StatusPtr(),
				PrincipalID: ptrString("test:arn"),
				AccountID:   ptrString("123456789012"),
			},
			exp: response{
				data: &leases.Lease{
					ID:               ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:           leases.StatusActive.StatusPtr(),
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
			lease: &leases.Lease{
				ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:      leases.StatusActive.StatusPtr(),
				PrincipalID: ptrString("test:arn"),
				AccountID:   ptrString("123456789012"),
			},
			exp: response{
				data: &leases.Lease{
					ID:               ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:           leases.StatusActive.StatusPtr(),
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

			leaseSvc := leases.NewService(
				leases.NewServiceInput{
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
		data *leases.Leases
		err  error
	}

	tests := []struct {
		name      string
		inputData leases.Lease
		ret       response
		exp       response
	}{
		{
			name: "standard",
			inputData: leases.Lease{
				Status: leases.StatusActive.StatusPtr(),
			},
			ret: response{
				data: &leases.Leases{
					leases.Lease{
						ID:     aws.String("1"),
						Status: leases.StatusActive.StatusPtr(),
					},
					leases.Lease{
						ID:     aws.String("2"),
						Status: leases.StatusActive.StatusPtr(),
					},
				},
				err: nil,
			},
			exp: response{
				data: &leases.Leases{
					leases.Lease{
						ID:     ptrString("1"),
						Status: leases.StatusActive.StatusPtr(),
					},
					leases.Lease{
						ID:     ptrString("2"),
						Status: leases.StatusActive.StatusPtr(),
					},
				},
				err: nil,
			},
		},
		{
			name: "internal error",
			inputData: leases.Lease{
				Status: leases.StatusActive.StatusPtr(),
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
			inputData: leases.Lease{
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

			leasesSvc := leases.NewService(
				leases.NewServiceInput{
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
		data *leases.Lease
		err  error
	}

	leaseExpiresAfterAWeek := time.Now().AddDate(0, 0, 7).Unix()
	leaseExpiresAfterAYear := time.Now().AddDate(1, 0, 0).Unix()
	leaseExpiresYesterday := time.Now().AddDate(0, 0, -1).Unix()
	timeNow := time.Now().Unix()

	tests := []struct {
		name                 string
		req                  *leases.Lease
		exp                  response
		getResponse          *leases.Leases
		writeErr             error
		leaseCreateErr       error
		principalSpentAmount float64
	}{
		{
			name: "should create",
			req: &leases.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
			},
			exp: response{
				data: &leases.Lease{
					ID:                       ptrString("6d666a28-4f2c-43af-8c94-1b715ca079ae"),
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					Status:                   leases.StatusActive.StatusPtr(),
					StatusReason:             leases.StatusReasonActive.StatusReasonPtr(),
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
			req: &leases.Lease{
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
			getResponse: &leases.Leases{
				leases.Lease{
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
			name: "should fail on lease expires yesterday",
			req: &leases.Lease{
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
			getResponse: &leases.Leases{
				leases.Lease{
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
			req: &leases.Lease{
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
			getResponse: &leases.Leases{
				leases.Lease{
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
			req: &leases.Lease{
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
			req: &leases.Lease{
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
			req: &leases.Lease{
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
			req: &leases.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				Status:                   leases.StatusActive.StatusPtr(),
				StatusReason:             leases.StatusReasonExpired.StatusReasonPtr(),
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
			name: "should fail on lease already exists",
			req: &leases.Lease{
				PrincipalID:              ptrString("User1"),
				AccountID:                ptrString("123456789012"),
				BudgetAmount:             ptrFloat(200.00),
				BudgetCurrency:           ptrString("USD"),
				BudgetNotificationEmails: ptrArrayString([]string{"test1@test.com", "test2@test.com"}),
				Metadata:                 map[string]interface{}{},
			},
			exp: response{
				data: nil,
				err:  errors.NewAlreadyExists("lease", "with principal User1 and account 123456789012"),
			},
			getResponse: &leases.Leases{
				leases.Lease{
					PrincipalID:              ptrString("User1"),
					AccountID:                ptrString("123456789012"),
					Status:                   leases.StatusActive.StatusPtr(),
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

			mocksAccountSvc := &mocks.AccountServicer{}

			mocksRwd.On("List", mock.AnythingOfType("*lease.Lease")).Return(tt.getResponse, nil)
			mocksRwd.On("Write", mock.AnythingOfType("*lease.Lease"), mock.AnythingOfType("*int64")).Return(tt.writeErr)
			mocksEventer.On("LeaseCreate", mock.AnythingOfType("*lease.Lease")).Return(nil)

			leaseSvc := leases.NewService(
				leases.NewServiceInput{
					DataSvc:                  mocksRwd,
					EventSvc:                 mocksEventer,
					AccountSvc:               mocksAccountSvc,
					DefaultLeaseLengthInDays: 7,
					PrincipalBudgetAmount:    1000.00,
					PrincipalBudgetPeriod:    "Weekly",
					MaxLeaseBudgetAmount:     1000.00,
					MaxLeasePeriod:           704800,
				},
			)

			result, err := leaseSvc.Create(tt.req)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			if result != nil {
				result.ID = tt.exp.data.ID
			}
			assert.Equal(t, tt.exp.data, result)
		})
	}
}
