package db

import (
	"fmt"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDb(t *testing.T) {

	// resetTest is the testing structure used for table driven testing on the
	// Reset Function
	type orphanTests struct {
		Name                 string
		AccountID            string
		CurrentAccountStatus AccountStatus
		GetAccountError      error
		UpdateAccountError   error
		ScanLeasesOutput     *dynamodb.ScanOutput
		ScanLeasesError      error
		CurrentLeaseStatus   LeaseStatus
		AfterLeaseStatus     LeaseStatus
		UpdateLeaseError     error
		ExpectedError        error
		ExpectedAccount      *Account
	}

	tests := []orphanTests{
		// Test with No Messages Received
		{
			Name:                 "Account switched to Orphann but no leases",
			AccountID:            "ABC123",
			CurrentAccountStatus: AccountStatus("Leased"),
			GetAccountError:      nil,
			UpdateAccountError:   nil,
			ScanLeasesOutput:     &dynamodb.ScanOutput{},
			ScanLeasesError:      nil,
			CurrentLeaseStatus:   LeaseStatus("Active"),
			AfterLeaseStatus:     LeaseStatus("Inactive"),
			UpdateLeaseError:     nil,
			ExpectedError:        nil,
			ExpectedAccount: &Account{
				AccountStatus: AccountStatus("Orphaned"),
			},
		},
		{
			Name:                 "Account switched to Orphann and switch lease",
			AccountID:            "ABC123",
			CurrentAccountStatus: AccountStatus("Leased"),
			GetAccountError:      nil,
			UpdateAccountError:   nil,
			ScanLeasesOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"AccountId": {
							S: aws.String("ABC123"),
						},
						"PrincipalId": {
							S: aws.String("Principal"),
						},
						"LeaseStatus": {
							S: aws.String("Active"),
						},
					},
				},
			},
			ScanLeasesError:    nil,
			CurrentLeaseStatus: LeaseStatus("Active"),
			AfterLeaseStatus:   LeaseStatus("Inactive"),
			UpdateLeaseError:   nil,
			ExpectedError:      nil,
			ExpectedAccount: &Account{
				AccountStatus: AccountStatus("Orphaned"),
			},
		},
		{
			Name:                 "Account failed to get",
			AccountID:            "ABC123",
			CurrentAccountStatus: AccountStatus("Leased"),
			GetAccountError:      fmt.Errorf("An error trying to get account ABC123"),
			UpdateAccountError:   nil,
			ScanLeasesOutput:     nil,
			ScanLeasesError:      nil,
			CurrentLeaseStatus:   LeaseStatus("Active"),
			AfterLeaseStatus:     LeaseStatus("Inactive"),
			UpdateLeaseError:     nil,
			ExpectedError:        fmt.Errorf("An error trying to get account ABC123"),
			ExpectedAccount:      nil,
		},
		{
			Name:                 "Account failed to transition a an account",
			AccountID:            "ABC123",
			CurrentAccountStatus: AccountStatus("Leased"),
			GetAccountError:      nil,
			UpdateAccountError:   fmt.Errorf("An error trying to Orphan ABC123"),
			ScanLeasesOutput:     nil,
			ScanLeasesError:      nil,
			CurrentLeaseStatus:   LeaseStatus("Active"),
			AfterLeaseStatus:     LeaseStatus("Inactive"),
			UpdateLeaseError:     nil,
			ExpectedError:        fmt.Errorf("An error trying to Orphan ABC123"),
			ExpectedAccount:      nil,
		},
		{
			Name:                 "Failed to scan leases",
			AccountID:            "ABC123",
			CurrentAccountStatus: AccountStatus("Leased"),
			GetAccountError:      nil,
			UpdateAccountError:   nil,
			ScanLeasesOutput:     nil,
			ScanLeasesError:      fmt.Errorf("An error trying to scan leases"),
			CurrentLeaseStatus:   LeaseStatus("Active"),
			AfterLeaseStatus:     LeaseStatus("Inactive"),
			UpdateLeaseError:     nil,
			ExpectedError:        fmt.Errorf("An error trying to scan leases"),
			ExpectedAccount: &Account{
				AccountStatus: AccountStatus("Orphaned"),
			},
		},
		{
			Name:                 "Update a lease error",
			AccountID:            "ABC123",
			CurrentAccountStatus: AccountStatus("Leased"),
			GetAccountError:      nil,
			UpdateAccountError:   nil,
			ScanLeasesOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"AccountId": {
							S: aws.String("ABC123"),
						},
						"PrincipalId": {
							S: aws.String("Principal"),
						},
						"LeaseStatus": {
							S: aws.String("Active"),
						},
					},
				},
			},
			ScanLeasesError:    nil,
			CurrentLeaseStatus: LeaseStatus("Active"),
			AfterLeaseStatus:   LeaseStatus("Inactive"),
			UpdateLeaseError:   fmt.Errorf("An error trying to update leases"),
			ExpectedError:      fmt.Errorf("An error trying to update leases"),
			ExpectedAccount: &Account{
				AccountStatus: AccountStatus("Orphaned"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("GetItem", &dynamodb.GetItemInput{
				ConsistentRead: aws.Bool(false),
				Key: map[string]*dynamodb.AttributeValue{
					"Id": {
						S: aws.String(test.AccountID),
					},
				},
				TableName: aws.String("account"),
			}).Return(
				&dynamodb.GetItemOutput{
					Item: map[string]*dynamodb.AttributeValue{
						"AccountStatus": {
							S: aws.String(string(test.CurrentAccountStatus)),
						},
					},
				}, test.GetAccountError,
			)

			mockDynamo.On("UpdateItem", mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
				return *input.TableName == "account"
			})).Return(
				&dynamodb.UpdateItemOutput{
					Attributes: map[string]*dynamodb.AttributeValue{
						"AccountStatus": {
							S: aws.String("Orphaned"),
						},
					},
				}, test.UpdateAccountError,
			)

			mockDynamo.On("Scan", &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":status": {
						S: aws.String("Active"),
					},
					":accountId": {
						S: aws.String(test.AccountID),
					},
				},
				FilterExpression: aws.String("LeaseStatus = :status and AccountId = :accountId"),
				TableName:        aws.String("lease"),
			}).Return(
				test.ScanLeasesOutput, test.ScanLeasesError,
			)

			mockDynamo.On("UpdateItem", mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
				return *input.TableName == "lease"
			})).Return(
				&dynamodb.UpdateItemOutput{
					Attributes: map[string]*dynamodb.AttributeValue{
						"LeaseStatus": {
							S: aws.String(string(test.AfterLeaseStatus)),
						},
					},
				}, test.UpdateLeaseError,
			)

			db := DB{
				Client:                   &mockDynamo,
				AccountTableName:         "account",
				LeaseTableName:           "lease",
				DefaultLeaseLengthInDays: 7,
				ConsistentRead:           false,
			}

			newAccount, err := db.OrphanAccount(test.AccountID)

			assert.Equal(t, err, test.ExpectedError, "Error didn't match")
			assert.Equal(t, newAccount, test.ExpectedAccount)
		})
	}
}

