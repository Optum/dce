package main

import (
	"errors"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestResetPipeline(t *testing.T) {
	t.Run("updateDBPostReset", func(t *testing.T) {

		t.Run("Should change any ResetLock assignments to Active", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Assignments
			dbSvc.
				On("FindAssignmentsByAccount", "111").
				Return([]*db.RedboxAccountAssignment{
					{
						AccountID:        "111",
						UserID:           "222",
						AssignmentStatus: db.ResetLock,
					},
					// Should not change the status of decommissioned assignments
					{
						AccountID:        "111",
						UserID:           "222",
						AssignmentStatus: db.Decommissioned,
					},
				}, nil)

			// Mock Assignment status change
			dbSvc.
				On("TransitionAssignmentStatus", "111", "222", db.ResetLock, db.Active).
				Return(&db.RedboxAccountAssignment{}, nil)

			// Mock Account status change
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.RedboxAccount{}, nil)

			err := updateDBPostReset(dbSvc, "111")
			require.Nil(t, err)
		})

		t.Run("Should change account status from NotReady to Ready", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Assignments
			dbSvc.
				On("FindAssignmentsByAccount", "111").
				Return([]*db.RedboxAccountAssignment{}, nil)

			// Should change the Account Status
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.RedboxAccount{}, nil)

			err := updateDBPostReset(dbSvc, "111")
			require.Nil(t, err)
		})

		t.Run("Should not change account status of Assigned accounts", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Assignments
			dbSvc.
				On("FindAssignmentsByAccount", "111").
				Return([]*db.RedboxAccountAssignment{}, nil)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, &db.StatusTransitionError{})

			err := updateDBPostReset(dbSvc, "111")
			require.Nil(t, err)
		})

		t.Run("Should handle DB errors (FindAssignmentsByAccount)", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Assignments
			dbSvc.
				On("FindAssignmentsByAccount", "111").
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, "111")
			require.Equal(t, errors.New("test error"), err)
		})

		t.Run("Should handle DB errors (TransitionAccountStatus)", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)
			// Mock Assignments
			dbSvc.
				On("FindAssignmentsByAccount", "111").
				Return([]*db.RedboxAccountAssignment{}, nil)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, "111")
			require.Equal(t, errors.New("test error"), err)
		})
	})
}
