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

func TestListUsagePrincipals(t *testing.T) {
	now := time.Now().UTC().Unix()
	var five int64 = 5

	tests := []struct {
		name       string
		query      *usage.Principal
		expUsgs    *usage.Principals
		expErr     error
		expQuery   *usage.Principal
		sOutputRec *dynamodb.ScanOutput
		sInput     *dynamodb.ScanInput
		sOutputErr error
	}{
		{
			name:  "scan get all usage but empty",
			query: &usage.Principal{},
			expQuery: &usage.Principal{
				Limit: &five,
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				Limit:          aws.Int64(5),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Usage-Principal-"),
					},
				},
				FilterExpression: aws.String("begins_with (#0, :0)"),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expUsgs: &usage.Principals{},
		},
		{
			name: "scan get all usage with pagniation",
			query: &usage.Principal{
				NextPrincipalID: aws.String("user1"),
				NextDate:        &now,
				Limit:           &five,
			},
			expQuery: &usage.Principal{
				NextPrincipalID: aws.String("user2"),
				NextDate:        &now,
				Limit:           &five,
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				Limit:          aws.Int64(5),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Usage-Principal-"),
					},
				},
				FilterExpression: aws.String("begins_with (#0, :0)"),
				ExclusiveStartKey: map[string]*dynamodb.AttributeValue{
					"PrincipalId": {
						S: aws.String("user1"),
					},
					"SK": {
						S: aws.String(fmt.Sprintf("%s%s", usagePrincipalSkPrefix, strconv.FormatInt(now, 10))),
					},
				},
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("user1"),
						},
					},
				},
				LastEvaluatedKey: map[string]*dynamodb.AttributeValue{
					"PrincipalId": {
						S: aws.String("user2"),
					},
					"SK": {
						S: aws.String(fmt.Sprintf("%s%s", usagePrincipalSkPrefix, strconv.FormatInt(now, 10))),
					},
				},
			},
			expUsgs: &usage.Principals{
				{
					PrincipalID: ptrString("user1"),
				},
			},
		},
		{
			name:  "scan failure with internal server error",
			query: &usage.Principal{},
			expQuery: &usage.Principal{
				Limit: &five,
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				Limit:          aws.Int64(5),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Usage-Principal-"),
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

			usageData := &UsagePrincipal{
				DynamoDB:  &mockDynamo,
				TableName: "Usage",
				Limit:     5,
			}

			usgs, err := usageData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expUsgs, usgs)
			assert.Equal(t, tt.expQuery, tt.query)
		})
	}

}

func TestListUsagePrincipalsQuery(t *testing.T) {
	begOfWeek := getBudgetPeriodTime(time.Now(), "WEEKLY")
	begOfWeekUnix := begOfWeek.Unix()
	begOfNextWeek := begOfWeek.AddDate(0, 0, 7).Unix()
	var five int64 = 5

	tests := []struct {
		name       string
		query      *usage.Principal
		expUsgs    *usage.Principals
		expQuery   *usage.Principal
		expErr     error
		qInput     *dynamodb.QueryInput
		qOutputRec *dynamodb.QueryOutput
		qOutputErr error
	}{
		{
			name: "query all usage by principal and date",
			query: &usage.Principal{
				PrincipalID: aws.String("user1"),
			},
			expQuery: &usage.Principal{
				PrincipalID: aws.String("user1"),
				Limit:       &five,
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("user1"),
					},
					":1": {
						S: aws.String("Usage-Principal-"),
					},
				},
				KeyConditionExpression: aws.String("(#0 = :0) AND (begins_with (#1, :1))"),
				Limit:                  aws.Int64(5),
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
			expUsgs: &usage.Principals{
				{
					PrincipalID: aws.String("user1"),
				},
			},
		},
		{
			name: "query all usages by principal ID with pagination",
			query: &usage.Principal{
				PrincipalID:     aws.String("user1"),
				NextDate:        &begOfWeekUnix,
				NextPrincipalID: aws.String("user1"),
			},
			expQuery: &usage.Principal{
				PrincipalID:     aws.String("user1"),
				NextDate:        &begOfNextWeek,
				NextPrincipalID: aws.String("user2"),
				Limit:           &five,
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("user1"),
					},
					":1": {
						S: aws.String(usagePrincipalSkPrefix),
					},
				},
				KeyConditionExpression: aws.String("(#0 = :0) AND (begins_with (#1, :1))"),
				Limit:                  aws.Int64(5),
				ExclusiveStartKey: map[string]*dynamodb.AttributeValue{
					"PrincipalId": {
						S: aws.String("user1"),
					},
					"SK": {
						S: aws.String(fmt.Sprintf("%s%s", usagePrincipalSkPrefix, strconv.FormatInt(begOfWeekUnix, 10))),
					},
				},
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("user1"),
						},
					},
				},
				LastEvaluatedKey: map[string]*dynamodb.AttributeValue{
					"PrincipalId": {
						S: aws.String("user2"),
					},
					"SK": {
						S: aws.String(fmt.Sprintf("%s%s", usagePrincipalSkPrefix, strconv.FormatInt(begOfNextWeek, 10))),
					},
				},
			},
			expUsgs: &usage.Principals{
				{
					PrincipalID: ptrString("user1"),
				},
			},
		},
		{
			name: "query internal error",
			query: &usage.Principal{
				PrincipalID: aws.String("user1"),
			},
			expQuery: &usage.Principal{
				PrincipalID: aws.String("user1"),
				Limit:       &five,
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Usage"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("PrincipalId"),
					"#1": aws.String("SK"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("user1"),
					},
					":1": {
						S: aws.String("Usage-Principal-"),
					},
				},
				KeyConditionExpression: aws.String("(#0 = :0) AND (begins_with (#1, :1))"),
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

			usageData := &UsagePrincipal{
				DynamoDB:  &mockDynamo,
				TableName: "Usage",
				Limit:     5,
			}

			usgs, err := usageData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expUsgs, usgs)
			assert.Equal(t, tt.expQuery, tt.query)
		})
	}

}
