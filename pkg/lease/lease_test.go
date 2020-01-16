package lease

import (
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/errors"
	dataMocks "github.com/Optum/dce/pkg/lease/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrInt64(i int64) *int64 {
	ptrI := i
	return &ptrI
}

func TestProperties(t *testing.T) {
	tests := []struct {
		name  string
		lease model.Lease
	}{
		{
			name: "standard",
			lease: model.Lease{
				ID:          ptrString("234567890121"),
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
				AccountID:   ptrString("123456789012"),
				PrincipalID: ptrString("User1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := New(nil, tt.lease)
			assert.Equal(t, tt.lease.ID, lease.ID())
			assert.Equal(t, tt.lease.AccountID, lease.AccountID())
			assert.Equal(t, tt.lease.PrincipalID, lease.PrincipalID())
			assert.Equal(t, tt.lease.LeaseStatus, lease.LeaseStatus())
		})
	}
}

func TestGetLeaseByID(t *testing.T) {

	tests := []struct {
		name       string
		ID         string
		returnData *model.Lease
		returnErr  error
		expReturn  *Lease
		expErr     error
	}{
		{
			name: "should get a lease by ID",
			ID:   "123456789012",
			returnData: &model.Lease{
				ID:          ptrString("123456789012"),
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
			},
			returnErr: nil,
			expReturn: &Lease{
				writer: nil,
				data: model.Lease{
					ID:          ptrString("123456789012"),
					LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
				},
			},
			expErr: nil,
		},
		{
			name:       "should get failure",
			returnData: nil,
			returnErr:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expReturn:  nil,
			expErr:     errors.NewInternalServer("failure", fmt.Errorf("original failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksReader := &dataMocks.Reader{}

			mocksReader.On("GetLeaseByID", tt.ID).
				Return(tt.returnData, tt.returnErr)

			lease, err := GetLeaseByID(tt.ID, mocksReader, nil)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.Equal(t, tt.expReturn, lease)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		expErr    error
		returnErr error
		lease     model.Lease
	}{
		{
			name: "should delete a lease",
			lease: model.Lease{
				ID:          ptrString("123456789012"),
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
			},
			returnErr: nil,
		},
		{
			name: "should error when lease is active",
			lease: model.Lease{
				ID:          ptrString("123456789012"),
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
			},
			returnErr: nil,
			expErr:    errors.NewConflict("lease", "123456789012", fmt.Errorf("leaseStatus: must not be active.")), //nolint golint
		},
		{
			name: "should error when delete fails",
			lease: model.Lease{
				ID:          ptrString("123456789012"),
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksDeleter := &dataMocks.WriterDeleter{}
			mocksDeleter.On("DeleteLease", mock.Anything).
				Return(tt.returnErr)
			lease := New(mocksDeleter, tt.lease)

			err := lease.Delete()
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)

		})
	}
}

func TestMarshallJSON(t *testing.T) {

	t.Run("should marshall into JSON", func(t *testing.T) {
		leaseID := "123456789012"
		leaseStatus := model.LeaseStatus("Active")

		lease := Lease{
			data: model.Lease{
				ID:          &leaseID,
				LeaseStatus: &leaseStatus,
			},
		}
		b, err := lease.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t,
			"{\"id\":\"123456789012\",\"accountStatus\":\"Ready\"}",
			string(b))
	})

}

func TestUpdate(t *testing.T) {

	tests := []struct {
		name        string
		expErr      error
		returnErr   error
		amReturnErr error
		origLease   model.Lease
		updLease    model.Lease
		expLease    model.Lease
	}{
		{
			name: "should update",
			origLease: model.Lease{
				ID:           ptrString("123456789012"),
				AccountID:    ptrString("234567890121"),
				PrincipalID:  ptrString("User1"),
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
				CreatedOn: ptrInt64(1573592058),
			},
			updLease: model.Lease{
				CreatedOn: ptrInt64(1593592058),
			},
			expLease: model.Lease{
				ID:           ptrString("123456789012"),
				AccountID:    ptrString("234567890121"),
				PrincipalID:  ptrString("User1"),
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
				CreatedOn: ptrInt64(1593592058),
			},
			returnErr: nil,
		},
		{
			name: "should fail validation on update",
			origLease: model.Lease{
				ID:          ptrString("123456789012"),
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
			},
			updLease: model.Lease{
				ID: ptrString("abc125"),
			},
			expLease: model.Lease{
				ID:           ptrString("123456789012"),
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
			},
			returnErr: nil,
			expErr:    errors.NewValidation("lease", fmt.Errorf("id: must be empty.")), //nolint golint
		},
		{
			name: "should fail on save",
			origLease: model.Lease{
				ID:           ptrString("123456789012"),
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			updLease: model.Lease{
				LastModifiedOn: ptrInt64(1593592058),
			},
			expLease: model.Lease{
				ID:           ptrString("123456789012"),
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
				LastModifiedOn: ptrInt64(1593592058),
			},
			returnErr: errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			expErr:    errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksWriter := &dataMocks.WriterDeleter{}
			mocksManager := &dataMocks.Manager{}

			mocksWriter.On("WriteAccount", mock.MatchedBy(func(input *model.Lease) bool {
				return (*input.ID == *tt.expLease.ID)
			}), mock.AnythingOfType("*int64")).Return(tt.returnErr)

			mocksManager.On("Setup", mock.AnythingOfType("string")).Return(tt.amReturnErr)

			account := New(mocksWriter, tt.origLease)

			err := account.Update(tt.updLease, mocksManager)

			assert.Truef(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.NotEqual(t, tt.origLease.LastModifiedOn, account)
			if tt.returnErr == nil {
				assert.Equal(t, tt.expLease.Metadata, account.data.Metadata)
			}

		})
	}
}
