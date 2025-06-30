package main

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// listNotReadyAccountsToCSV retrieves all accounts with the status "NotReady" from the ACCOUNT_TABLE,
// saves them to a CSV file, and uploads the file to the specified S3 bucket.
func listNotReadyAccountsToCSV(dbSvc db.DBer, filePath, bucket, s3Key string) error {
    log.Println("Fetching all accounts with status NotReady from ACCOUNT_TABLE")

    // Query the ACCOUNT_TABLE for accounts with status "NotReady"
    input := db.Account{
        AccountStatus: db.NotReady,
    }
    accounts, err := dbSvc.FindAccountsByStatus(input.AccountStatus)
    if err != nil {
        log.Printf("Failed to fetch accounts: %s", err)
        return err
    }

    log.Printf("Found %d accounts with status NotReady", len(accounts))
  

    // Create or open the CSV file
    file, err := os.Create(filePath)
    if err != nil {
        log.Printf("Failed to create CSV file: %s", err)
        return err
    }
    defer file.Close()

    // Create a CSV writer
    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Write the header row
    err = writer.Write([]string{"AccountID", "Status", "LastUpdated"})
    if err != nil {
        log.Printf("Failed to write header to CSV file: %s", err)
        return err
    }

    // Write account data to the CSV file
    for _, account := range accounts {

        err := writer.Write([]string{
            account.ID,
            string(account.AccountStatus),
        })
        if err != nil {
            log.Printf("Failed to write account data to CSV file: %s", err)
            return err
        }
    }

    log.Printf("Successfully saved NotReady accounts to %s", filePath)

    // Upload the file to S3
    sess := session.Must(session.NewSession())
    s3Svc := s3.New(sess)
    fileForUpload, err := os.Open(filePath)
    if err != nil {
        log.Printf("Failed to open CSV file for upload: %s", err)
        return err
    }
    defer fileForUpload.Close()

    _, err = s3Svc.PutObject(&s3.PutObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(s3Key),
        Body:   fileForUpload,
    })
    if err != nil {
        log.Printf("Failed to upload file to S3: %s", err)
        return err
    }

    log.Printf("Successfully uploaded %s to s3://%s/%s", filepath.Base(filePath), bucket, s3Key)
    return nil
}