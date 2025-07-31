package main

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/Optum/dce/pkg/db"
)

func initializeDBService() db.DBer {
    // Initialize DB connection from environment variables
    dao, err := db.NewFromEnv()
    if err != nil {
        log.Fatalf("Failed to initialize database: %s", err)
    }
    return dao
}

func main() {
    dbSvc := initializeDBService()

    // Verify required environment variables
    requiredBucketPrefix := os.Getenv("REQUIRED_BUCKET_PREFIX")
    if requiredBucketPrefix == "" {
        log.Fatalf("REQUIRED_BUCKET_PREFIX environment variable must be set")
    }
    log.Printf("Using REQUIRED_BUCKET_PREFIX: %s", requiredBucketPrefix)

    // Assert that dbSvc implements the db.DBer interface
    var _ db.DBer = dbSvc

    // Generate the current date string
    currentDate := time.Now().Format("2006-01-02")
    
    // 1. First call scanAccountsForMissingRequiredBuckets - this now marks accounts as NotReady in DB
    // Use dynamic prefix from environment variable instead of hardcoded value
    prefixFilePath := fmt.Sprintf("Missing_%s_Buckets_%s.csv", requiredBucketPrefix, currentDate)
    bucket := os.Getenv("ARTIFACT_BUCKET_NAME")
    if bucket == "" {
        log.Fatalf("ARTIFACT_BUCKET_NAME environment variable must be set")
    }
    prefixS3Key := fmt.Sprintf("MissingBucketAccounts/%s", prefixFilePath)
    
    log.Printf("Starting scan for accounts missing %s buckets...", requiredBucketPrefix)
    err := scanAccountsForMissingRequiredBuckets(dbSvc, prefixFilePath, bucket, prefixS3Key)
    if err != nil {
        log.Printf("Error scanning for %s buckets: %s", requiredBucketPrefix, err)
    }
    
    // 2. Then call listNotReadyAccountsToCSV - this now includes bucket status field
    notReadyFilePath := fmt.Sprintf("not_ready_accounts_%s.csv", currentDate)
    notReadyS3Key := fmt.Sprintf("NotReadyAccounts/%s", notReadyFilePath)
    
    log.Printf("Starting scan for NotReady accounts...")
    err = listNotReadyAccountsToCSV(dbSvc, notReadyFilePath, bucket, notReadyS3Key)
    if err != nil {
        log.Fatalf("Error listing NotReady accounts: %s", err)
    }
    
    log.Println("Both report functions completed successfully")
}