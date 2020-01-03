package data

import (
	gErrors "errors"
	"fmt"
	"log"
	"strconv"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGet(t *testing.T) {

	accountStatusReady := model.Ready
	// accountStatusNotReady := model.NotReady

	tests := []struct {
		name            string
		accountID       string
		dynamoErr       error
		dynamoOutput    *dynamodb.GetItemOutput
		expectedErr     error
		expectedAccount model.Account
	}{
		{
			name:      "should return an account object",
			accountID: "abc123",
			expectedAccount: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:arn"),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"Id": {
						S: aws.String("abc123"),
					},
					"AccountStatus": {
						S: aws.String("Ready"),
					},
					"LastModifiedOn": {
						N: aws.String(strconv.Itoa(1573592058)),
					},
					"AdminRoleArn": {
						S: aws.String("test:arn"),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:            "should return nil object when not found",
			accountID:       "abc123",
			expectedAccount: model.Account{},
			dynamoErr:       nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewNotFound("account", "abc123"),
		},
		{
			name:            "should return nil when dynamodb err",
			accountID:       "abc123",
			expectedAccount: model.Account{},
			dynamoErr:       gErrors.New("failure"),
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewInternalServer("get failed for account \"abc123\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("GetItem", mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
				return (*input.TableName == "Accounts" &&
					*input.Key["Id"].S == tt.accountID)
			})).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			accountData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Accounts",
			}

			item := &model.Account{}
			err := accountData.GetAccountByID(tt.accountID, item)
			assert.Equal(t, tt.expectedAccount, *item)
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}

func TestDelete(t *testing.T) {

	accountStatusReady := model.Ready
	// accountStatusNotReady := model.NotReady

	tests := []struct {
		name         string
		account      model.Account
		dynamoErr    error
		dynamoOutput *dynamodb.DeleteItemOutput
		expectedErr  error
	}{
		{
			name: "should delete an account",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.DeleteItemOutput{
				Attributes: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: nil,
		},
		{
			name: "should delete an account",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			dynamoErr: gErrors.New("failure"),
			dynamoOutput: &dynamodb.DeleteItemOutput{
				Attributes: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewInternalServer("delete failed for account \"abc123\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("DeleteItem", mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
				return (*input.TableName == "Accounts" &&
					*input.Key["Id"].S == *tt.account.ID)
			})).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			accountData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Accounts",
			}

			err := accountData.DeleteAccount(&tt.account)
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}

func TestUpdate(t *testing.T) {
	accountStatusReady := model.Ready
	// accountStatusNotReady := model.NotReady

	tests := []struct {
		name              string
		account           model.Account
		dynamoErr         error
		expectedErr       error
		oldLastModifiedOn *int64
	}{
		{
			name: "update",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         nil,
			expectedErr:       nil,
		},
		{
			name: "create",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			dynamoErr:   nil,
			expectedErr: nil,
		},
		{
			name: "conditional failure",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         awserr.New("ConditionalCheckFailedException", "Message", fmt.Errorf("Bad")),
			expectedErr:       errors.NewConflict("account", "abc123", fmt.Errorf("unable to update account with LastModifiedOn=\"1573592058\"")),
		},
		{
			name: "other dynamo error",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         gErrors.New("failure"),
			expectedErr:       errors.NewInternalServer("update failed for account \"abc123\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("PutItem", mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
				if tt.oldLastModifiedOn == nil {
					return (*input.TableName == "Accounts" &&
						*input.Item["Id"].S == *tt.account.ID &&
						*input.Item["AccountStatus"].S == string(*tt.account.Status) &&
						*input.Item["LastModifiedOn"].N == strconv.FormatInt(*tt.account.LastModifiedOn, 10) &&
						*input.Item["AdminRoleArn"].S == string(*tt.account.AdminRoleArn) &&
						*input.ConditionExpression == "attribute_not_exists (#0)")
				}
				return (*input.TableName == "Accounts" &&
					*input.Item["Id"].S == *tt.account.ID &&
					*input.Item["AccountStatus"].S == string(*tt.account.Status) &&
					*input.Item["LastModifiedOn"].N == strconv.FormatInt(*tt.account.LastModifiedOn, 10) &&
					*input.Item["AdminRoleArn"].S == string(*tt.account.AdminRoleArn) &&
					*input.ExpressionAttributeValues[":0"].N == strconv.FormatInt(*tt.oldLastModifiedOn, 10))
			})).Return(
				&dynamodb.PutItemOutput{}, tt.dynamoErr,
			)
			accountData := &Account{
				DynamoDB:  &mockDynamo,
				TableName: "Accounts",
			}

			err := accountData.WriteAccount(&tt.account, tt.oldLastModifiedOn)
			if err != nil {
				log.Printf(err.Error())
				log.Printf(tt.expectedErr.Error())
			}
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}
