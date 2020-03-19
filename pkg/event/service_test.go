package event

import (
	"errors"
	"testing"

	"github.com/Optum/dce/pkg/account"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/event/mocks"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {

	t.Run("New Eventer", func(t *testing.T) {
		mockSns := &awsMocks.SNSAPI{}
		mockSqs := &awsMocks.SQSAPI{}
		mockCwe := &awsMocks.CloudWatchEventsAPI{}

		accountCreatedTopicArn, _ := arn.Parse("arn:aws:sns:us-east-1:123456789012:createAccount")
		accountDeletedTopicArn, _ := arn.Parse("arn:aws:sns:us-east-1:123456789012:deleteAccount")
		leaseAddedTopicArn, _ := arn.Parse("arn:aws:sns:us-east-1:123456789012:createLease")
		accountResetQueueURL := "http://sqs.com/queue"

		eventer, err := NewService(NewServiceInput{
			SnsClient:              mockSns,
			SqsClient:              mockSqs,
			CweClient:              mockCwe,
			AccountCreatedTopicArn: accountCreatedTopicArn.String(),
			AccountDeletedTopicArn: accountDeletedTopicArn.String(),
			LeaseAddedTopicArn:     leaseAddedTopicArn.String(),
			AccountResetQueueURL:   accountResetQueueURL,
		})

		assert.Nil(t, err)
		assert.Equal(t, []Publisher{
			&SnsEvent{
				sns:      mockSns,
				topicArn: accountCreatedTopicArn,
			},
			&CloudWatchEvent{
				cw:         mockCwe,
				detailType: aws.String("AccountCreated"),
			},
		}, eventer.accountCreate)
		assert.Equal(t, []Publisher{
			&SnsEvent{
				sns:      mockSns,
				topicArn: accountDeletedTopicArn,
			},
			&CloudWatchEvent{
				cw:         mockCwe,
				detailType: aws.String("AccountDeleted"),
			},
		}, eventer.accountDelete)

		assert.Equal(t, []Publisher{
			&CloudWatchEvent{
				cw:         mockCwe,
				detailType: aws.String("AccountUpdated"),
			},
		}, eventer.accountUpdate)
		assert.Equal(t, []Publisher{
			&SnsEvent{
				sns:      mockSns,
				topicArn: leaseAddedTopicArn,
			},
			&CloudWatchEvent{
				cw:         mockCwe,
				detailType: aws.String("LeaseCreated"),
			},
		}, eventer.leaseCreate)
		assert.Equal(t, []Publisher{
			&CloudWatchEvent{
				cw:         mockCwe,
				detailType: aws.String("LeaseUpdated"),
			},
		}, eventer.leaseUpdate)
		assert.Equal(t, []Publisher{
			&CloudWatchEvent{
				cw:         mockCwe,
				detailType: aws.String("LeaseEnded"),
			},
		}, eventer.leaseEnd)
	})

}

func TestEventAccountPublishers(t *testing.T) {

	tests := []struct {
		name                            string
		event                           *account.Account
		eventOld                        *account.Account
		expectedAccountCreatePublishErr error
		expectedAccountDeletePublishErr error
		expectedAccountUpdatePublishErr error
		expectedAccountResetPublishErr  error
	}{
		{
			name: "publish events",
			event: &account.Account{
				Status: account.StatusReady.StatusPtr(),
			},
			eventOld: &account.Account{
				Status: account.StatusNotReady.StatusPtr(),
			},
			expectedAccountCreatePublishErr: nil,
			expectedAccountDeletePublishErr: nil,
			expectedAccountUpdatePublishErr: nil,
			expectedAccountResetPublishErr:  nil,
		},
		{
			name: "publish event with errors",
			event: &account.Account{
				Status: account.StatusReady.StatusPtr(),
			},
			expectedAccountCreatePublishErr: errors.New("failure"),
			expectedAccountDeletePublishErr: errors.New("failure"),
			expectedAccountUpdatePublishErr: errors.New("failure"),
			expectedAccountResetPublishErr:  errors.New("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCreateAccountPublisher := mocks.Publisher{}
			mockCreateAccountPublisher.On("Publish", tt.event).Return(tt.expectedAccountCreatePublishErr)
			mockDeleteAccountPublisher := mocks.Publisher{}
			mockDeleteAccountPublisher.On("Publish", tt.event).Return(tt.expectedAccountDeletePublishErr)
			mockUpdateAccountPublisher := mocks.Publisher{}
			mockUpdateAccountPublisher.On("Publish", updateEvent{
				Old: tt.eventOld,
				New: tt.event,
			}).Return(tt.expectedAccountUpdatePublishErr)
			mockResetAccountPublisher := mocks.Publisher{}
			mockResetAccountPublisher.On("Publish", tt.event).Return(tt.expectedAccountResetPublishErr)

			eventSvc := Service{
				accountCreate: []Publisher{&mockCreateAccountPublisher},
				accountDelete: []Publisher{&mockDeleteAccountPublisher},
				accountUpdate: []Publisher{&mockUpdateAccountPublisher},
				accountReset:  []Publisher{&mockResetAccountPublisher},
			}

			var err error
			err = eventSvc.AccountCreate(tt.event)
			assert.Equal(t, tt.expectedAccountCreatePublishErr, err)
			mockCreateAccountPublisher.AssertExpectations(t)

			err = eventSvc.AccountDelete(tt.event)
			assert.Equal(t, tt.expectedAccountDeletePublishErr, err)
			mockDeleteAccountPublisher.AssertExpectations(t)

			err = eventSvc.AccountUpdate(tt.eventOld, tt.event)
			assert.Equal(t, tt.expectedAccountUpdatePublishErr, err)
			mockUpdateAccountPublisher.AssertExpectations(t)

			err = eventSvc.AccountReset(tt.event)
			assert.Equal(t, tt.expectedAccountResetPublishErr, err)
			mockResetAccountPublisher.AssertExpectations(t)

		})
	}

}

