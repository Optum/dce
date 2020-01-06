package account

import (
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		expErr  error
		account model.Account
	}{
		{
			name: "should validate",
			account: model.Account{
				ID:           ptrString("123456789012"),
				Status:       model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
		},
		{
			name: "should not validate no admin role",
			account: model.Account{
				ID:     ptrString("123456789012"),
				Status: model.AccountStatusLeased.AccountStatusPtr(),
			},
			expErr: errors.NewValidation("account", fmt.Errorf("adminRoleArn: is required.")), //nolint golint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := New(nil, tt.account)

			err := account.Validate()
			assert.True(t, errors.Is(err, tt.expErr))

		})
	}
}
