package event

import (
	gErrors "errors"
	"math"
	"testing"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/stretchr/testify/assert"
)

func TestCwe(t *testing.T) {

	type data struct {
		Key string `json:"key"`
	}

	tests := []struct {
		name            string
		cweErr          error
		event           interface{}
		expectedErr     error
		expectedMessage string
	}{
		{
			name:   "publish CWE event",
			cweErr: nil,
			event: data{
				Key: "value",
			},
			expectedMessage: "{\"key\":\"value\"}",
			expectedErr:     nil,
		},
		{
			name:   "publish CWE error",
			cweErr: gErrors.New("error"),
			event: data{
				Key: "value",
			},
			expectedMessage: "{\"key\":\"value\"}",
			expectedErr:     errors.NewInternalServer("failed to publish message to CloudWatch Event Bus", nil),
		},
		{
			name:        "unmarshal error",
			cweErr:      nil,
			event:       math.Inf(1),
			expectedErr: errors.NewInternalServer("unable to marshal response", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCwe := &mocks.CloudWatchEventsAPI{}
			eventer, _ := NewCloudWatchEvent(mockCwe, "emit")

			// Mock Publish call
			mockCwe.On("PutEvents",
				&cloudwatchevents.PutEventsInput{
					Entries: []*cloudwatchevents.PutEventsRequestEntry{
						{
							Detail:     &tt.expectedMessage,
							DetailType: aws.String("emit"),
							Source:     aws.String("dce"),
						},
					},
				},
			).Return(nil, tt.cweErr)

			err := eventer.Publish(tt.event)
			if tt.expectedErr == tt.cweErr {
				mockCwe.AssertExpectations(t)
			}

			if err != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.Nil(t, tt.expectedErr)
			}

		})
	}

}
