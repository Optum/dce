package lease_test

import (
	"fmt"
	"github.com/Optum/dce/pkg/lease"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name   string
		expErr error
		lease  lease.Lease
	}{
		{
			name: "should validate",
			lease: lease.Lease{
				ID:             ptrString("d892f19d-3204-4674-aa78-bd12de5ff226"),
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("user1"),
				Status:         lease.StatusActive.StatusPtr(),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
		},
		{
			name: "should not validate no status",
			lease: lease.Lease{
				ID:             ptrString("d892f19d-3204-4674-aa78-bd12de5ff226"),
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("user1"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			expErr: errors.NewValidation("lease", fmt.Errorf("leaseStatus: must be a valid lease status.")), //nolint golint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := tt.lease.Validate()
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}
