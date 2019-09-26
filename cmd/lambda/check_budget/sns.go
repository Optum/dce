package main

import (
	"github.com/Optum/Dce/pkg/common"
	"github.com/Optum/Dce/pkg/db"
	"log"
)

type publishLeaseLockedInput struct {
	snsSvc              common.Notificationer
	leaseLockedTopicArn string
	lease               *db.DceLease
}

func publishLeaseLocked(input *publishLeaseLockedInput) error {
	// Prepare the SNS message body
	leaseLockedMsg, err := common.PrepareSNSMessageJSON(input.lease)
	if err != nil {
		log.Printf("Failed to prepare SNS message for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		return err
	}

	_, err = input.snsSvc.PublishMessage(&input.leaseLockedTopicArn, &leaseLockedMsg, true)
	if err != nil {
		log.Printf("Failed to publish SNS message for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		return err
	}
	return nil
}
