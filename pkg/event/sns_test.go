package event

import (
	"errors"
	"math"
	"testing"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/require"
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
			expectedMessage: "{\"default\":\"{\\\"key\\\":\\\"value\\\"}\",\"Body\":\"{\\\"key\\\":\\\"value\\\"}\"}",
			expectedErr:     nil,
		},
		{
			name:   "publish sns error",
			snsErr: errors.New("error"),
			event: data{
				Key: "value",
			},
			expectedMessage: "{\"default\":\"{\\\"key\\\":\\\"value\\\"}\",\"Body\":\"{\\\"key\\\":\\\"value\\\"}\"}",
			expectedErr:     errors.New("error"),
		},
		{
			name:        "unmarshal error",
			snsErr:      nil,
			event:       math.Inf(1),
			expectedErr: errors.New("Unable to marshal response"),
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
			require.Equal(t, tt.expectedErr, err)

		})
	}

}
