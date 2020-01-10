package account

import (
	"fmt"
	"testing"
	"time"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func TestProperties(t *testing.T) {
	tests := []struct {
		name    string
		account model.Account
	}{
		{
			name: "standard",
			account: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := New(tt.account, nil)
			assert.Equal(t, tt.account.ID, account.ID())
			assert.Equal(t, tt.account.AdminRoleArn, account.AdminRoleArn())
			assert.Equal(t, tt.account.Metadata, account.Metadata())
			assert.Equal(t, tt.account.PrincipalRoleArn, account.PrincipalRoleArn())
			assert.Equal(t, tt.account.PrincipalPolicyHash, account.PrincipalPolicyHash())
			assert.Equal(t, tt.account.Status, account.Status())
		})
	}
}

func TestGetAccountByID(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name       string
		ID         string
		returnData *model.Account
		returnErr  error
		expReturn  *Account
		expErr     error
	}{
		{
			name: "should get an account by ID",
			ID:   "123456789012",
			returnData: &model.Account{
				ID:     ptrString("123456789012"),
				Status: model.AccountStatusReady.AccountStatusPtr(),
			},
			returnErr: nil,
			expReturn: &Account{
				writer: nil,
				data: model.Account{
					ID:             ptrString("123456789012"),
					Status:         model.AccountStatusReady.AccountStatusPtr(),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
			},
			expErr: nil,
		},
		{
			name:       "should get failure",
			returnData: nil,
			returnErr:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expReturn:  nil,
			expErr:     errors.NewInternalServer("failure", fmt.Errorf("original failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksReader := &dataMocks.Reader{}

			mocksReader.On("GetAccountByID", tt.ID).
				Return(tt.returnData, tt.returnErr)

			account, err := GetAccountByID(tt.ID, mocksReader, nil)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.Equal(t, tt.expReturn, account)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		expErr    error
		returnErr error
		account   model.Account
	}{
		{
			name: "should delete an account",
			account: model.Account{
				ID:     ptrString("123456789012"),
				Status: model.AccountStatusReady.AccountStatusPtr(),
			},
			returnErr: nil,
		},
		{
			name: "should error when account leased",
			account: model.Account{
				ID:     ptrString("123456789012"),
				Status: model.AccountStatusLeased.AccountStatusPtr(),
			},
			returnErr: nil,
			expErr:    errors.NewConflict("account", "123456789012", fmt.Errorf("accountStatus: must not be leased.")), //nolint golint
		},
		{
			name: "should error when delete fails",
			account: model.Account{
				ID:     ptrString("123456789012"),
				Status: model.AccountStatusReady.AccountStatusPtr(),
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksDeleter := &dataMocks.WriterDeleter{}
			mocksDeleter.On("DeleteAccount", mock.Anything).
				Return(tt.returnErr)
			account := New(tt.account, mocksDeleter)

			err := account.Delete()
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}

func TestMarshallJSON(t *testing.T) {

	t.Run("should marshall into JSON", func(t *testing.T) {
		accountID := "123456789012"
		accountStatus := model.AccountStatus("Ready")

		account := Account{
			data: model.Account{
				ID:     &accountID,
				Status: &accountStatus,
			},
		}
		b, err := account.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t,
			"{\"id\":\"123456789012\",\"accountStatus\":\"Ready\"}",
			string(b))
	})

}

func TestUpdate(t *testing.T) {

	tests := []struct {
		name        string
		expErr      error
		returnErr   error
		amReturnErr error
		origAccount model.Account
		updAccount  model.Account
		expAccount  model.Account
	}{
		{
			name: "should update",
			origAccount: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
			updAccount: model.Account{
				AdminRoleArn: ptrString("test:arn:new"),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			expAccount: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			returnErr: nil,
		},
		{
			name: "should fail validation on update",
			origAccount: model.Account{
				ID:     ptrString("123456789012"),
				Status: model.AccountStatusReady.AccountStatusPtr(),
			},
			updAccount: model.Account{
				ID: ptrString("abc125"),
			},
			expAccount: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
			returnErr: nil,
			expErr:    errors.NewValidation("account", fmt.Errorf("id: must be empty.")), //nolint golint
		},
		{
			name: "should fail on save",
			origAccount: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
			updAccount: model.Account{
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			expAccount: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksWriter := &dataMocks.WriterDeleter{}
			mocksManager := &dataMocks.Manager{}

			mocksWriter.On("WriteAccount", mock.MatchedBy(func(input *model.Account) bool {
				return (*input.ID == *tt.expAccount.ID)
			}), mock.AnythingOfType("*int64")).Return(tt.returnErr)

			mocksManager.On("Setup", mock.AnythingOfType("string")).Return(tt.amReturnErr)

			account := New(tt.origAccount, mocksWriter)

			err := account.Update(tt.updAccount, mocksManager)

			assert.Truef(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.NotEqual(t, tt.origAccount.LastModifiedOn, account)
			if tt.returnErr == nil {
				assert.Equal(t, tt.expAccount.Metadata, account.data.Metadata)
			}

		})
	}
}
