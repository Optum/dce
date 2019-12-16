package account

import (
	gErrors "errors"
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAccountsByStatus(t *testing.T) {

	t.Run("should return a list of accounts by Status", func(t *testing.T) {
		mocksReader := &dataMocks.MultipleReader{}
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

		accounts, err := GetAccountsByStatus(model.Ready, mocksReader)
		assert.NoError(t, err)
		assert.Len(t, *accounts, 2)
		assert.Equal(t, *(*accounts)[0].data.ID, "1")
		assert.Equal(t, *(*accounts)[0].data.Status, model.AccountStatus("Ready"))
	})

}

func TestUpdateAccount(t *testing.T) {

	t.Run("should fail when Status is provided", func(t *testing.T) {
		mocksWriter := &dataMocks.Writer{}
		mocksManager := &dataMocks.Manager{}
		accountReadyStatus := model.Ready
		accountNotReadyStatus := model.NotReady
		accountID := "123456789012"
		mocksWriter.On("Update", mock.AnythingOfType("*model.Account"), mock.AnythingOfType("*int64")).
			Return(nil)

		mocksManager.On("Setup", "roleArn").Return(nil)

		account := Account{
			data: model.Account{
				ID:     &accountID,
				Status: &accountReadyStatus,
			},
		}
		data := model.Account{
			Status: &accountNotReadyStatus,
		}
		err := account.Update(data, mocksWriter, mocksManager)
		assert.True(t, gErrors.Is(err, &errors.ErrValidation{}))
		assert.Equal(t, *account.data.ID, accountID)
		assert.Equal(t, *account.data.Status, accountReadyStatus)
	})

}
