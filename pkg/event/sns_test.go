package event

import (
	gErrors "errors"
	"math"
	"testing"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
)

func TestSns(t *testing.T) {

	type data struct {
		Key string `json:"key"`
	}

	tests := []struct {
		name            string
		snsErr          error
		event           interface{}
		expectedErr     error
		expectedMessage string
	}{
		{
			name:   "publish sns event",
			snsErr: nil,
			event: data{
				Key: "value",
			},
			expectedMessage: "{\"Body\":\"{\\\"key\\\":\\\"value\\\"}\",\"default\":\"{\\\"key\\\":\\\"value\\\"}\"}",
			expectedErr:     nil,
		},
		{
			name:   "publish sns error",
			snsErr: gErrors.New("error"),
			event: data{
				Key: "value",
			},
			expectedMessage: "{\"Body\":\"{\\\"key\\\":\\\"value\\\"}\",\"default\":\"{\\\"key\\\":\\\"value\\\"}\"}",
			expectedErr:     errors.NewInternalServer("failed to publish message to SNS topic", nil),
		},
		{
			name:        "unmarshal error",
			snsErr:      nil,
			event:       math.Inf(1),
			expectedErr: errors.NewInternalServer("unable to marshal response", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSns := &mocks.SNSAPI{}
			eventer, _ := NewSnsEvent(mockSns, "arn:aws:sns:us-east-1:123456789012:test")

			// Mock StartBuild call
			mockSns.On("Publish",
				&sns.PublishInput{
					Message:          aws.String(tt.expectedMessage),
					TopicArn:         aws.String("arn:aws:sns:us-east-1:123456789012:test"),
					MessageStructure: aws.String("json"),
				},
			).Return(nil, tt.snsErr)

			err := eventer.Publish(tt.event)
			if tt.expectedErr == tt.snsErr {
				mockSns.AssertExpectations(t)
			}

			if err != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.Nil(t, tt.expectedErr)
			}

		})
	}

}
