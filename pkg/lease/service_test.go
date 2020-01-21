package lease_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func TestGetLeaseByID(t *testing.T) {

	type response struct {
		data *lease.Lease
		err  error
	}

	tests := []struct {
		name string
		ID   string
		ret  response
		exp  response
	}{
		{
			name: "should get an lease by ID",
			ID:   "70c2d96d-7938-4ec9-917d-476f2b09cc04",
			ret: response{
				data: &lease.Lease{
					ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status: lease.StatusActive.StatusPtr(),
				},
				err: nil,
			},
			exp: response{
				data: &lease.Lease{
					ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status: lease.StatusActive.StatusPtr(),
				},
				err: nil,
			},
		},
		{
			name: "should get failure",
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}

			mocksRwd.On("Get", tt.ID).Return(tt.ret.data, tt.ret.err)

			leaseSvc := lease.NewService(lease.NewServiceInput{
				DataSvc: mocksRwd,
			})

			getLease, err := leaseSvc.Get(tt.ID)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, getLease)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		expErr    error
		returnErr error
		lease     lease.Lease
	}{
		{
			name: "should delete a lease",
			lease: lease.Lease{
				ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status: lease.StatusActive.StatusPtr(),
			},
			returnErr: nil,
		},
		{
			name: "should error when delete fails",
			lease: lease.Lease{
				ID:     ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status: lease.StatusActive.StatusPtr(),
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}
			mocksRwd.On("Delete", mock.Anything).
				Return(tt.returnErr)

			leaseSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc: mocksRwd,
				},
			)
			err := leaseSvc.Delete(&tt.lease)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}

func TestSave(t *testing.T) {
	now := time.Now().Unix()

	type response struct {
		data *lease.Lease
		err  error
	}

	tests := []struct {
		name      string
		returnErr error
		lease     *lease.Lease
		exp       response
	}{
		{
			name: "should save lease with timestamps",
			lease: &lease.Lease{
				ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:         lease.StatusActive.StatusPtr(),
				PrincipalID:    ptrString("test:arn"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			exp: response{
				data: &lease.Lease{
					ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:         lease.StatusActive.StatusPtr(),
					PrincipalID:    ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should save with new created on",
			lease: &lease.Lease{
				ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:      lease.StatusActive.StatusPtr(),
				PrincipalID: ptrString("test:arn"),
			},
			exp: response{
				data: &lease.Lease{
					ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:         lease.StatusActive.StatusPtr(),
					PrincipalID:    ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: nil,
			},
			returnErr: nil,
		},
		{
			name: "should fail on return err",
			lease: &lease.Lease{
				ID:          ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
				Status:      lease.StatusActive.StatusPtr(),
				PrincipalID: ptrString("test:arn"),
			},
			exp: response{
				data: &lease.Lease{
					ID:             ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
					Status:         lease.StatusActive.StatusPtr(),
					PrincipalID:    ptrString("test:arn"),
					LastModifiedOn: &now,
					CreatedOn:      &now,
				},
				err: errors.NewInternalServer("failure", nil),
			},
			returnErr: errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriterDeleter{}

			mocksRwd.On("Write", mock.AnythingOfType("*lease.Lease"), mock.AnythingOfType("*int64")).Return(tt.returnErr)

			leaseSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc: mocksRwd,
				},
			)

			err := leaseSvc.Save(tt.lease)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, tt.lease)

		})
	}
}

func TestGetLeases(t *testing.T) {

	type response struct {
		data *lease.Leases
		err  error
	}

	tests := []struct {
		name      string
		inputData lease.Lease
		ret       response
		exp       response
	}{
		{
			name: "standard",
			inputData: lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			ret: response{
				data: &lease.Leases{
					lease.Lease{
						ID:     aws.String("1"),
						Status: lease.StatusActive.StatusPtr(),
					},
					lease.Lease{
						ID:     aws.String("2"),
						Status: lease.StatusActive.StatusPtr(),
					},
				},
				err: nil,
			},
			exp: response{
				data: &lease.Leases{
					lease.Lease{
						ID:     ptrString("1"),
						Status: lease.StatusActive.StatusPtr(),
					},
					lease.Lease{
						ID:     ptrString("2"),
						Status: lease.StatusActive.StatusPtr(),
					},
				},
				err: nil,
			},
		},
		{
			name: "internal error",
			inputData: lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			},
		},
		{
			name: "validation error",
			inputData: lease.Lease{
				ID: ptrString("70c2d96d-7938-4ec9-917d-476f2b09cc04"),
			},
			ret: response{
				data: nil,
				err:  nil,
			},
			exp: response{
				data: nil,
				err:  errors.NewValidation("lease", fmt.Errorf("id: must be empty.")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRWD := &mocks.ReaderWriterDeleter{}
			mocksRWD.On("List", mock.AnythingOfType("*lease.Lease")).Return(tt.ret.data, tt.ret.err)

			leasesSvc := lease.NewService(
				lease.NewServiceInput{
					DataSvc: mocksRWD,
				},
			)

			leases, err := leasesSvc.List(&tt.inputData)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, leases)
		})
	}

}
