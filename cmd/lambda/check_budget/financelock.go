package main

import (
	"log"

	"github.com/Optum/Dce/pkg/db"
)

type financeLockInput struct {
	lease *db.DceLease
	dbSvc db.DBer
}

// Update the Lease record in the DB, to be FinanceLocked.
// Status changes:
//		Active --> FinanceLock
//		ResetLock --> ResetFinanceLock
//
// other statuses (Decommissioned, FinanceLock) will not be modified
func financeLock(input *financeLockInput) error {
	// Figure out what status we want for our FinanceLock'd account
	var nextStatus db.LeaseStatus
	switch input.lease.LeaseStatus {
	// Active --> FinanceLock
	case db.Active:
		nextStatus = db.FinanceLock
	// FinanceLock --> ResetFinanceLock
	case db.ResetLock:
		nextStatus = db.ResetFinanceLock
	}

	// If lease is Decommissioned or already FinanceLocked, don't change it
	if nextStatus == "" {
		log.Printf("Lease for %s @ %s is %s. Not changing status",
			input.lease.PrincipalID, input.lease.AccountID, input.lease.LeaseStatus)
	} else {
		log.Printf("Changing status of lease %s @ %s from %s to %s",
			input.lease.PrincipalID, input.lease.AccountID, input.lease.LeaseStatus, nextStatus)

		lease, err := input.dbSvc.TransitionLeaseStatus(
			input.lease.AccountID, input.lease.PrincipalID,
			input.lease.LeaseStatus, db.FinanceLock,
		)
		if err != nil {
			log.Printf("Failed to change status of lease %s @ %s from %s to %s",
				input.lease.PrincipalID, input.lease.AccountID, input.lease.LeaseStatus, nextStatus)
			return err
		}
		input.lease = lease
	}

	return nil
}
