package account

import (
	"log"
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGet(t *testing.T) {

	t.Run("should return an account object", func(t *testing.T) {
		mocksReader := &dataMocks.Reader{}

		accountID := "abc123"
		mocksReader.On("GetAccountByID", accountID, mock.Anything).
			Return(nil)

		account, err := GetAccountByID(accountID, mocksReader)
		assert.NoError(t, err)
		assert.NotNil(t, account)
	})

}

func TestDelete(t *testing.T) {

	t.Run("should delete an account", func(t *testing.T) {
		accountID := "abc123"
		mocksDeleter := &dataMocks.Deleter{}

		mocksDeleter.On("Delete", mock.Anything).
			Return(nil)

		account := Account{
			data: model.Account{
				ID:     accountID,
				Status: model.AccountStatus("Ready"),
			},
		}
		err := account.Delete(mocksDeleter)
		assert.NoError(t, err)
	})

}

func TestMarshallJSON(t *testing.T) {

	t.Run("should marshall into JSON", func(t *testing.T) {
		accountID := "abc123"

		account := Account{
			data: model.Account{
				ID:     accountID,
				Status: model.AccountStatus("Ready"),
			},
		}
		b, err := account.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t,
			"{\"id\":\"abc123\",\"accountStatus\":\"Ready\",\"lastModifiedOn\":0,\"createdOn\":0,\"adminRoleArn\":\"\",\"principalRoleArn\":\"\",\"principalPolicyHash\":\"\",\"metadata\":null}",
			string(b))
	})

}

func TestUpdatStatus(t *testing.T) {

	t.Run("should Update status", func(t *testing.T) {
		accountID := "abc123"
		mocksWriter := &dataMocks.Writer{}

		mocksWriter.On("Update", mock.Anything, mock.Anything).
			Return(nil)

		account := Account{
			data: model.Account{
				ID:             accountID,
				Status:         model.AccountStatus("Ready"),
				LastModifiedOn: int64(1573592058),
			},
		}

		newStatus := model.Leased
		err := account.UpdateStatus(newStatus, mocksWriter)
		assert.NoError(t, err)
		assert.Equal(t, account.data.Status, newStatus)
		assert.NotEqual(t, account.data.LastModifiedOn, int64(1573592058))
	})

}

func TestUpdate(t *testing.T) {

	t.Run("should Update", func(t *testing.T) {
		accountID := "abc123"
		mocksWriter := &dataMocks.Writer{}

		mocksWriter.On("Update", mock.MatchedBy(func(input *model.Account) bool {
			log.Printf("Test")
			return (input.ID == "abc123" &&
				input.Status == model.AccountStatus("Ready") &&
				input.Metadata["key"] == "value")
		}), mock.AnythingOfType("*int64")).Return().
			Return(nil)

		account := Account{
			data: model.Account{
				ID:             accountID,
				Status:         model.AccountStatus("Ready"),
				LastModifiedOn: int64(1573592058),
			},
		}

		account.data.Metadata = map[string]interface{}{
			"key": "value",
		}
		err := account.Update(mocksWriter)

		assert.NoError(t, err)
		assert.NotEqual(t, account.data.LastModifiedOn, int64(1573592058))
		assert.Equal(t, account.data.Metadata, map[string]interface{}{
			"key": "value",
		})
	})

}

func TestAssumeRole(t *testing.T) {

	t.Run("should be able to assume role", func(t *testing.T) {

		accountID := "abc123"
		Status := model.AccountStatus("Ready")

		mockTokenService := commonMocks.TokenService{}
		mockTokenService.On("NewSession", "aws:role:adminrole").Return(nil, nil)

		// awsSession, err := session.NewSession()
		account := Account{
			data: model.Account{
				ID:             accountID,
				Status:         Status,
				AdminRoleArn:   "aws:role:adminrole",
				LastModifiedOn: 1573592058,
			},
		}
		newSession, err := account.AssumeAdminRole()
		assert.NoError(t, err)
		assert.NotNil(t, newSession)
	})

}

func TestGetReadyAccount(t *testing.T) {

	t.Run("should be able to get a ready account", func(t *testing.T) {
		mocksReader := &dataMocks.Reader{}

		mocksReader.On("GetAccountsByStatus", "Ready").
			Return(
				&model.Accounts{
					model.Account{
						ID:     "1",
						Status: model.Ready,
					},
					model.Account{
						ID:     "2",
						Status: model.Ready,
					},
				}, nil,
			)

		readyAccount, err := GetReadyAccount(mocksReader)
		assert.NoError(t, err)
		assert.Equal(t, readyAccount.data.ID, "1")
		assert.Equal(t, readyAccount.data.Status, model.AccountStatus("Ready"))
	})

}
