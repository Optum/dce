package data

import (
	"fmt"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestGetUsageScan(t *testing.T) {
	tests := []struct {
		name       string
		query      *usage.Usage
		expUsages  *usage.Usages
		expErr     error
		sOutputRec *dynamodb.ScanOutput
		sInput     *dynamodb.ScanInput
		sOutputErr error
	}{
		{
			name:  "scan get all Usages but empty",
			query: &usage.Usage{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usages"),
				Limit:          ptrInt64(25),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expUsages: &usage.Usages{},
		},
		{
			name:  "scan get all Usages",
			query: &usage.Usage{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usages"),
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
			expUsages: &usage.Usages{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "scan get all Usages with accountId",
			query: &usage.Usage{
				AccountID: ptrString("1"),
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Usages"),
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
			expUsages: &usage.Usages{
				{
					AccountID:   ptrString("1"),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name:  "scan failure with internal server error",
			query: &usage.Usage{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usages"),
				Limit:          ptrInt64(25),
			},
			sOutputRec: nil,
			sOutputErr: fmt.Errorf("failure"),
			expUsages:  nil,
			expErr:     errors.NewInternalServer("error getting usages", fmt.Errorf("failure")),
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

			usgData := &Usage{
				DynamoDB:  &mockDynamo,
				TableName: "Usages",
				Limit:     25,
			}
			usgs, err := usgData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expUsages, usgs)
		})
	}

}

func TestGetUsagesQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      *usage.Usage
		expUsages  *usage.Usages
		expErr     error
		qInput     *dynamodb.QueryInput
		qOutputRec *dynamodb.QueryOutput
		qOutputErr error
	}{
		{
			name: "query all Usages by StartDate",
			query: &usage.Usage{
				StartDate: ptrInt64(1580924093),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usages"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("StartDate"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						N: aws.String("1580924093"),
					},
				},
				KeyConditionExpression: aws.String("#0 = :0"),
				Limit:                  ptrInt64(25),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"StartDate": {
							N: aws.String("1580924093"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
					},
				},
			},
			expUsages: &usage.Usages{
				{
					StartDate:   ptrInt64(1580924093),
					PrincipalID: ptrString("User1"),
				},
			},
		},
		{
			name: "query all Usages by StartDate with filter",
			query: &usage.Usage{
				StartDate: ptrInt64(1580924093),
				AccountID: aws.String("123456789012"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usages"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AccountId"),
					"#1": aws.String("StartDate"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":1": {
						N: aws.String("1580924093"),
					},
					":0": {
						S: aws.String("123456789012"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
				Limit:                  ptrInt64(25),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"StartDate": {
							N: aws.String("1580924093"),
						},
						"AccountId": {
							S: aws.String("123456789012"),
						},
					},
				},
			},
			expUsages: &usage.Usages{
				{
					StartDate: ptrInt64(1580924093),
					AccountID: ptrString("123456789012"),
				},
			},
		},
		{
			name: "query internal error",
			query: &usage.Usage{
				StartDate: ptrInt64(1580924093),
				AccountID: aws.String("123456789012"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usages"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AccountId"),
					"#1": aws.String("StartDate"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("123456789012"),
					},
					":1": {
						N: aws.String("1580924093"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
				Limit:                  ptrInt64(25),
			},
			qOutputRec: nil,
			qOutputErr: fmt.Errorf("failure"),
			expUsages:  nil,
			expErr:     errors.NewInternalServer("failed to query usages", fmt.Errorf("failure")),
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

			leaseData := &Usage{
				DynamoDB:  &mockDynamo,
				TableName: "Usages",
				Limit:     25,
			}
			Usages, err := leaseData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expUsages, Usages)
		})
	}

}
