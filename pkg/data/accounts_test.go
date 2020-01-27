package data

import (
	"fmt"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountsScan(t *testing.T) {
	tests := []struct {
		name        string
		query       *account.Account
		expAccounts *account.Accounts
		expErr      error
		sOutputRec  *dynamodb.ScanOutput
		sInput      *dynamodb.ScanInput
		sOutputErr  error
	}{
		{
			name:  "scan get all accounts but empty",
			query: &account.Account{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				Limit:          aws.Int64(5),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expAccounts: &account.Accounts{},
		},
		{
			name:  "scan get all accounts",
			query: &account.Account{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				Limit:          aws.Int64(5),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			expAccounts: &account.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name: "scan get all accounts with admin role arn",
			query: &account.Account{
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			sInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Accounts"),
				FilterExpression: aws.String("#0 = :0"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AdminRoleArn"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("arn:aws:iam::123456789012:role/AdminRole"),
					},
				},
				Limit: aws.Int64(5),
			},
			sOutputRec: &dynamodb.ScanOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			expAccounts: &account.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name:  "scan failure with internal server error",
			query: &account.Account{},
			sInput: &dynamodb.ScanInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				Limit:          aws.Int64(5),
			},
			sOutputRec:  nil,
			sOutputErr:  fmt.Errorf("failure"),
			expAccounts: nil,
			expErr:      errors.NewInternalServer("error getting accounts", fmt.Errorf("failure")),
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

			accountData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Accounts",
				Limit:     5,
			}
			accounts, err := accountData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expAccounts, accounts)
		})
	}

}

func TestGetAccountsQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       *account.Account
		expAccounts *account.Accounts
		expErr      error
		qInput      *dynamodb.QueryInput
		qOutputRec  *dynamodb.QueryOutput
		qOutputErr  error
	}{
		{
			name: "query all accounts by status",
			query: &account.Account{
				Status: account.StatusReady.StatusPtr(),
			},
			qInput: &dynamodb.QueryInput{
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
				Limit:                  aws.Int64(5),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			expAccounts: &account.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name: "query all accounts by status with filter",
			query: &account.Account{
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				IndexName:      aws.String("AccountStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AdminRoleArn"),
					"#1": aws.String("AccountStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("arn:aws:iam::123456789012:role/AdminRole"),
					},
					":1": {
						S: aws.String("Ready"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
				Limit:                  aws.Int64(5),
			},
			qOutputRec: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					map[string]*dynamodb.AttributeValue{
						"Id": {
							S: aws.String("1"),
						},
					},
				},
			},
			expAccounts: &account.Accounts{
				{
					ID: ptrString("1"),
				},
			},
		},
		{
			name: "query internal error",
			query: &account.Account{
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRole"),
			},
			qInput: &dynamodb.QueryInput{
				ConsistentRead: aws.Bool(false),
				TableName:      aws.String("Accounts"),
				IndexName:      aws.String("AccountStatus"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AdminRoleArn"),
					"#1": aws.String("AccountStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("arn:aws:iam::123456789012:role/AdminRole"),
					},
					":1": {
						S: aws.String("Ready"),
					},
				},
				KeyConditionExpression: aws.String("#1 = :1"),
				FilterExpression:       aws.String("#0 = :0"),
				Limit:                  aws.Int64(5),
			},
			qOutputRec:  nil,
			qOutputErr:  fmt.Errorf("failure"),
			expAccounts: nil,
			expErr:      errors.NewInternalServer("failed to query accounts", fmt.Errorf("failure")),
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

			accountData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Accounts",
				Limit:     5,
			}
			accounts, err := accountData.List(tt.query)
			assert.True(t, errors.Is(err, tt.expErr))
			assert.Equal(t, tt.expAccounts, accounts)
		})
	}

}
