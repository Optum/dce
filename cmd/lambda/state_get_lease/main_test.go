package main

import (
	"context"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptr64(i int64) *int64 {
	ptrI := i
	return &ptrI
}
func TestGetLease(t *testing.T) {

	now := time.Now()
	nextWeek := time.Now().AddDate(0, 0, 7)
	tests := []struct {
		name     string
		input    lease.Lease
		getLease *lease.Lease
		getErr   error
		expErr   error
		expOut   eventOut
	}{
		{
			name: "when valid lease provided get current lease with TTL",
			input: lease.Lease{
				ID: ptrString("id-1"),
			},
			getLease: &lease.Lease{
				PrincipalID:      ptrString("test"),
				ID:               ptrString("id-1"),
				Status:           lease.StatusActive.StatusPtr(),
				StatusModifiedOn: ptr64(now.Unix()),
				ExpiresOn:        ptr64(nextWeek.Unix()),
				AccountID:        ptrString("123456789012"),
			},
			expOut: eventOut{
				&lease.Lease{
					AccountID:        ptrString("123456789012"),
					PrincipalID:      ptrString("test"),
					ID:               ptrString("id-1"),
					Status:           lease.StatusActive.StatusPtr(),
					StatusModifiedOn: ptr64(now.Unix()),
					ExpiresOn:        ptr64(nextWeek.Unix()),
				},
				601200,
				709200,
			},
		},
		{
			name: "when inactive lease provided get current lease with TTL",
			input: lease.Lease{
				ID: ptrString("id-1"),
			},
			getLease: &lease.Lease{
				PrincipalID:      ptrString("test"),
				ID:               ptrString("id-1"),
				Status:           lease.StatusInactive.StatusPtr(),
				StatusModifiedOn: ptr64(now.Unix()),
				ExpiresOn:        ptr64(nextWeek.Unix()),
				AccountID:        ptrString("123456789012"),
			},
			expOut: eventOut{
				&lease.Lease{
					AccountID:        ptrString("123456789012"),
					PrincipalID:      ptrString("test"),
					ID:               ptrString("id-1"),
					Status:           lease.StatusInactive.StatusPtr(),
					StatusModifiedOn: ptr64(now.Unix()),
					ExpiresOn:        ptr64(nextWeek.Unix()),
				},
				0,
				108000,
			},
		},
	}

	// Iterate through each test in the list
	for _, tt := range tests {
		cfgBldr := &config.ConfigurationBuilder{}
		svcBldr := &config.ServiceBuilder{Config: cfgBldr}
		// Setup mocks

		leaseSvcMock := mocks.Servicer{}
		leaseSvcMock.On("Get", *tt.input.ID).Return(tt.getLease, tt.getErr)

		svcBldr.Config.WithService(&leaseSvcMock)
		_, err := svcBldr.Build()
		assert.Nil(t, err)
		if err == nil {
			services = svcBldr
		}

		out, err := handler(context.TODO(), tt.input)
		assert.True(t, errors.Is(err, tt.expErr))
		assert.Equal(t, tt.expOut, out)
	}
}
