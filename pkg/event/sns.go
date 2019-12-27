package event

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

// SnsEvent is for publishing events to SQS
type SnsEvent struct {
	sns      snsiface.SNSAPI
	topicArn arn.ARN
}

// Publish an event to the topic
func (s *SnsEvent) Publish(i interface{}) error {
	bodyJSON, err := json.Marshal(i)
	if err != nil {
		return errors.New("Unable to marshal response")
	}

	// Wrap the body in a SNS message object
	message, err := json.Marshal(struct {
		Default string `json:"default"`
		Body    string `json:"Body"`
	}{
		Default: string(bodyJSON),
		Body:    string(bodyJSON),
	})
	if err != nil {
		return fmt.Errorf("Failed to prepare SNS body JSON: %w", err)
	}

	// Send the message
	_, err = s.sns.Publish(&sns.PublishInput{
		Message:          aws.String(string(message)),
		TopicArn:         aws.String(s.topicArn.String()),
		MessageStructure: aws.String("json"),
	})
	return err
}

// NewSnsEvent creates a new SNS eventing struct
func NewSnsEvent(sns snsiface.SNSAPI, a string) (*SnsEvent, error) {

	snsArn, err := arn.Parse(a)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse arn %q: %w", a, err)
	}
	return &SnsEvent{
		sns:      sns,
		topicArn: snsArn,
	}, nil
}
