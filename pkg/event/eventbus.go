package event

import (
	"encoding/json"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents/cloudwatcheventsiface"
)

// CloudWatchEvent is for publishing events to Event Bus
type CloudWatchEvent struct {
	cw         cloudwatcheventsiface.CloudWatchEventsAPI
	eventBus   *string
	detailType *string
	source     *string
}

// Publish an event to the topic
func (c *CloudWatchEvent) Publish(i interface{}) error {
	bodyJSON, err := json.Marshal(i)
	if err != nil {
		return errors.NewInternalServer("unable to marshal response", err)
	}

	// Send the message
	_, err = c.cw.PutEvents(&cloudwatchevents.PutEventsInput{
		Entries: []*cloudwatchevents.PutEventsRequestEntry{
			{
				EventBusName: c.eventBus,
				Detail:       aws.String(string(bodyJSON)),
				DetailType:   c.detailType,
				Source:       c.source,
			},
		},
	})
	if err != nil {
		return errors.NewInternalServer("failed to publish message to CloudWatch Event Bus", err)
	}
	return nil
}

// NewCloudWatchEvent creates a new AWS Eventing Bus
func NewCloudWatchEvent(cw cloudwatcheventsiface.CloudWatchEventsAPI, eventBus string, detailType string, source string) (*CloudWatchEvent, error) {

	return &CloudWatchEvent{
		cw:         cw,
		eventBus:   &eventBus,
		detailType: &detailType,
		source:     &source,
	}, nil
}