func TestGetLeaseByID(t *testing.T) {

	// resetTest is the testing structure used for table driven testing on the
	// Reset Function
	type getLeaseByIDTest struct {
		Name              string
		LeaseID           string
		QueryLeasesOutput *dynamodb.QueryOutput
		QueryLeasesError  error
		ExpectedError     error
		ExpectedLease     *Lease
	}

	tests := []getLeaseByIDTest{
		// Test with No Messages Received
		{
			Name:          "Get leases returns one result",
			LeaseID:       "ABC123",
			ExpectedError: nil,
			QueryLeasesOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"Id": {
							S: aws.String("ABC123"),
						},
					},
				},
			},
			ExpectedLease: &Lease{
				ID: "ABC123",
			},
			QueryLeasesError: nil,
		},
		{
			Name:    "Error when more than one lease",
			LeaseID: "ABC123",

			QueryLeasesOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"Id": {
							S: aws.String("ABC123"),
						},
					},
					{
						"Id": {
							S: aws.String("ABC123"),
						},
					},
				},
			},
			QueryLeasesError: nil,
			ExpectedLease:    nil,
			ExpectedError:    fmt.Errorf("Found more than one Lease with id: ABC123"),
		},
		{
			Name:    "Error when no lease",
			LeaseID: "ABC123",

			QueryLeasesOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			QueryLeasesError: nil,
			ExpectedLease:    nil,
			ExpectedError:    fmt.Errorf("No Lease found with id: ABC123"),
		},
		{
			Name:    "Error when no lease",
			LeaseID: "ABC123",

			QueryLeasesOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			QueryLeasesError: fmt.Errorf("error"),
			ExpectedLease:    nil,
			ExpectedError:    fmt.Errorf("error"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("Query", &dynamodb.QueryInput{
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":a1": {
						S: aws.String(test.LeaseID),
					},
				},
				KeyConditionExpression: aws.String("Id = :a1"),
				IndexName:              aws.String("LeaseId"),
				TableName:              aws.String("lease"),
			}).Return(
				test.QueryLeasesOutput, test.QueryLeasesError,
			)

			db := DB{
				Client:                   &mockDynamo,
				AccountTableName:         "account",
				LeaseTableName:           "lease",
				DefaultLeaseLengthInDays: 7,
				ConsistentRead:           false,
			}

			lease, err := db.GetLeaseByID(test.LeaseID)

			assert.Equal(t, test.ExpectedError, err, "Error didn't match")
			assert.Equal(t, test.ExpectedLease, lease)
		})
	}
}
