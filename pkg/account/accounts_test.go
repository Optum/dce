package account

import (
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountsByStatus(t *testing.T) {

	t.Run("should return a list of accounts by Status", func(t *testing.T) {
		mocksReader := &dataMocks.MultipleReader{}

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

		accounts, err := GetAccountsByStatus(model.Ready, mocksReader)
		assert.NoError(t, err)
		assert.Len(t, *accounts, 2)
		assert.Equal(t, (*accounts)[0].data.ID, "1")
		assert.Equal(t, (*accounts)[0].data.Status, model.AccountStatus("Ready"))
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
						ID:               "1",
						PrincipalRoleArn: principalID,
					},
					model.Account{
						ID:               "2",
						PrincipalRoleArn: principalID,
					},
				}, nil,
			)

		accounts, err := GetAccountsByPrincipalID(principalID, mocksReader)
		assert.NoError(t, err)
		assert.Len(t, *accounts, 2)
		assert.Equal(t, (*accounts)[0].data.ID, "1")
		assert.Equal(t, (*accounts)[0].data.PrincipalRoleArn, principalID)
	})

}
