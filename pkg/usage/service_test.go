package usage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/pkg/usage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrFloat64(f float64) *float64 {
	ptrF := f
	return &ptrF
}

func TestUpsertLeaseUsage(t *testing.T) {

	now := time.Now()

	tests := []struct {
		name string
		req  usage.Lease
		ret  error
		exp  error
	}{
		{
			name: "should upsert usage information",
			req: usage.Lease{
				Date:         &now,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(1.0),
				CostCurrency: ptrString("USD"),
			},
			ret: nil,
			exp: nil,
		},
		{
			name: "should get failure",
			req: usage.Lease{
				Date:         &now,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(1.0),
				CostCurrency: ptrString("USD"),
			},
			ret: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			exp: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
		},
		{
			name: "should fail validation",
			req: usage.Lease{
				Date:        &now,
				PrincipalID: ptrString("test"),
				LeaseID:     ptrString("id-1"),
				CostAmount:  ptrFloat64(1.0),
			},
			exp: errors.NewValidation("usage", fmt.Errorf("costCurrency: must be a valid cost currency.")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.LeaseReaderWriter{}

			mocksRwd.On("Write", mock.AnythingOfType("*usage.Lease")).Return(tt.ret)

			usageSvc := usage.NewService(usage.NewServiceInput{
				DataSvc: mocksRwd,
			})

			err := usageSvc.UpsertLeaseUsage(&tt.req)
			assert.True(t, errors.Is(err, tt.exp), "actual error %q doesn't match expected error %q", err, tt.exp)
		})
	}
}
