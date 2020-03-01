package data

import (
	gErrors "errors"
	"strconv"
	"testing"
	"time"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetUsagePrincipal(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name          string
		principalID   string
		date          time.Time
		dynamoErr     error
		dynamoOutput  *dynamodb.GetItemOutput
		expectedErr   error
		expectedUsage *usage.Principal
	}{
		{
			name:        "should return an usage principal object",
			principalID: "id-1",
			date:        now,
			expectedUsage: &usage.Principal{
				PrincipalID:  ptrString("user"),
				Date:         &now,
				CostAmount:   ptrFloat64(1),
				CostCurrency: ptrString("USD"),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"PrincipalId": {
						S: aws.String("user"),
					},
					"Date": {
						N: aws.String(strconv.FormatInt(now.Unix(), 10)),
					},
					"CostAmount": {
						N: aws.String("1"),
					},
					"CostCurrency": {
						S: aws.String("USD"),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:          "should return nil object when not found",
			principalID:   "id-1",
			date:          now,
			expectedUsage: nil,
			dynamoErr:     nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewNotFound("usage", "id-1"),
		},
		{
			name:          "should return nil when dynamodb err",
			principalID:   "id-1",
			date:          now,
			expectedUsage: nil,
			dynamoErr:     gErrors.New("failure"),
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewInternalServer("get failed for usage \"id-1\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("GetItem", mock.AnythingOfType("*dynamodb.GetItemInput")).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			usageSvc := &UsagePrincipal{
				DynamoDB:  &mockDynamo,
				TableName: "Usage",
			}

			result, err := usageSvc.Get(tt.principalID, tt.date)

			assert.Equal(t, tt.expectedUsage, result)
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}
