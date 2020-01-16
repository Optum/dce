package account

import (
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		expErr  error
		account accountData
	}{
		{
			name: "should validate",
			account: accountData{
				ID:           ptrString("123456789012"),
				Status:       AccountStatusReady.StatusPtr(),
				AdminRoleArn: ptrString("test:arn"),
			},
		},
		{
			name: "should not validate no admin role",
			account: accountData{
				ID:     ptrString("123456789012"),
				Status: AccountStatusLeased.StatusPtr(),
			},
			expErr: errors.NewValidation("account", fmt.Errorf("adminRoleArn: must be a string.")), //nolint golint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := New(nil, tt.account)

			err := account.Validate()
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
