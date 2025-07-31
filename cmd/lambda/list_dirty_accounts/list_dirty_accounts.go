package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
)

// scanAccountsForMissingRequiredBuckets scans all accounts for required S3 buckets
// If accounts don't have such buckets, marks them as NotReady in the database
func scanAccountsForMissingRequiredBuckets(dbSvc db.DBer, filePath, bucket, s3Key string) error {
    log.Println("Scanning accounts for missing buckets (excluding Leased accounts)")

    // Get accounts with different statuses and combine them
    readyAccounts, err := dbSvc.FindAccountsByStatus(db.Ready)
    if err != nil {
        log.Printf("Failed to fetch Ready accounts: %s", err)
        return err
    }

    notReadyAccounts, err := dbSvc.FindAccountsByStatus(db.NotReady)
    if err != nil {
        log.Printf("Failed to fetch NotReady accounts: %s", err)
        return err
    }

    // Combine accounts (we check only Ready and NotReady accounts)
    var accounts []*db.Account
    accounts = append(accounts, readyAccounts...)
    accounts = append(accounts, notReadyAccounts...)

    log.Printf("Found %d total accounts to check", len(accounts))

    awsRegion := os.Getenv("AWS_CURRENT_REGION")
    if awsRegion == "" {
        awsRegion = "us-east-1"
    }

    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String(awsRegion),
    }))

    // Count of accounts marked as NotReady
    markedCount := 0

    // For each account, check for required buckets
    for _, account := range accounts {
        // Use the account credentials to check for required buckets
        hasBucket, err := checkForBuckets(sess, account.ID, dbSvc)
        if err != nil {
            log.Printf("Error checking required buckets for account %s: %s", account.ID, err)
            continue
        }

        if !hasBucket {
            log.Printf("Account %s is missing required buckets - marking as NotReady", account.ID)

            if account.Metadata == nil {
                account.Metadata = make(map[string]interface{})
            }

            account.Metadata["BucketExists"] = false
            account.Metadata["BucketNotFound"] = true
            account.Metadata["Reason"] = "Required bucket doesn't exist"

            if account.AccountStatus != db.NotReady {
                account.AccountStatus = db.NotReady
                markedCount++
            }

            err := dbSvc.PutAccount(*account)
            if err != nil {
                log.Printf("Failed to update account %s status: %s", account.ID, err)
            }
        } else if account.AccountStatus == db.NotReady {
            if requiredBucket, ok := account.Metadata["BucketNotFound"].(bool); ok && requiredBucket {
                log.Printf("Account %s has required buckets but was marked as NotReady - updating metadata", account.ID)
                account.Metadata["BucketExists"] = true
                account.Metadata["BucketNotFound"] = false

                err := dbSvc.PutAccount(*account)
                if err != nil {
                    log.Printf("Failed to update account %s metadata: %s", account.ID, err)
                }
            }
        }
    }

    log.Printf("Marked or updated %d accounts due to missing required buckets", markedCount)
    return nil
}
// checkForBuckets checks if an account has S3 buckets starting with the required prefix
func checkForBuckets(sess *session.Session, accountID string, dbSvc db.DBer) (bool, error) {
    // Retrieve the bucket prefix from the environment variable
    bucketPrefix := os.Getenv("REQUIRED_BUCKET_PREFIX")
    if bucketPrefix == "" {
        return false, fmt.Errorf("REQUIRED_BUCKET_PREFIX environment variable is not set")
    }

    // Get the account record from database to retrieve AdminRoleArn
    account, err := dbSvc.GetAccount(accountID)
    if err != nil {
        return false, fmt.Errorf("failed to get account %s from database: %w", accountID, err)
    }

    // Use AdminRoleArn from account record
    adminRoleArn := account.AdminRoleArn
    if adminRoleArn == "" {
        return false, fmt.Errorf("AdminRoleArn not found for account %s", accountID)
    }

    // Create STS client for assuming roles
    stsSvc := sts.New(sess)
    sessionName := fmt.Sprintf("Bucket-Check-%s", time.Now().Format("20060102-150405"))

    // Assume the role using the AdminRoleArn from database
    log.Printf("Assuming role %s for bucket check", adminRoleArn)
    assumeRoleInput := &sts.AssumeRoleInput{
        RoleArn:         aws.String(adminRoleArn),
        RoleSessionName: aws.String(sessionName),
        DurationSeconds: aws.Int64(900),
    }

    log.Printf("Attempting to assume role %s in account %s", adminRoleArn, accountID)
    assumeRoleOutput, err := stsSvc.AssumeRole(assumeRoleInput)
    if err != nil {
        return false, fmt.Errorf("failed to assume role %s in account %s: %w", adminRoleArn, accountID, err)
    }

    crossAccountCreds := credentials.NewStaticCredentials(
        *assumeRoleOutput.Credentials.AccessKeyId,
        *assumeRoleOutput.Credentials.SecretAccessKey,
        *assumeRoleOutput.Credentials.SessionToken,
    )

    crossAccountConfig := aws.NewConfig().
        WithCredentials(crossAccountCreds).
        WithRegion(aws.StringValue(sess.Config.Region))

    crossAccountSess := session.Must(session.NewSession(crossAccountConfig))

    s3Svc := s3.New(crossAccountSess)

    result, err := s3Svc.ListBuckets(&s3.ListBucketsInput{})
    if err != nil {
        return false, fmt.Errorf("failed to list buckets for account %s using role %s: %w", accountID, adminRoleArn, err)
    }

    // Check if any bucket name starts with the required prefix
    for _, bucket := range result.Buckets {
        if bucket.Name != nil && len(*bucket.Name) >= len(bucketPrefix) && (*bucket.Name)[:len(bucketPrefix)] == bucketPrefix {
            log.Printf("Found required bucket %s in account %s using role %s", *bucket.Name, accountID, adminRoleArn)
            return true, nil
        }
    }

    log.Printf("No required buckets found in account %s using role %s", accountID, adminRoleArn)
    return false, nil
}
// listNotReadyAccountsToCSV retrieves all accounts with the status "NotReady" from the ACCOUNT_TABLE,
// saves them to a CSV file, and uploads the file to the specified S3 bucket.
func listNotReadyAccountsToCSV(dbSvc db.DBer, filePath, bucket, s3Key string) error {
    log.Println("Fetching all accounts with status NotReady from ACCOUNT_TABLE")

    // Query the ACCOUNT_TABLE for accounts with status "NotReady"
    accounts, err := dbSvc.FindAccountsByStatus(db.NotReady)
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

    // Write the header row with additional field for required Not Found
    err = writer.Write([]string{"AccountID", "Status", "LastUpdated", "Reason", "required_Buckets_Not_Found"})
    if err != nil {
        log.Printf("Failed to write header to CSV file: %s", err)
        return err
    }

    // Write account data to the CSV file
    for _, account := range accounts {
        // Default reason if not specified
        reason := "Account marked as NotReady"
        
        // Default required not found value
        BucketNotFound := "false"
        
        if account.Metadata != nil {
            if r, ok := account.Metadata["Reason"].(string); ok && r != "" {
                reason = r
            }
            
            // Check if required not found flag is set
            if requiredBucket, ok := account.Metadata["BucketNotFound"].(bool); ok && requiredBucket {
                BucketNotFound = "true"
            }
        }
        
        err := writer.Write([]string{
            account.ID,
            string(account.AccountStatus),
            time.Unix(account.LastModifiedOn, 0).Format(time.RFC3339),
            reason,
            BucketNotFound,
        })
        if err != nil {
            log.Printf("Failed to write account data to CSV file: %s", err)
            return err
        }
    }

    writer.Flush()

    if err := writer.Error(); err != nil {
        log.Printf("CSV writer error: %s", err)
        return err
    }

    file.Close()

    log.Printf("Successfully saved NotReady accounts to %s", filePath)

    // Upload the file to S3
    awsRegion := os.Getenv("AWS_CURRENT_REGION")
    if awsRegion == "" {
        awsRegion = "us-east-1" 
    }
    
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String(awsRegion),
    }))
    s3Svc := s3.New(sess)
    fileForUpload, err := os.Open(filePath)
    if err != nil {
        log.Printf("Failed to open CSV file for upload: %s", err)
        return err
    }
    defer fileForUpload.Close()

    if bucket == "" {
        log.Printf("Error: S3 bucket name is empty. Set ARTIFACT_BUCKET_NAME environment variable.")
        return fmt.Errorf("S3 bucket name cannot be empty")
    }

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