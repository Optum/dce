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
	type response struct {
		data *usage.Lease
		err  error
	}

	tests := []struct {
		name string
		req  usage.Lease
		ret  response
		exp  response
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
			ret: response{
				data: &usage.Lease{
					PrincipalID: ptrString("test"),
					LeaseID:     ptrString("id-1"),
				},
				err: nil,
			},
			exp: response{
				data: &usage.Lease{
					PrincipalID: ptrString("test"),
					LeaseID:     ptrString("id-1"),
				},
				err: nil,
			},
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
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
		},
		{
			name: "should fail validation",
			req: usage.Lease{
				Date:        &now,
				PrincipalID: ptrString("test"),
				LeaseID:     ptrString("id-1"),
				CostAmount:  ptrFloat64(1.0),
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("usage", fmt.Errorf("costCurrency: must be a valid cost currency.")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.LeaseReaderWriter{}

			mocksRwd.On("Write", mock.AnythingOfType("*usage.Lease")).Return(tt.ret.data, tt.ret.err)

			usageSvc := usage.NewService(usage.NewServiceInput{
				DataSvc: mocksRwd,
			})

			usg, err := usageSvc.UpsertLeaseUsage(&tt.req)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, usg)
		})
	}
}
