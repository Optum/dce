package data

import (
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestGetAccounts(t *testing.T) {
	statusReady := model.Ready

	tests := []struct {
		name        string
		query       model.Account
		result      *model.Accounts
		err         error
		scanOutput  *dynamodb.ScanOutput
		scanInput   *dynamodb.ScanInput
		queryInput  *dynamodb.QueryInput
		queryOuptut *dynamodb.QueryOutput
	}{
		{
			name:  "get all accounts but empty",
			query: model.Account{},
			scanInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
			},
			scanOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
		},
		{
			name:  "get all accounts",
			query: model.Account{},
			scanInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
			},
			scanOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			result: &model.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name: "get all accounts with admin role arn",
			query: model.Account{
				AdminRoleArn: ptrString("test:arn"),
			},
			scanInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Accounts"),
				FilterExpression: aws.String("#0 = :0"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AdminRoleArn"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("test:arn"),
					},
				},
			},
			scanOutput: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			result: &model.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name: "get all accounts by status",
			query: model.Account{
				Status: &statusReady,
			},
			queryInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				IndexName:      aws.String("AccountStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AccountStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Ready"),
					},
				},
				KeyConditionExpression: aws.String("#0 = :0"),
			},
			queryOuptut: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			result: &model.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name: "get all accounts by status with filter",
			query: model.Account{
				Status:       &statusReady,
				AdminRoleArn: aws.String("test:arn"),
			},
			queryInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				IndexName:      aws.String("AccountStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AdminRoleArn"),
					"#1": aws.String("AccountStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("test:arn"),
					},
					":1": {
						S: aws.String("Ready"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
			},
			queryOuptut: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			result: &model.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			if tt.scanInput != nil {
				mockDynamo.On("Scan", tt.scanInput).Return(
					tt.scanOutput, nil,
				)
			}

			if tt.queryInput != nil {
				mockDynamo.On("Query", tt.queryInput).Return(
					tt.queryOuptut, nil,
				)
			}

			accountData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Accounts",
			}
			accounts, err := accountData.GetAccounts(&tt.query)
			assert.NoError(t, err)
			if tt.result == nil {
				assert.Empty(t, accounts)
			} else {
				assert.Equal(t, tt.result, accounts)
			}
		})
	}

}
