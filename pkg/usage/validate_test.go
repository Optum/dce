package usage_test

import (
	"fmt"
	"testing"
	"time"

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
				Date:        time.Now(),
				PrincipalID: "user1",
			},
		},
		{
			name: "should not validate cost currency",
			usage: usage.NewUsageInput{
				Date:         time.Now(),
				CostCurrency: "BAD",
			},
			expErr: errors.NewValidation("usage", fmt.Errorf("costCurrency: must be a valid value.")), //nolint golint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := usage.NewUsage(tt.usage)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
