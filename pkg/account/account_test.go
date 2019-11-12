package account

import (
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
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
		accountStatus := model.AccountStatus("Ready")
		mocksDeleter.On("Delete", mock.Anything).
			Return(nil)

		account := Account{
			data: model.Account{
				ID:     &accountID,
				Status: &accountStatus,
			},
		}
		err := account.Delete(mocksDeleter)
		assert.NoError(t, err)
	})

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
		mocksWriter := &dataMocks.Writer{}
		accountStatus := model.AccountStatus("Ready")
		lastModifiedOn := int64(1573592058)

		mocksWriter.On("Update", mock.Anything, mock.Anything).
			Return(nil)

		account := Account{
			data: model.Account{
				ID:             &accountID,
				Status:         &accountStatus,
				LastModifiedOn: &lastModifiedOn,
			},
		}

		newStatus := model.Leased
		err := account.UpdateStatus(newStatus, mocksWriter)
		assert.NoError(t, err)
		assert.Equal(t, *account.data.Status, newStatus)
		assert.NotEqual(t, *account.data.LastModifiedOn, int64(1573592058))
	})

}

func TestUpdate(t *testing.T) {

	t.Run("should Update", func(t *testing.T) {
		accountID := "123456789012"
		accountStatusReady := model.Ready
		accountStatusNotReady := model.NotReady
		lastModifiedOn := int64(1573592058)
		mocksWriter := &dataMocks.Writer{}
		mocksManager := &dataMocks.Manager{}
		metadata := map[string]interface{}{
			"key": "value",
		}

		mocksWriter.On("Update", mock.MatchedBy(func(input *model.Account) bool {
			return (*input.ID == accountID &&
				*input.Status == "NotReady" &&
				input.Metadata["key"] == "value")
		}), mock.AnythingOfType("*int64")).Return().
			Return(nil)

		mocksManager.On("Setup", mock.AnythingOfType("string")).Return(nil)

		account := Account{
			data: model.Account{
				ID:             &accountID,
				Status:         &accountStatusReady,
				LastModifiedOn: &lastModifiedOn,
			},
		}

		newData := model.Account{
			ID:     &accountID,
			Status: &accountStatusNotReady,
		}

		account.data.Metadata = metadata
		err := account.Update(newData, mocksWriter, mocksManager)

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
		accountStatus := model.Ready

		mocksReader.On("GetAccountsByStatus", "Ready").
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

		readyAccount, err := GetReadyAccount(mocksReader)
		assert.NoError(t, err)
		assert.Equal(t, *readyAccount.data.ID, "1")
		assert.Equal(t, *readyAccount.data.Status, model.AccountStatus("Ready"))
	})

}
