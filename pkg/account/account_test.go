package account

import (
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrInt64(i int64) *int64 {
	ptrI := i
	return &ptrI
}

func TestProperties(t *testing.T) {
	accountStatusReady := model.Ready
	tests := []struct {
		name    string
		account model.Account
	}{
		{
			name: "standard",
			account: model.Account{
				ID:           ptrString("abc123"),
				Status:       &accountStatusReady,
				AdminRoleArn: ptrString("test:arn"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := New(nil, tt.account)
			assert.Equal(t, tt.account.ID, account.ID())
			assert.Equal(t, tt.account.AdminRoleArn, account.AdminRoleArn())
			assert.Equal(t, tt.account.Metadata, account.Metadata())
			assert.Equal(t, tt.account.PrincipalRoleArn, account.PrincipalRoleArn())
			assert.Equal(t, tt.account.PrincipalPolicyHash, account.PrincipalPolicyHash())
		})
	}
}

func TestGet(t *testing.T) {

	t.Run("should return an account object", func(t *testing.T) {
		mocksReader := &dataMocks.Reader{}
		mocksWriter := &dataMocks.WriterDeleter{}

		accountID := "abc123"
		mocksReader.On("GetAccountByID", accountID, mock.Anything).
			Return(nil)

		account, err := GetAccountByID(accountID, mocksReader, mocksWriter)
		assert.NoError(t, err)
		assert.NotNil(t, account)
	})

}

func TestDelete(t *testing.T) {
	accountStatusReady := model.Ready
	accountStatusLeased := model.Leased
	tests := []struct {
		name    string
		account model.Account
		errMsg  string
	}{
		{
			name: "should delete an account",
			account: model.Account{
				ID:     ptrString("abc123"),
				Status: &accountStatusReady,
			},
		},
		{
			name: "should error when account leased",
			account: model.Account{
				ID:     ptrString("abc123"),
				Status: &accountStatusLeased,
			},
			errMsg: "operation cannot be fulfilled on account \"abc123\": accountStatus: must not be leased.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksDeleter := &dataMocks.WriterDeleter{}
			mocksDeleter.On("DeleteAccount", mock.Anything).
				Return(nil)
			account := New(mocksDeleter, tt.account)

			err := account.Delete()
			if tt.errMsg != "" {
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.Nil(t, err)
			}

		})
	}
}

func TestMarshallJSON(t *testing.T) {

	t.Run("should marshall into JSON", func(t *testing.T) {
		accountID := "abc123"
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
			"{\"id\":\"abc123\",\"accountStatus\":\"Ready\"}",
			string(b))
	})

}

func TestUpdatStatus(t *testing.T) {

	t.Run("should Update status", func(t *testing.T) {
		accountID := "abc123"
		mocksWriter := &dataMocks.WriterDeleter{}
		accountStatus := model.AccountStatus("Ready")
		lastModifiedOn := int64(1573592058)

		mocksWriter.On("WriteAccount", mock.Anything, mock.Anything).
			Return(nil)

		account := Account{
			writer: mocksWriter,
			data: model.Account{
				ID:             &accountID,
				Status:         &accountStatus,
				LastModifiedOn: &lastModifiedOn,
				CreatedOn:      &lastModifiedOn,
				AdminRoleArn:   ptrString("test:arn"),
			},
		}

		newStatus := model.Leased
		err := account.UpdateStatus(newStatus)
		assert.NoError(t, err)
		assert.Equal(t, *account.data.Status, newStatus)
		assert.NotEqual(t, *account.data.LastModifiedOn, int64(1573592058))
	})

}

func TestUpdate(t *testing.T) {

	t.Run("should Update", func(t *testing.T) {
		accountID := "123456789012"
		accountStatusReady := model.Ready
		mocksWriter := &dataMocks.WriterDeleter{}
		mocksManager := &dataMocks.Manager{}
		metadata := map[string]interface{}{
			"key": "value",
		}

		mocksWriter.On("WriteAccount", mock.MatchedBy(func(input *model.Account) bool {
			return (*input.ID == accountID &&
				*input.Status == "Ready" &&
				input.Metadata["key"] == "value")
		}), mock.AnythingOfType("*int64")).Return().
			Return(nil)

		mocksManager.On("Setup", mock.AnythingOfType("string")).Return(nil)

		account := New(
			mocksWriter,
			model.Account{
				ID:           &accountID,
				Status:       &accountStatusReady,
				AdminRoleArn: ptrString("test:arn"),
			})

		newData := model.Account{
			Metadata: metadata,
		}

		err := account.Update(newData, mocksManager)

		assert.NoError(t, err)
		assert.NotEqual(t, *account.data.LastModifiedOn, int64(1573592058))
		assert.Equal(t, account.data.Metadata, map[string]interface{}{
			"key": "value",
		})
	})

}

func TestGetReadyAccount(t *testing.T) {

	t.Run("should be able to get a ready account", func(t *testing.T) {
		mocksReader := &dataMocks.Reader{}
		mocksWriter := &dataMocks.WriterDeleter{}
		accountStatus := model.Ready

		mocksReader.On("GetAccounts", &model.Account{
			Status: model.Ready.AccountStatusPtr(),
		}).
			Return(
				&model.Accounts{
					model.Account{
						ID:     aws.String("1"),
						Status: &accountStatus,
					},
					model.Account{
						ID:     aws.String("2"),
						Status: &accountStatus,
					},
				}, nil,
			)

		readyAccount, err := GetReadyAccount(mocksReader, mocksWriter)
		assert.NoError(t, err)
		assert.Equal(t, *readyAccount.data.ID, "1")
		assert.Equal(t, *readyAccount.data.Status, model.AccountStatus("Ready"))
	})

}
