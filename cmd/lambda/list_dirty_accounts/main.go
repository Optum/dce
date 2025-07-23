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

    // Assert that dbSvc implements the db.DBer interface
    var _ db.DBer = dbSvc

    // Generate the current date string
    currentDate := time.Now().Format("2006-01-02")
    
    // 1. First call scanAccountsForMissingLPBuckets - this now marks accounts as NotReady in DB
    lpFilePath := fmt.Sprintf("LP_Missing_%s.csv", currentDate)
    bucket := os.Getenv("ARTIFACT_BUCKET_NAME")
    if bucket == "" {
        log.Fatalf("ARTIFACT_BUCKET_NAME environment variable must be set")
    }
    lpS3Key := fmt.Sprintf("LPMissingAccounts/%s", lpFilePath)
    
    log.Printf("Starting scan for accounts missing LP buckets...")
    err := scanAccountsForMissingLPBuckets(dbSvc, lpFilePath, bucket, lpS3Key)
    if err != nil {
        log.Printf("Error scanning for LP buckets: %s", err)

    }
    
    // 2. Then call listNotReadyAccountsToCSV - this now includes LP_Not_Found field
    notReadyFilePath := fmt.Sprintf("not_ready_accounts_%s.csv", currentDate)
    notReadyS3Key := fmt.Sprintf("NotReadyAccounts/%s", notReadyFilePath)
    
    log.Printf("Starting scan for NotReady accounts...")
    err = listNotReadyAccountsToCSV(dbSvc, notReadyFilePath, bucket, notReadyS3Key)
    if err != nil {
        log.Fatalf("Error listing NotReady accounts: %s", err)
    }
    
    log.Println("Both report functions completed successfully")
}