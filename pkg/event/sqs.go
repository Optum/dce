package event

import (
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// SqsEvent is for publishing events to SQS
type SqsEvent struct {
	sqs sqsiface.SQSAPI
	url string
}

// Publish an event to the topic
func (s *SqsEvent) Publish(i interface{}) error {
	bodyJSON, err := json.Marshal(i)
	if err != nil {
		return errors.New("Unable to marshal response")
	}

	// Create the input
	input := sqs.SendMessageInput{
		QueueUrl:    aws.String(s.url),
		MessageBody: aws.String(string(bodyJSON)),
	}

	// Send the message
	_, err = s.sqs.SendMessage(&input)
	return err
}

// NewSqsEvent creates a new SQS eventing struct
func NewSqsEvent(sqs sqsiface.SQSAPI, url string) (*SqsEvent, error) {

	return &SqsEvent{
		sqs: sqs,
		url: url,
	}, nil
}
