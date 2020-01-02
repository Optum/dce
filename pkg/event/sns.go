package event

import (
	"encoding/json"
	"fmt"

	"github.com/Optum/dce/pkg/errors"
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
		return errors.NewInternalServer("unable to marshal response", err)
	}

	// Wrap the body in a SNS message object
	message, err := json.Marshal(map[string]string{
		"default": string(bodyJSON),
		"Body":    string(bodyJSON),
	})
	if err != nil {
		return errors.NewInternalServer("failed to prepare SNS body JSON", err)
	}

	// Send the message
	_, err = s.sns.Publish(&sns.PublishInput{
		Message:          aws.String(string(message)),
		TopicArn:         aws.String(s.topicArn.String()),
		MessageStructure: aws.String("json"),
	})
	if err != nil {
		return errors.NewInternalServer("failed to publish message to SNS topic", err)
	}
	return nil
}

// NewSnsEvent creates a new SNS eventing struct
func NewSnsEvent(sns snsiface.SNSAPI, a string) (*SnsEvent, error) {

	snsArn, err := arn.Parse(a)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("unable to parse arn %q", a),
			err,
		)
	}
	return &SnsEvent{
		sns:      sns,
		topicArn: snsArn,
	}, nil
}
