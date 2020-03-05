package data

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestListUsageLeases(t *testing.T) {
	tests := []struct {
		name       string
		query      *usage.Lease
		expUsgs    *usage.Leases
		expErr     error
		sOutputRec *dynamodb.ScanOutput
		sInput     *dynamodb.ScanInput
		sOutputErr error
	}{
		{
			name:  "scan get all usage but empty",
			query: &usage.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				Limit:          aws.Int64(5),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Usage-Lease-Summary-"),
					},
				},
				FilterExpression: aws.String("begins_with (#0, :0)"),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expUsgs: &usage.Leases{},
		},
		{
			name:  "scan get all Lease Usage with pagniation",
			query: &usage.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				Limit:          aws.Int64(5),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Usage-Lease-Summary-"),
					},
				},
				FilterExpression: aws.String("begins_with (#0, :0)"),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("user-1"),
						},
					},
				},
			},
			expUsgs: &usage.Leases{
				{
					PrincipalID: ptrString("user-1"),
				},
			},
		},
		{
			name: "scan get all lease usage with principal ID",
			query: &usage.Lease{
				PrincipalID: aws.String("user1"),
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Usage"),
				FilterExpression: aws.String("(#0 = :0) AND (begins_with (#1, :1))"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("user1"),
					},
					":1": {
						S: aws.String("Usage-Lease-Summary-"),
					},
				},
				Limit: aws.Int64(5),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("user1"),
						},
					},
				},
			},
			expUsgs: &usage.Leases{
				{
					PrincipalID: aws.String("user1"),
				},
			},
		},
		{
			name:  "scan failure with internal server error",
			query: &usage.Lease{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				Limit:          aws.Int64(5),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Usage-Lease-Summary-"),
					},
				},
				FilterExpression: aws.String("begins_with (#0, :0)"),
			},
			sOutputRec: nil,
			sOutputErr: fmt.Errorf("failure"),
			expUsgs:    nil,
			expErr:     errors.NewInternalServer("error getting usage", fmt.Errorf("failure")),
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

			usageData := &UsageLease{
				DynamoDB:  &mockDynamo,
				TableName: "Usage",
				Limit:     5,
			}
			fmt.Printf("%+v\n", tt.name)
			usgs, err := usageData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expUsgs, usgs)
		})
	}

}

func TestGetListUsageQuery(t *testing.T) {
	begOfWeek := getBudgetPeriodTime(time.Now(), "WEEKLY")

	tests := []struct {
		name       string
		query      *usage.Lease
		expUsgs    *usage.Leases
		expErr     error
		qInput     *dynamodb.QueryInput
		qOutputRec *dynamodb.QueryOutput
		qOutputErr error
	}{
		{
			name: "query all Lease Usage by Principal and Date",
			query: &usage.Lease{
				PrincipalID: aws.String("user1"),
				Date:        &begOfWeek,
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("Date"),
					"#2": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("user1"),
					},
					":1": {
						N: aws.String(strconv.FormatInt(begOfWeek.Unix(), 10)),
					},
					":2": {
						S: aws.String("Usage-Lease-Summary-"),
					},
				},
				FilterExpression:       aws.String("(#0 = :0) AND (#1 = :1)"),
				KeyConditionExpression: aws.String("begins_with (#2, :2)"),
				Limit:                  aws.Int64(5),
				IndexName:              aws.String("SortKey"),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("user1"),
						},
					},
				},
			},
			expUsgs: &usage.Leases{
				{
					PrincipalID: aws.String("user1"),
				},
			},
		},
		{
			name: "query internal error",
			query: &usage.Lease{
				PrincipalID: aws.String("user1"),
				Date:        &begOfWeek,
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				IndexName:      aws.String("SortKey"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("Date"),
					"#2": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("user1"),
					},
					":1": {
						N: aws.String(strconv.FormatInt(begOfWeek.Unix(), 10)),
					},
					":2": {
						S: aws.String("Usage-Lease-Summary-"),
					},
				},
				KeyConditionExpression: aws.String("begins_with (#2, :2)"),
				FilterExpression:       aws.String("(#0 = :0) AND (#1 = :1)"),
				Limit:                  aws.Int64(5),
			},
			qOutputRec: nil,
			qOutputErr: fmt.Errorf("failure"),
			expUsgs:    nil,
			expErr:     errors.NewInternalServer("failed to query usage", fmt.Errorf("failure")),
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

			usageData := &UsageLease{
				DynamoDB:  &mockDynamo,
				TableName: "Usage",
				Limit:     5,
			}
			usgs, err := usageData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expUsgs, usgs)
		})
	}

}
