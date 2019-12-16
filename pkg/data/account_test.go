package data

import (
	gErrors "errors"
	"fmt"
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

	t.Run("should return an account object", func(t *testing.T) {
		mockDynamo := awsmocks.DynamoDBAPI{}

		accountID := "abc123"
		currentStatus := "Ready"

		mockDynamo.On("GetItem", &dynamodb.GetItemInput{
			ConsistentRead: aws.Bool(false),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			TableName: aws.String("Accounts"),
		}).Return(
			&dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"Id": {
						S: aws.String(accountID),
					},
					"Status": {
						S: aws.String(string(currentStatus)),
					},
				},
			}, nil,
		)
		accountData := &Account{
			AwsDynamoDB: &mockDynamo,
			TableName:   "Accounts",
		}
		item := &model.Account{}
		err := accountData.GetAccountByID(accountID, item)
		assert.NoError(t, err)
		assert.NotNil(t, item.ID, accountID)
	})

	t.Run("should return nil object when not found", func(t *testing.T) {
		mockDynamo := awsmocks.DynamoDBAPI{}

		accountID := "abc123"

		mockDynamo.On("GetItem", &dynamodb.GetItemInput{
			ConsistentRead: aws.Bool(false),
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			TableName: aws.String("Accounts"),
		}).Return(
			&dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			}, nil,
		)
		accountData := &Account{
			AwsDynamoDB: &mockDynamo,
			TableName:   "Accounts",
		}
		item := &model.Account{}
		err := accountData.GetAccountByID(accountID, item)
		assert.Error(t, err, "Account not found")
		assert.Nil(t, item.ID)
	})

}

func TestDelete(t *testing.T) {

	t.Run("should delete an account", func(t *testing.T) {
		mockDynamo := awsmocks.DynamoDBAPI{}

		accountID := "abc123"

		mockDynamo.On("DeleteItem", &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{
				"Id": {
					S: aws.String(accountID),
				},
			},
			TableName:    aws.String("Accounts"),
			ReturnValues: aws.String("ALL_NEW"),
		}).Return(
			&dynamodb.DeleteItemOutput{
				Attributes: map[string]*dynamodb.AttributeValue{
					"Id": {
						S: aws.String(accountID),
					},
					"AccountStatus": {
						S: aws.String("Ready"),
					},
				},
			}, nil,
		)
		accountData := &Account{
			AwsDynamoDB: &mockDynamo,
			TableName:   "Accounts",
		}
		item := &model.Account{
			ID: &accountID,
		}
		err := accountData.Delete(item)
		assert.NoError(t, err)
		assert.Equal(t, *item.ID, accountID)
		assert.Equal(t, *item.Status, model.Ready)
	})

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
			name: "normal",
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
			name: "nil",
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
			name: "normal",
			account: model.Account{
				ID:             ptrString("abc123"),
				Status:         &accountStatusReady,
				LastModifiedOn: ptrInt64(1573592058),
				AdminRoleArn:   ptrString("test:Arn"),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         awserr.New("ConditionalCheckFailedException", "Message", fmt.Errorf("Bad")),
			expectedErr:       errors.ErrConflict,
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
				AwsDynamoDB: &mockDynamo,
				TableName:   "Accounts",
			}

			err := accountData.Update(&tt.account, tt.oldLastModifiedOn)
			assert.True(t, gErrors.Is(err, tt.expectedErr))
		})
	}

}
