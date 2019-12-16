package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/pkg/errors"
)

// Event - Data Layer Struct
type Event struct {
	AwsSns          snsiface.SNSAPI
	CreatedTopicArn string
	DeletedTopicArn string
}

type PublishInput struct {
	message interface{}
	source  string
	name    string
}

func (a *Event) PublishJSON(i PublishInput) error {
	serializedMessage, err := prepareSnsMessageJSON(i.message)

	if err != nil {
		log.Printf("Failed to serialized SNS message: %s", err)
		return err
	}

	topicArn := a.pickTopicArn(i.source, i.name)

	_, err := a.publish(&sns.PublishInput{
		TopicArn:         topicArn,
		Message:          serializedMessage,
		MessageStructure: aws.String("json"),
	})

	return err
}

func (a *Event) pickTopicArn(source string, name string) (*string, error) {
	switch source {
	case "account":
		switch name {
		case "created":
			return aws.String(a.CreatedTopicArn), nil
		case "deleted":
			return aws.String(a.DeletedTopicArn), nil
		}
		return nil, fmt.Errorf("No SNS Topic ARN found for source %s and name %s", source, name)
	}
}

func (a *Event) publish(m *sns.PublishInput) (*string, error) {

	// Publish the Message to the Topic
	publishOutput, err := a.AwsSns.Publish(m)
	if err != nil {
		return nil, err
	}

	return publishOutput.MessageId, err
}

// prepareSnsMessageJSON creates a JSON message
// from a struct, for publising to an SNS topic
func prepareSnsMessageJSON(body interface{}) (string, error) {
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
