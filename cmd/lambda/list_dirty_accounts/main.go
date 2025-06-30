package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Optum/dce/pkg/db"
)

func initializeDBService() db.DBer {
    dao, err := db.NewFromEnv()
    if err != nil {
        errorMessage := fmt.Sprintf("Failed to initialize database: %s", err)
        log.Fatal(errorMessage)
    }
    return dao
}

func main() {
    dbSvc := initializeDBService() // Your DB service initialization

    // Assert that dbSvc implements the db.DBer interface
    var _ db.DBer = dbSvc

    filePath := "not_ready_accounts.csv"
    bucket := os.Getenv("ARTIFACT_BUCKET_NAME")
    s3Key := "NotReadyAccounts/not_ready_accounts.csv"

    err := listNotReadyAccountsToCSV(dbSvc, filePath, bucket, s3Key)
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
}