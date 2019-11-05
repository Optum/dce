package main

import (
	"log"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
)

type leaseChangeEvent struct {
	snsSvc             common.Notificationer
	leaseEventTopicArn string
	lease              *db.Lease
}

//
func publishLeaseChangedEvent(input *leaseChangeEvent) error {
	// Prepare the SNS message body
	leaseEventMsg, err := common.PrepareSNSMessageJSON(input.lease)
	if err != nil {
		log.Printf("Failed to prepare SNS message for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		return err
	}

	_, err = input.snsSvc.PublishMessage(&input.leaseEventTopicArn, &leaseEventMsg, true)
	if err != nil {
		log.Printf("Failed to publish SNS message for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		return err
	}
	return nil
}
