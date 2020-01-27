package account_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name    string
		expErr  error
		account account.Account
	}{
		{
			name: "should validate",
			account: account.Account{
				ID:             ptrString("123456789012"),
				Status:         account.StatusReady.StatusPtr(),
				AdminRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
		},
		{
			name: "should not validate no admin role",
			account: account.Account{
				ID:             ptrString("123456789012"),
				Status:         account.StatusLeased.StatusPtr(),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			expErr: errors.NewValidation("account", fmt.Errorf("adminRoleArn: must be a string.")), //nolint golint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := tt.account.Validate()
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
