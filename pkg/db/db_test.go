package db

import (
	"fmt"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

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
