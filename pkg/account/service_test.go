package account_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/arn"
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
				AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			updAccount: account.Account{
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			exp: response{
				data: &account.Account{
					ID:           ptrString("123456789012"),
					Status:       account.StatusReady.StatusPtr(),
					AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
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
				err:  errors.NewValidation("account", fmt.Errorf("id: must be a valid value.")), //nolint golint
			},
			returnErr: nil,
		},
		{
			name: "should fail on save",
			origAccount: account.Account{
				ID:           ptrString("123456789012"),
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
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

			mocksManager.On("ValidateAccess", mock.AnythingOfType("*arn.ARN")).Return(tt.amReturnErr)

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
				AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			exp: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusReady.StatusPtr(),
					AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
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
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
			},
			exp: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusReady.StatusPtr(),
					AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
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
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
			},
			exp: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusReady.StatusPtr(),
					AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
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

func TestCreate(t *testing.T) {
	now := time.Now().Unix()

	type response struct {
		data *account.Account
		err  error
	}

	tests := []struct {
		name             string
		req              *account.Account
		exp              response
		getResponse      response
		writeErr         error
		accountCreateErr error
		accountResetErr  error
	}{
		{
			name: "should create",
			req: &account.Account{
				ID:           ptrString("123456789012"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			exp: response{
				data: &account.Account{
					ID:                 ptrString("123456789012"),
					Status:             account.StatusNotReady.StatusPtr(),
					AdminRoleArn:       arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
					LastModifiedOn:     &now,
					CreatedOn:          &now,
					PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
					PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
				},
				err: nil,
			},
			getResponse: response{
				data: nil,
				err:  errors.NewNotFound("account", "123456789012"),
			},
			writeErr:         nil,
			accountCreateErr: nil,
			accountResetErr:  nil,
		},
		{
			name: "should fail on account already exists",
			req: &account.Account{
				ID:           ptrString("123456789012"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			exp: response{
				data: nil,
				err:  errors.NewAlreadyExists("account", "123456789012"),
			},
			getResponse: response{
				data: &account.Account{
					ID:             ptrString("123456789012"),
					Status:         account.StatusNotReady.StatusPtr(),
					AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
		},
		{
			name: "should fail on get error",
			req: &account.Account{
				ID:           ptrString("123456789012"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
			getResponse: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
		},
		{
			name: "should fail on save",
			req: &account.Account{
				ID:           ptrString("123456789012"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
			getResponse: response{
				data: nil,
				err:  errors.NewNotFound("account", "123456789012"),
			},
			writeErr: errors.NewInternalServer("error", nil),
		},
		{
			name: "should fail on publish AccountCreate event error",
			req: &account.Account{
				ID:           ptrString("123456789012"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
			getResponse: response{
				data: nil,
				err:  errors.NewNotFound("account", "123456789012"),
			},
			accountCreateErr: errors.NewInternalServer("error", nil),
		},
		{
			name: "should fail on publish AccountReset event error",
			req: &account.Account{
				ID:           ptrString("123456789012"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
			getResponse: response{
				data: nil,
				err:  errors.NewNotFound("account", "123456789012"),
			},
			accountResetErr: errors.NewInternalServer("error", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}
			mocksManager := &mocks.Manager{}
			mocksEventer := &mocks.Eventer{}

			mocksRwd.On("Get", *tt.req.ID).Return(tt.getResponse.data, tt.getResponse.err)
			mocksRwd.On("Write", mock.AnythingOfType("*account.Account"), mock.AnythingOfType("*int64")).Return(tt.writeErr)
			mocksManager.On("UpsertPrincipalAccess", mock.AnythingOfType("*account.Account")).Return(nil)
			mocksEventer.On("AccountCreate", mock.AnythingOfType("*account.Account")).Return(tt.accountCreateErr)
			mocksEventer.On("AccountReset", mock.AnythingOfType("*account.Account")).Return(tt.accountResetErr)

			accountSvc := account.NewService(
				account.NewServiceInput{
					DataSvc:           mocksRwd,
					ManagerSvc:        mocksManager,
					EventSvc:          mocksEventer,
					PrincipalRoleName: "DCEPrincipal",
				},
			)

			result, err := accountSvc.Create(tt.req)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, result)

		})
	}
}
