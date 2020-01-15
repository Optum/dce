package data

import (
	"fmt"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestGetLeasesScan(t *testing.T) {
	tests := []struct {
		name       string
		query      model.Lease
		expLeases  *model.Leases
		expErr     error
		sOutputRec *dynamodb.ScanOutput
		sInput     *dynamodb.ScanInput
		sOutputErr error
	}{
		{
			name:  "scan get all leases but empty",
			query: model.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expLeases: &model.Leases{},
		},
		{
			name:  "scan get all leases",
			query: model.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
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
			expLeases: &model.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "scan get all leases with principalId",
			query: model.Lease{
				PrincipalID: ptrString("User1"),
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Leases"),
				FilterExpression: aws.String("#0 = :0"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("User1"),
					},
				},
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
			expLeases: &model.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name:  "scan failure with internal server error",
			query: model.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
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

			leaseData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
			}
			leases, err := leaseData.GetLeases(&tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expLeases, leases)
		})
	}

}

func TestGetLeasesQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      model.Lease
		expLeases  *model.Leases
		expErr     error
		qInput     *dynamodb.QueryInput
		qOutputRec *dynamodb.QueryOutput
		qOutputErr error
	}{
		{
			name: "query all leases by status",
			query: model.Lease{
				LeaseStatus: model.LeaseStatusActive.LeaseStatusPtr(),
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
			expLeases: &model.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "query all leases by status with filter",
			query: model.Lease{
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
				PrincipalID: aws.String("User1"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				IndexName:      aws.String("LeaseStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("LeaseStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("User1"),
					},
					":1": {
						S: aws.String("Active"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
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
			expLeases: &model.Leases{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "query internal error",
			query: model.Lease{
				LeaseStatus:  model.LeaseStatusActive.LeaseStatusPtr(),
				PrincipalID: aws.String("User1"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Leases"),
				IndexName:      aws.String("LeaseStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("LeaseStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("User1"),
					},
					":1": {
						S: aws.String("Active"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
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

			leaseData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
			}
			leases, err := leaseData.GetLeases(&tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expLeases, leases)
		})
	}

}
