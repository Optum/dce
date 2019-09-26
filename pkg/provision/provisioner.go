package provision

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Optum/Dce/pkg/db"
)

// Provisioner interface for providing helper methods for provisioning a
// principal to a Dce Account
type Provisioner interface {
	FindActiveLeaseForPrincipal(string) (*db.DceLease, error)
	FindLeaseWithAccount(string, string) (*db.DceLease,
		error)
	ActivateAccount(bool, string, string, float64, string, []string) (*db.DceLease,
		error)
	RollbackProvisionAccount(bool, string, string) error
}

// AccountProvision implements Provisioner for official Dce Provisioning
type AccountProvision struct {
	DBSvc db.DBer
}

// FindActiveLeaseForPrincipal is a helper function to find if there's any actively
// leased (Active/FinanceLock/ResetLock) account attached to a principal
func (prov *AccountProvision) FindActiveLeaseForPrincipal(principalID string) (
	*db.DceLease, error) {
	// Check if the principal has any existing Active/FinanceLock/ResetLock
	// Leases
	activeLease := &db.DceLease{}
	leases, err := prov.DBSvc.FindLeasesByPrincipal(principalID)
	if err != nil {
		return nil, err
	}
	for _, lease := range leases {
		if lease.LeaseStatus != db.Decommissioned {
			activeLease = lease
			break
		}
	}
	return activeLease, nil
}

// FindLeaseWithAccount is a helper function to find if there's any
// lease with the provided account. Returns an error if there's
// another active lease that is not the principal
func (prov *AccountProvision) FindLeaseWithAccount(principalID string,
	accountID string) (*db.DceLease, error) {
	// Check if the principal and Account has been leased before and verify the
	// Account has no existing Active/FinanceLock/ResetLock Leases
	leases, err := prov.DBSvc.FindLeasesByAccount(accountID)
	if err != nil {
		return nil, err
	}

	matchingLease := &db.DceLease{}
	for _, l := range leases {
		// Check if the status is Active
		// If so, return an error
		if l.LeaseStatus != db.Decommissioned {
			errStr := fmt.Sprintf("Attempt to lease Active Account as new "+
				"Dce - %s", accountID)
			return nil, errors.New(errStr)
		}

		// Check if the there exists principal + Account lease
		if l.PrincipalID == principalID {
			matchingLease = l
		}
	}
	return matchingLease, nil
}

// ActivateAccount is a helper function to either create or update
// an existing Account Lease from a Decommissioned to an Active state.
// Returns the lease that has been activated - does not return any previous
// leases
func (prov *AccountProvision) ActivateAccount(create bool,
	principalID string, accountID string, budgetAmount float64, budgetCurrency string, budgetNotificationEmails []string) (*db.DceLease, error) {
	// Create a new Dce Account Lease if there doesn't exist one already
	// else, update the existing lease to active
	var assgn *db.DceLease
	var err error
	if create {
		log.Printf("Create new Lease for Principal %s and Account %s\n",
			principalID, accountID)
		timeNow := time.Now().Unix()
		lease := &db.DceLease{
			AccountID:                accountID,
			PrincipalID:              principalID,
			LeaseStatus:              db.Active,
			BudgetAmount:             budgetAmount,
			BudgetCurrency:           budgetCurrency,
			BudgetNotificationEmails: budgetNotificationEmails,
			CreatedOn:                timeNow,
			LastModifiedOn:           timeNow,
			LeaseStatusModifiedOn:    timeNow,
		}
		_, err = prov.DBSvc.PutLease(*lease) // new leases return an empty lease
		// Failed to Create Lease
		if err != nil {
			return nil, err
		}
		assgn = lease
	} else {
		log.Printf("Update existing Lease for Principal %s and Account %s\n",
			principalID, accountID)
		assgn, err = prov.DBSvc.TransitionLeaseStatus(accountID, principalID,
			db.Decommissioned, db.Active)
		// Failed to Update Lease
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
	transitionAccountStatus bool, principalID string, accountID string) error {
	// Reverse Account Lease- Set next state as Decommissioned
	_, errLease := prov.DBSvc.TransitionLeaseStatus(accountID, principalID,
		db.Active, db.Decommissioned)

	// Reverse Account - Set next state as Ready
	if transitionAccountStatus {
		_, errAccount := prov.DBSvc.TransitionAccountStatus(accountID,
			db.Leased, db.Ready)
		if errAccount != nil {
			return errAccount
		}
	}
	return errLease
}
