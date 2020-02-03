package data

import (
	"fmt"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestGetLeasesScan(t *testing.T) {
	tests := []struct {
		name       string
		query      *lease.Lease
		expLeases  *lease.Leases
		expErr     error
		sOutputRec *dynamodb.ScanOutput
		sInput     *dynamodb.ScanInput
		sOutputErr error
	}{
		{
			name:  "scan get all leases but empty",
			query: &lease.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				Limit:          ptrInt64(25),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expLeases: &lease.Leases{},
		},
		{
			name:  "scan get all leases",
			query: &lease.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				Limit:          ptrInt64(25),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"AccountId": {
							S: aws.String("1"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
					},
				},
			},
			expLeases: &lease.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "scan get all leases with accountId",
			query: &lease.Lease{
				AccountID: ptrString("1"),
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Leases"),
				FilterExpression: aws.String("#0 = :0"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AccountId"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("1"),
					},
				},
				Limit: ptrInt64(25),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"AccountId": {
							S: aws.String("1"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
					},
				},
			},
			expLeases: &lease.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name:  "scan failure with internal server error",
			query: &lease.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				Limit:          ptrInt64(25),
			},
			sOutputRec: nil,
			sOutputErr: fmt.Errorf("failure"),
			expLeases:  nil,
			expErr:     errors.NewInternalServer("error getting leases", fmt.Errorf("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			if tt.sInput != nil {
				mockDynamo.On("Scan", tt.sInput).Return(
					tt.sOutputRec, tt.sOutputErr,
				)
			}

			leaseData := &Lease{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
				Limit:     25,
			}
			leases, err := leaseData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expLeases, leases)
		})
	}

}

func TestGetLeasesQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      *lease.Lease
		expLeases  *lease.Leases
		expErr     error
		qInput     *dynamodb.QueryInput
		qOutputRec *dynamodb.QueryOutput
		qOutputErr error
	}{
		{
			name: "query all leases by status",
			query: &lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				IndexName:      aws.String("LeaseStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("LeaseStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Active"),
					},
				},
				KeyConditionExpression: aws.String("#0 = :0"),
				Limit:                  ptrInt64(25),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"AccountId": {
							S: aws.String("1"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
					},
				},
			},
			expLeases: &lease.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "query all leases by status with filter",
			query: &lease.Lease{
				Status:      lease.StatusActive.StatusPtr(),
				PrincipalID: aws.String("User1"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				IndexName:      aws.String("PrincipalId"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("LeaseStatus"),
					"#1": aws.String("PrincipalId"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Active"),
					},
					":1": {
						S: aws.String("User1"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
				Limit:                  ptrInt64(25),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"AccountId": {
							S: aws.String("1"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
					},
				},
			},
			expLeases: &lease.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "query internal error",
			query: &lease.Lease{
				Status:      lease.StatusActive.StatusPtr(),
				PrincipalID: aws.String("User1"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				IndexName:      aws.String("PrincipalId"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("LeaseStatus"),
					"#1": aws.String("PrincipalId"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Active"),
					},
					":1": {
						S: aws.String("User1"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
				Limit:                  ptrInt64(25),
			},
			qOutputRec: nil,
			qOutputErr: fmt.Errorf("failure"),
			expLeases:  nil,
			expErr:     errors.NewInternalServer("failed to query leases", fmt.Errorf("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			if tt.qInput != nil {
				mockDynamo.On("Query", tt.qInput).Return(
					tt.qOutputRec, tt.qOutputErr,
				)
			}

			leaseData := &Lease{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
				Limit:     25,
			}
			leases, err := leaseData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expLeases, leases)
		})
	}

}
