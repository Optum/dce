package account

import (
	"fmt"
	"testing"

	dataMocks "github.com/Optum/dce/pkg/account/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAccounts(t *testing.T) {

	tests := []struct {
		name       string
		inputData  accountData
		returnData accountsData
		returnErr  error
		expResult  *Accounts
		expErr     error
	}{
		{
			name: "standard",
			inputData: accountData{
				Status: AccountStatusReady.StatusPtr(),
			},
			returnData: accountsData{
				accountData{
					ID:     aws.String("1"),
					Status: AccountStatusReady.StatusPtr(),
				},
				accountData{
					ID:     aws.String("2"),
					Status: AccountStatusReady.StatusPtr(),
				},
			},
			returnErr: nil,
			expErr:    nil,
			expResult: &Accounts{
				data: accountsData{
					accountData{
						ID:     ptrString("1"),
						Status: AccountStatusReady.StatusPtr(),
					},
					accountData{
						ID:     ptrString("2"),
						Status: AccountStatusReady.StatusPtr(),
					},
				},
			},
		},
		{
			name: "internal error",
			inputData: accountData{
				Status: AccountStatusReady.StatusPtr(),
			},
			returnData: nil,
			returnErr:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			expErr:     errors.NewInternalServer("failure", fmt.Errorf("original error")),
			expResult:  nil,
		},
		{
			name: "validation error",
			inputData: accountData{
				ID: ptrString("123456789012"),
			},
			returnData: nil,
			returnErr:  nil,
			expErr:     errors.NewValidation("account", fmt.Errorf("id: must be empty.")), //nolint golint
			expResult:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksReader := &dataMocks.MultipleReader{}
			mocksReader.On("GetAccounts", mock.MatchedBy(func(account *Account) bool {
				if account.data.Status != nil {
					bool := account.data.Status.String() == tt.inputData.Status.String()
					return bool
				}
				return false
			}), mock.MatchedBy(func(accounts *Accounts) bool {
				accounts.data = tt.returnData
				return true
			})).Return(tt.expErr)

			query := Account{
				data: tt.inputData,
			}
			accounts, err := GetAccounts(&query, mocksReader)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.Equal(t, tt.expResult, accounts)
		})
	}

}
