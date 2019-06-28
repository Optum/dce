package provision

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Optum/Redbox/pkg/db"
)

// Provisioner interface for providing helpfer methods for provisioning a
// User to a Redbox Account
type Provisioner interface {
	FindUserActiveAssignment(string) (*db.RedboxAccountAssignment, error)
	FindUserAssignmentWithAccount(string, string) (*db.RedboxAccountAssignment,
		error)
	ActivateAccountAssignment(bool, string, string) (*db.RedboxAccountAssignment,
		error)
	RollbackProvisionAccount(bool, string, string) error
}

// AccountProvision implements Provisioner for official Redbox Provisioning
type AccountProvision struct {
	DBSvc db.DBer
}

// FindUserActiveAssignment is a helper function to find if there's any actively
// assigned (Active/FinanceLock/ResetLock) account attached to a user
func (prov *AccountProvision) FindUserActiveAssignment(userID string) (
	*db.RedboxAccountAssignment, error) {
	// Check if the users has any existing Active/FinanceLock/ResetLock
	// Assignments
	activeAssignment := &db.RedboxAccountAssignment{}
	userAssignments, err := prov.DBSvc.FindAssignmentByUser(userID)
	if err != nil {
		return nil, err
	}
	for _, assignment := range userAssignments {
		if assignment.AssignmentStatus != db.Decommissioned {
			activeAssignment = assignment
			break
		}
	}
	return activeAssignment, nil
}

// FindUserAssignmentWithAccount is a helper function to find if there's any
// user assignment with the provided account. Returns an error if there's
// another active assignment that is not the user
func (prov *AccountProvision) FindUserAssignmentWithAccount(userID string,
	accountID string) (*db.RedboxAccountAssignment, error) {
	// Check if the User and Account has been assigned before and verify the
	// Account has no existing Active/FinanceLock/ResetLock Assignments
	userAssignment := &db.RedboxAccountAssignment{}
	accountAssignments, err := prov.DBSvc.FindAssignmentsByAccount(accountID)
	if err != nil {
		return nil, err
	}
	for _, assignment := range accountAssignments {
		// Check if the status is Active
		// If so, return an error
		if assignment.AssignmentStatus != db.Decommissioned {
			errStr := fmt.Sprintf("Attempt to Assign Active Account as new "+
				"Redbox - %s", accountID)
			return nil, errors.New(errStr)
		}

		// Check if the there exists User + Account assignment
		if assignment.UserID == userID {
			userAssignment = assignment
		}
	}
	return userAssignment, nil
}

// ActivateAccountAssignment is a helper function to either create or update
// an existing Account Assignment from a Decommissioned to an Active state.
// Returns the assignment that has been activated - does not return any previous
// assignments
func (prov *AccountProvision) ActivateAccountAssignment(create bool,
	userID string, accountID string) (*db.RedboxAccountAssignment, error) {
	// Create a new Redbox Account Assignment if there doesn't exist one already
	// else, update the existing assignment to active
	var assgn *db.RedboxAccountAssignment
	var err error
	if create {
		log.Printf("Create new Assignment for User %s and Account %s\n",
			userID, accountID)
		timeNow := time.Now().Unix()
		userAssignment := &db.RedboxAccountAssignment{
			AccountID:        accountID,
			UserID:           userID,
			AssignmentStatus: db.Active,
			CreatedOn:        timeNow,
			LastModifiedOn:   timeNow,
		}
		_, err = prov.DBSvc.PutAccountAssignment(*userAssignment) // new assignments return an empty assignment
		// Failed to Create Assignment
		if err != nil {
			return nil, err
		}
		assgn = userAssignment
	} else {
		log.Printf("Update existing Assignment for User %s and Account %s\n",
			userID, accountID)
		assgn, err = prov.DBSvc.TransitionAssignmentStatus(accountID, userID,
			db.Decommissioned, db.Active)
		// Failed to Update Assignment
		if err != nil {
			return nil, err
		}
	}
	return assgn, nil
}

// RollbackProvisionAccount will rollback database changes created during the
// provisionAccount function. Tries to rollback everything and returns the last
// error if any.
func (prov *AccountProvision) RollbackProvisionAccount(
	transitionAccountStatus bool, userID string, accountID string) error {
	// Reverse Account Assignment- Set next state as Decommissioned
	_, errAssign := prov.DBSvc.TransitionAssignmentStatus(accountID, userID,
		db.Active, db.Decommissioned)

	// Reverse Account - Set next state as Ready
	if transitionAccountStatus {
		_, errAccount := prov.DBSvc.TransitionAccountStatus(accountID,
			db.Assigned, db.Ready)
		if errAccount != nil {
			return errAccount
		}
	}
	return errAssign
}
