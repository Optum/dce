package account

import (
	gErrors "errors"
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
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

func TestGetAccountsByPrincipalId(t *testing.T) {

	t.Run("should return a list of accounts queried on Principal ID", func(t *testing.T) {
		principalID := "arn:aws:test"

		mocksReader := &dataMocks.MultipleReader{}

		mocksReader.On("GetAccountsByPrincipalID", principalID).
			Return(
				&model.Accounts{
					model.Account{
						ID:               aws.String("1"),
						PrincipalRoleArn: &principalID,
					},
					model.Account{
						ID:               aws.String("1"),
						PrincipalRoleArn: &principalID,
					},
				}, nil,
			)

		accounts, err := GetAccountsByPrincipalID(principalID, mocksReader)
		assert.NoError(t, err)
		assert.Len(t, *accounts, 2)
		assert.Equal(t, *(*accounts)[0].data.ID, "1")
		assert.Equal(t, *(*accounts)[0].data.PrincipalRoleArn, principalID)
	})

}

func TestUpdateAccount(t *testing.T) {

	t.Run("should fail when AdminRoleArn is provided", func(t *testing.T) {
		mocksWriter := &dataMocks.Writer{}
		mocksManager := &dataMocks.Manager{}
		accountStatus := model.Ready
		accountID := "123456789012"
		mocksWriter.On("Update").
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

		account := Account{
			data: model.Account{
				ID:     &accountID,
				Status: &accountStatus,
			},
		}
		data := model.Account{
			AdminRoleArn: aws.String("roleArn"),
		}
		err := account.Update(data, mocksWriter, mocksManager)
		assert.True(t, gErrors.Is(err, &errors.ErrValidation{}))
		assert.Equal(t, *account.data.ID, accountID)
		assert.Equal(t, *account.data.Status, accountStatus)
	})

}
