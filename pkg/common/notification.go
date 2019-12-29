package common

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pkg/errors"
)

// Notificationer interface requires methods to interact with an AWS SNS Topic
// and publish a message to it
type Notificationer interface {
	PublishMessage(topicArn *string, message *string, isJSON bool) (*string, error)
}

// SNS implements the Notification interface with AWS SNS SDK
type SNS struct {
	Client *sns.SNS
}

// PublishMessage pushes the provided messeage to an SNS Topic and returns the
// messages' ID.
func (notif *SNS) PublishMessage(topicArn *string, message *string,
	isJSON bool) (*string, error) {
	// Create the SNS PublishInput
	var publishInput *sns.PublishInput
	if isJSON {
		messageStructure := "json"
		publishInput = &sns.PublishInput{
			TopicArn:         topicArn,
			Message:          message,
			MessageStructure: &messageStructure,
		}
	} else {
		publishInput = &sns.PublishInput{
			TopicArn: topicArn,
			Message:  message,
		}
	}

	// Publish the Message to the Topic
	publishOutput, err := notif.Client.Publish(publishInput)
	if err != nil {
		return nil, err
	}

	return publishOutput.MessageId, err
}

// PrepareSNSMessageJSON creates a JSON message
// from a struct, for publising to an SNS topic
func PrepareSNSMessageJSON(body interface{}) (string, error) {
	// Marshal the message body as JSON
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", errors.Wrap(err, "Failed to prepare SNS body JSON")
	}

	// Wrap the body in a SNS message object
	messageJSON, err := json.Marshal(struct {
		Default string `json:"default"`
		Body    string `json:"Body"`
	}{
		Default: string(bodyJSON),
		Body:    string(bodyJSON),
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to prepare SNS body JSON")
	}

	return string(messageJSON), nil
}

// CreateJSONPublishInput creates a `sns.PublishInput`
func CreateJSONPublishInput(topicArn *string, message *string) *sns.PublishInput {
	var publishInput *sns.PublishInput
	messageStructure := "json"
	publishInput = &sns.PublishInput{
		TopicArn:         topicArn,
		Message:          message,
		MessageStructure: &messageStructure,
	}
	return publishInput
}
