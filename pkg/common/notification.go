package common

import (
	"github.com/aws/aws-sdk-go/service/sns"
)

// Notification interface requires methods to interact with an AWS SNS Topic
// and publish a message to it
type Notification interface {
	Publish(*sns.PublishInput) (*sns.PublishOutput, error)
}

// SNS implements the Notification interface with AWS SNS SDK
type SNS struct {
	Client *sns.SNS
}

// Publish pushes the provided message to the provided SNS Topic
func (notif SNS) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	return notif.Client.Publish(input)
}
