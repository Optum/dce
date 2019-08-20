package main

import (
	"errors"
	"testing"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/stretchr/testify/require"
)

func TestResetPipeline(t *testing.T) {
	t.Run("updateDBPostReset", func(t *testing.T) {

		t.Run("Should change any ResetLock leases to Active", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.RedboxLease{
					{
						AccountID:   "111",
						PrincipalID: "222",
						LeaseStatus: db.ResetLock,
					},
					// Should not change the status of decommissioned leases
					{
						AccountID:   "111",
						PrincipalID: "222",
						LeaseStatus: db.Decommissioned,
					},
				}, nil)

			// Mock Lease status change
			dbSvc.
				On("TransitionLeaseStatus", "111", "222", db.ResetLock, db.Active).
				Return(&db.RedboxLease{}, nil)

			// Mock Account status change
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.RedboxAccount{}, nil)

			err := updateDBPostReset(dbSvc, "111")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should change any ResetFinanceLock leases to FinanceLock", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.RedboxLease{
					// Should call TransitionLeaseStatus for this ResetFinanceLock lease status
					{
						AccountID:   "111",
						PrincipalID: "222",
						LeaseStatus: db.ResetFinanceLock,
					},
					// Should not call TransitionLeaseStatus for this decommisioned lease status
					{
						AccountID:   "111",
						PrincipalID: "333",
						LeaseStatus: db.Decommissioned,
					},
				}, nil)

			// Mock Lease status change
			dbSvc.
				On("TransitionLeaseStatus", "111", "222", db.ResetFinanceLock, db.FinanceLock).
				Return(&db.RedboxLease{}, nil)

			// Mock Account status change
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.RedboxAccount{}, nil)

			err := updateDBPostReset(dbSvc, "111")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 1)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should change account status from NotReady to Ready", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.RedboxLease{}, nil)

			// Should change the Account Status
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.RedboxAccount{}, nil)

			err := updateDBPostReset(dbSvc, "111")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should not change account status of Leased accounts", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.RedboxLease{}, nil)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, &db.StatusTransitionError{})

			err := updateDBPostReset(dbSvc, "111")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should handle DB errors (FindLeasesByAccount)", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, "111")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 0)
			require.Equal(t, errors.New("test error"), err)
		})

		t.Run("Should handle DB errors (TransitionAccountStatus)", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Leases
			dbSvc.
				On("FindLeasesByAccount", "111").
				Return([]*db.RedboxLease{}, nil)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, "111")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Equal(t, errors.New("test error"), err)
		})
	})
}
