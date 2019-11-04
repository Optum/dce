package main

import (
	"fmt"
	errors2 "github.com/Optum/Redbox/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/pkg/errors"
	"log"
)

/*
As part of the v0.21.0 release, we are renaming all our DynamoDB tables
to remove the "Redbox" prefix, and to standardize naming conventions.

	RedboxAccountProd 	--> Accounts
	RedboxLeaseProd 		--> Leases
	UsageCache					--> Usage

DynamoDB does not support in-place table renaming, so we will
need to migrate data from each table to the newly renamed table.

This script does a simple data dump and import for each table.
*/

func main() {
	// Configure DynamoDB client
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion("us-east-1"),
	)

	err := migrate(&migrateInput{
		db: dynDB,
		tables: map[string]string{
			"RedboxAccountProd": "Accounts",
			"RedboxLeaseProd":   "Leases",
			"UsageCache":        "Usage",
		},
	})
	if err != nil {
		log.Fatalf("Migration failed: %s", err)
	}
}

type migrateInput struct {
	db     *dynamodb.DynamoDB
	tables map[string]string
}

func migrate(input *migrateInput) error {
	// Iterate through all tables
	for srcTableName, dstTableName := range input.tables {
		// Dump table records
		scanRes, err := input.db.Scan(&dynamodb.ScanInput{
			TableName: &srcTableName,
		})
		if err != nil {
			return errors.Wrapf(err, "Scan failed for %s", srcTableName)
		}

		// Create records in the new table
		var deferredErrors []error
		for i, item := range scanRes.Items {
			_, err = input.db.PutItem(&dynamodb.PutItemInput{
				TableName: &dstTableName,
				Item:      item,
			})
			if err != nil {
				deferredErrors = append(deferredErrors, err)
				log.Printf(`
Failed to put item %d/%d to %s
Error: %s
Item: %v
`, i+1, len(scanRes.Items), dstTableName, err, item)
				continue
			}
			log.Printf("Migrated record %d/%d from %s to %s", i+1, len(scanRes.Items), srcTableName, dstTableName)
		}

		// Handle deferred errors
		if len(deferredErrors) > 0 {
			return errors2.NewMultiError(
				fmt.Sprintf(
					"%d/%d migrations to %s failed",
					len(deferredErrors), len(scanRes.Items), dstTableName,
				),
				deferredErrors,
			)
		}
	}

	return nil
}
