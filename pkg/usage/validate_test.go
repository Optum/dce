package usage_test

import (
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		expErr error
		usage  usage.NewUsageInput
	}{
		{
			name: "should validate",
			usage: usage.NewUsageInput{
				StartDate:   1580924093,
				AccountID:   "123456789012",
				PrincipalID: "user1",
			},
		},
		{
			name: "should not validate no status",
			usage: usage.NewUsageInput{
				StartDate:   1580924093,
				AccountID:   "ABCADF",
				PrincipalID: "user1",
			},
			expErr: errors.NewValidation("usage", fmt.Errorf("accountId: must be a string with 12 digits.")), //nolint golint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := usage.NewUsage(tt.usage)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
