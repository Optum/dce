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
		name       string
		query      model.Account
		result     *model.Accounts
		err        error
		scanOutput *dynamodb.ScanOutput
		scanInput  *dynamodb.ScanInput
	}{
		{
			name:  "getAllAccountsEmpty",
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
			name:  "getAllAccounts",
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
			name: "getAllAccounts",
			query: model.Account{
				Status: &statusReady,
			},
			scanInput: &dynamodb.ScanInput{
				ConsistentRead:   aws.Bool(false),
				TableName:        aws.String("Accounts"),
				FilterExpression: aws.String("#0 = :0"),
				ExpressionAttributeNames: map[string]*string{
					"#0": aws.String("AccountStatus"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":0": {
						S: aws.String("Ready"),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}
			mockDynamo.On("Scan", tt.scanInput).Return(
				tt.scanOutput, nil,
			)

			accountData := &Account{
				AwsDynamoDB: &mockDynamo,
				TableName:   "Accounts",
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