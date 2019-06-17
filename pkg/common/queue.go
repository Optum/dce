package common

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Queue interface requires a method to receive an SQS Message Output based on
// the provided SQS Message Input
type Queue interface {
	SendMessage(*string, *string) error
	ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	NewFromEnv() error
}

// SQSQueue implments the Queue interface using the AWS SQS Service
type SQSQueue struct {
	Client *sqs.SQS
}

// SendMessage sends the provided message to the queue using the AWS SQS Service
// and returns an errors
func (queue SQSQueue) SendMessage(queueURL *string, message *string) error {
	// Create the input
	input := sqs.SendMessageInput{
		QueueUrl:    queueURL,
		MessageBody: message,
	}

	// Send the message
	_, err := queue.Client.SendMessage(&input)
	return err
}

// ReceiveMessage method returns an AWS SQS Message Output based on the provided
// Message Input through the AWS SQS Client.
func (queue SQSQueue) ReceiveMessage(input *sqs.ReceiveMessageInput) (
	*sqs.ReceiveMessageOutput, error) {
	return queue.Client.ReceiveMessage(input)
}

// DeleteMessage method returns an AWS SQS Delete Message Output based on the
// provided Delete Message Input through the SQS Client
func (queue SQSQueue) DeleteMessage(input *sqs.DeleteMessageInput) (
	*sqs.DeleteMessageOutput, error) {
	return queue.Client.DeleteMessage(input)
}

// NewFromEnv creates an SQS instance configured from environment variables.
// Requires env vars for:
// - AWS_CURRENT_REGION
func (queue SQSQueue) NewFromEnv() error {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(RequireEnv("AWS_CURRENT_REGION"))})
	if err != nil {
		return err
	}
	queue.Client = sqs.New(awsSession)
	if err != nil {
		return err
	}
	return nil
}
