package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Force keep imports by referencing them in a blank identifier declaration
var (
    _ = csv.NewWriter
    _ = aws.String
    _ = session.Must
    _ = s3.New
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

    // Generate the file name with the current date
    currentDate := time.Now().Format("2006-01-02")
    filePath := fmt.Sprintf("not_ready_accounts_%s.csv", currentDate)

    bucket := os.Getenv("ARTIFACT_BUCKET_NAME")
    s3Key := fmt.Sprintf("NotReadyAccounts/%s", filePath)

    err := listNotReadyAccountsToCSV(dbSvc, filePath, bucket, s3Key)
    if err != nil {
        log.Fatalf("Error: %s", err)
    }
}