package event

import (
	"errors"
	"math"
	"testing"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/require"
)

func TestSqs(t *testing.T) {

	type data struct {
		Key string `json:"key"`
	}

	tests := []struct {
		name                string
		sqsErr              error
		event               interface{}
		expectedErr         error
		expectedMessageBody string
	}{
		{
			name:   "publish sqs event",
			sqsErr: nil,
			event: data{
				Key: "value",
			},
			expectedMessageBody: "{\"key\":\"value\"}",
			expectedErr:         nil,
		},
		{
			name:   "publish sqs error",
			sqsErr: errors.New("error"),
			event: data{
				Key: "value",
			},
			expectedMessageBody: "{\"key\":\"value\"}",
			expectedErr:         errors.New("error"),
		},
		{
			name:        "unmarshal error",
			sqsErr:      nil,
			event:       math.Inf(1),
			expectedErr: errors.New("Unable to marshal response"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSqs := &mocks.SQSAPI{}
			eventer, _ := NewSqsEvent(mockSqs, "http://url.com")

			// Mock StartBuild call
			mockSqs.On("SendMessage",
				&sqs.SendMessageInput{
					MessageBody: aws.String(tt.expectedMessageBody),
					QueueUrl:    aws.String("http://url.com"),
				},
			).Return(nil, tt.sqsErr)

			err := eventer.Publish(tt.event)
			if tt.expectedErr == tt.sqsErr {
				mockSqs.AssertExpectations(t)
			}
			require.Equal(t, tt.expectedErr, err)

		})
	}

}