func TestEventLeasePublishers(t *testing.T) {

	tests := []struct {
		name                          string
		event                         *lease.Lease
		eventOld                      *lease.Lease
		expectedLeaseCreatePublishErr error
		expectedLeaseEndPublishErr    error
		expectedLeaseUpdatePublishErr error
	}{
		{
			name: "publish events",
			event: &lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			eventOld: &lease.Lease{
				Status: lease.StatusInactive.StatusPtr(),
			},
			expectedLeaseCreatePublishErr: nil,
			expectedLeaseEndPublishErr:    nil,
			expectedLeaseUpdatePublishErr: nil,
		},
		{
			name: "publish event with errors",
			event: &lease.Lease{
				Status: lease.StatusActive.StatusPtr(),
			},
			eventOld: &lease.Lease{
				Status: lease.StatusInactive.StatusPtr(),
			},
			expectedLeaseCreatePublishErr: errors.New("failure"),
			expectedLeaseEndPublishErr:    errors.New("failure"),
			expectedLeaseUpdatePublishErr: errors.New("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLeaseCreatedPublisher := mocks.Publisher{}
			mockLeaseCreatedPublisher.On("Publish", tt.event).Return(tt.expectedLeaseCreatePublishErr)
			mockLeaseEndedPublisher := mocks.Publisher{}
			mockLeaseEndedPublisher.On("Publish", tt.event).Return(tt.expectedLeaseEndPublishErr)
			mockLeaseUpdatedPublisher := mocks.Publisher{}
			mockLeaseUpdatedPublisher.On("Publish", updateEvent{
				Old: tt.eventOld,
				New: tt.event,
			}).Return(tt.expectedLeaseUpdatePublishErr)

			eventSvc := Service{
				leaseCreate: []Publisher{&mockLeaseCreatedPublisher},
				leaseEnd:    []Publisher{&mockLeaseEndedPublisher},
				leaseUpdate: []Publisher{&mockLeaseUpdatedPublisher},
			}

			var err error
			err = eventSvc.LeaseCreate(tt.event)
			assert.Equal(t, tt.expectedLeaseCreatePublishErr, err)
			mockLeaseCreatedPublisher.AssertExpectations(t)

			err = eventSvc.LeaseEnd(tt.event)
			assert.Equal(t, tt.expectedLeaseEndPublishErr, err)
			mockLeaseEndedPublisher.AssertExpectations(t)

			err = eventSvc.LeaseUpdate(tt.eventOld, tt.event)
			assert.Equal(t, tt.expectedLeaseUpdatePublishErr, err)
			mockLeaseUpdatedPublisher.AssertExpectations(t)
		})
	}

}

func TestPublishingWithRange(t *testing.T) {

	type data struct {
		Key string `json:"key"`
	}

	tests := []struct {
		name       string
		event      interface{}
		returnErr1 error
		returnErr2 error
		returnErr3 error
		expErr     error
	}{
		{
			name: "publish events",
			event: data{
				Key: "value",
			},
			returnErr1: nil,
			returnErr2: nil,
			returnErr3: nil,
			expErr:     nil,
		},
		{
			name: "publish events with error on 1",
			event: data{
				Key: "value",
			},
			returnErr1: errors.New("failure"),
			returnErr2: nil,
			returnErr3: nil,
			expErr:     errors.New("failure"),
		},
		{
			name: "publish events with error on 3",
			event: data{
				Key: "value",
			},
			returnErr1: nil,
			returnErr2: nil,
			returnErr3: errors.New("failure"),
			expErr:     errors.New("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPublisher1 := mocks.Publisher{}
			mockPublisher1.On("Publish", tt.event).Return(tt.returnErr1)
			mockPublisher2 := mocks.Publisher{}
			mockPublisher2.On("Publish", tt.event).Return(tt.returnErr2)
			mockPublisher3 := mocks.Publisher{}
			mockPublisher3.On("Publish", tt.event).Return(tt.returnErr3)

			eventSvc := Service{}

			publishers := []Publisher{
				&mockPublisher1,
				&mockPublisher2,
				&mockPublisher3,
			}

			err := eventSvc.publish(tt.event, publishers...)
			assert.Equal(t, tt.expErr, err)
		})
	}

}
