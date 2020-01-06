package account

import (
	"fmt"
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestGetAccounts(t *testing.T) {

	tests := []struct {
		name       string
		inputData  *model.Account
		returnData *model.Accounts
		returnErr  error
		expResult  *Accounts
		expErr     error
	}{
		{
			name: "standard",
			inputData: &model.Account{
				Status: model.AccountStatusReady.AccountStatusPtr(),
			},
			returnData: &model.Accounts{
				model.Account{
					ID:     aws.String("1"),
					Status: model.AccountStatusReady.AccountStatusPtr(),
				},
				model.Account{
					ID:     aws.String("2"),
					Status: model.AccountStatusReady.AccountStatusPtr(),
				},
			},
			returnErr: nil,
			expErr:    nil,
			expResult: &Accounts{
				data: model.Accounts{
					model.Account{
						ID:     ptrString("1"),
						Status: model.AccountStatusReady.AccountStatusPtr(),
					},
					model.Account{
						ID:     ptrString("2"),
						Status: model.AccountStatusReady.AccountStatusPtr(),
					},
				},
			},
		},
		{
			name: "internal error",
			inputData: &model.Account{
				Status: model.AccountStatusReady.AccountStatusPtr(),
			},
			returnData: nil,
			returnErr:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			expErr:     errors.NewInternalServer("failure", fmt.Errorf("original error")),
			expResult:  nil,
		},
		{
			name: "validation error",
			inputData: &model.Account{
				ID: ptrString("123456789012"),
			},
			returnData: nil,
			returnErr:  nil,
			expErr:     errors.NewValidation("account", fmt.Errorf("id: should be nil.")), //nolint golint
			expResult:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksReader := &dataMocks.MultipleReader{}
			mocksReader.On("GetAccounts", tt.inputData).
				Return(tt.returnData, tt.expErr)

			accounts, err := GetAccounts(tt.inputData, mocksReader)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expResult, accounts)
		})
	}

}
