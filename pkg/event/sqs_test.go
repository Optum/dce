package event

import (
	gErrors "errors"
	"math"
	"testing"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
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
			sqsErr: gErrors.New("error"),
			event: data{
				Key: "value",
			},
			expectedMessageBody: "{\"key\":\"value\"}",
			expectedErr:         errors.NewInternalServer("unable to send message to sqs", nil),
		},
		{
			name:        "unmarshal error",
			sqsErr:      nil,
			event:       math.Inf(1),
			expectedErr: errors.NewInternalServer("unable to marshal response", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSqs := &mocks.SQSAPI{}
			eventer, _ := NewSqsEvent(mockSqs, "http://url.com")

			// Mock Publish call
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

			if err != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.Nil(t, tt.expectedErr)
			}

		})
	}

}
