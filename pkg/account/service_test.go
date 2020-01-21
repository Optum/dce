package account_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func TestGetAccountByID(t *testing.T) {

	type response struct {
		data *account.Account
		err  error
	}

	tests := []struct {
		name string
		ID   string
		ret  response
		exp  response
	}{
		{
			name: "should get an account by ID",
			ID:   "123456789012",
			ret: response{
				data: &account.Account{
					ID:     ptrString("123456789012"),
					Status: account.StatusReady.StatusPtr(),
				},
				err: nil,
			},
			exp: response{
				data: &account.Account{
					ID:     ptrString("123456789012"),
					Status: account.StatusReady.StatusPtr(),
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
			mocksRwd := &mocks.ReaderWriterDeleter{}

			mocksRwd.On("Get", tt.ID).Return(tt.ret.data, tt.ret.err)

			accountSvc := account.NewService(account.NewServiceInput{
				DataSvc: mocksRwd,
			})

			getAccount, err := accountSvc.Get(tt.ID)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, getAccount)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		expErr    error
		returnErr error
		account   account.Account
	}{
		{
			name: "should delete an account",
			account: account.Account{
				ID:     ptrString("123456789012"),
				Status: account.StatusReady.StatusPtr(),
			},
			returnErr: nil,
		},
		{
			name: "should error when account leased",
			account: account.Account{
				ID:     ptrString("123456789012"),
				Status: account.StatusLeased.StatusPtr(),
			},
			returnErr: nil,
			expErr:    errors.NewConflict("account", "123456789012", fmt.Errorf("accountStatus: must not be leased.")), //nolint golint
		},
		{
			name: "should error when delete fails",
			account: account.Account{
				ID:     ptrString("123456789012"),
				Status: account.StatusReady.StatusPtr(),
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}
			mocksRwd.On("Delete", mock.Anything).
				Return(tt.returnErr)

			accountSvc := account.NewService(
				account.NewServiceInput{
					DataSvc: mocksRwd,
				},
			)
			err := accountSvc.Delete(&tt.account)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}

func TestUpdate(t *testing.T) {
	now := time.Now().Unix()

	type response struct {
		data *account.Account
		err  error
	}

	tests := []struct {
		name        string
		returnErr   error
		amReturnErr error
		origAccount account.Account
		updAccount  account.Account
		exp         response
	}{
		{
			name: "should update",
			origAccount: account.Account{
				ID:             ptrString("123456789012"),
				Status:         account.StatusReady.StatusPtr(),
				AdminRoleArn:   ptrString("test:arn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			updAccount: account.Account{
				AdminRoleArn: ptrString("test:arn:new"),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			exp: response{
				data: &account.Account{
					ID:           ptrString("123456789012"),
					Status:       account.StatusReady.StatusPtr(),
					AdminRoleArn: ptrString("test:arn:new"),
					Metadata: map[string]interface{}{
						"key": "value",
					},
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should fail validation on update",
			origAccount: account.Account{
				ID:     ptrString("123456789012"),
				Status: account.StatusReady.StatusPtr(),
			},
			updAccount: account.Account{
				ID: ptrString("abc125"),
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("account", fmt.Errorf("id: must be empty.")), //nolint golint
			},
			returnErr: nil,
		},
		{
			name: "should fail on save",
			origAccount: account.Account{
				ID:           ptrString("123456789012"),
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
			updAccount: account.Account{
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", nil),
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}
			mocksManager := &mocks.Manager{}

			mocksRwd.On("Get", *tt.origAccount.ID).Return(&tt.origAccount, tt.returnErr)
			mocksRwd.On("Write", mock.AnythingOfType("*account.Account"), mock.AnythingOfType("*int64")).Return(tt.returnErr)

			mocksManager.On("Setup", mock.AnythingOfType("string")).Return(tt.amReturnErr)

			accountSvc := account.NewService(
				account.NewServiceInput{
					DataSvc:    mocksRwd,
					ManagerSvc: mocksManager,
				},
			)

			result, err := accountSvc.Update(*tt.origAccount.ID, &tt.updAccount)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, result)

		})
	}
}

func TestSave(t *testing.T) {
	now := time.Now().Unix()

	type response struct {
		data *account.Account
		err  error
	}

	tests := []struct {
		name      string
		returnErr error
		account   *account.Account
		exp       response
	}{
		{
			name: "should save account with timestamps",
			account: &account.Account{
				ID:             ptrString("123456789012"),
				Status:         account.StatusReady.StatusPtr(),
				AdminRoleArn:   ptrString("test:arn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			exp: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusReady.StatusPtr(),
					AdminRoleArn:   ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should save with new created on",
			account: &account.Account{
				ID:           ptrString("123456789012"),
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
			exp: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusReady.StatusPtr(),
					AdminRoleArn:   ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should fail on return err",
			account: &account.Account{
				ID:           ptrString("123456789012"),
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
			exp: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusReady.StatusPtr(),
					AdminRoleArn:   ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: errors.NewInternalServer("failure", nil),
			},
			returnErr: errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}
			mocksManager := &mocks.Manager{}

			mocksRwd.On("Write", mock.AnythingOfType("*account.Account"), mock.AnythingOfType("*int64")).Return(tt.returnErr)
			mocksManager.On("Setup", mock.AnythingOfType("string")).Return(nil)

			accountSvc := account.NewService(
				account.NewServiceInput{
					DataSvc:    mocksRwd,
					ManagerSvc: mocksManager,
				},
			)

			err := accountSvc.Save(tt.account)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, tt.account)

		})
	}
}

func TestGetAccounts(t *testing.T) {

	type response struct {
		data *account.Accounts
		err  error
	}

	tests := []struct {
		name      string
		inputData account.Account
		ret       response
		exp       response
	}{
		{
			name: "standard",
			inputData: account.Account{
				Status: account.StatusReady.StatusPtr(),
			},
			ret: response{
				data: &account.Accounts{
					account.Account{
						ID:     aws.String("1"),
						Status: account.StatusReady.StatusPtr(),
					},
					account.Account{
						ID:     aws.String("2"),
						Status: account.StatusReady.StatusPtr(),
					},
				},
				err: nil,
			},
			exp: response{
				data: &account.Accounts{
					account.Account{
						ID:     ptrString("1"),
						Status: account.StatusReady.StatusPtr(),
					},
					account.Account{
						ID:     ptrString("2"),
						Status: account.StatusReady.StatusPtr(),
					},
				},
				err: nil,
			},
		},
		{
			name: "internal error",
			inputData: account.Account{
				Status: account.StatusReady.StatusPtr(),
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
			inputData: account.Account{
				ID: ptrString("123456789012"),
			},
			ret: response{
				data: nil,
				err:  nil,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("account", fmt.Errorf("id: must be empty.")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRWD := &mocks.ReaderWriterDeleter{}
			mocksRWD.On("List", mock.AnythingOfType("*account.Account")).Return(tt.ret.data, tt.ret.err)

			accountsSvc := account.NewService(
				account.NewServiceInput{
					DataSvc: mocksRWD,
				},
			)

			accounts, err := accountsSvc.List(&tt.inputData)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, accounts)
		})
	}

}
