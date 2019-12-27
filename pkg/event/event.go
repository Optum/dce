package event

import (
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// Publisher interface defines anything that can publish an event
type Publisher interface {
	Publish(i interface{}) error
}

// NewEventerInput are the items required to create a new Eventer service
type NewEventerInput struct {
	SnsClient              snsiface.SNSAPI
	SqsClient              sqsiface.SQSAPI
	AccountCreatedTopicArn string `env:"ACCOUNT_CREATED_TOPIC_ARN" defaultEnv:"DefaultAccountCreatedTopicArn"`
	AccountDeletedTopicArn string `env:"ACCOUNT_DELETED_TOPIC_ARN" defaultEnv:"DefaultAccountDeletedTopicArn"`
	AccountResetQueueURL   string `env:"RESET_SQS_URL" defaultEnv:"DefaultResetSQSUrl"`
	LeaseAddedTopicArn     string `env:"LEASE_ADDED_TOPIC" defaultEnv:"DefaultLeaseAddedTopicArn"`
}

// Eventer is the public interface for publishing events
type Eventer struct {
	accountCreate []Publisher
	accountDelete []Publisher
	accountUpdate []Publisher
	accountReset  []Publisher
	leaseCreate   []Publisher
	leaseEnd      []Publisher
	leaseUpdate   []Publisher
}

func (e *Eventer) publish(i interface{}, p ...Publisher) error {
	for _, n := range p {
		err := n.Publish(i)
		if err != nil {
			return err
		}
	}
	return nil
}

// AccountCreate publish events
func (e *Eventer) AccountCreate(i interface{}) error {
	return e.publish(i, e.accountCreate...)
}

// AccountDelete publish events
func (e *Eventer) AccountDelete(i interface{}) error {
	return e.publish(i, e.accountDelete...)
}

// AccountUpdate publish events
func (e *Eventer) AccountUpdate(i interface{}) error {
	return e.publish(i, e.accountUpdate...)
}

// AccountReset publish events
func (e *Eventer) AccountReset(i interface{}) error {
	return e.publish(i, e.accountReset...)
}

// LeaseCreate publish events
func (e *Eventer) LeaseCreate(i interface{}) error {
	return e.publish(i, e.leaseCreate...)
}

// LeaseEnd publish events
func (e *Eventer) LeaseEnd(i interface{}) error {
	return e.publish(i, e.leaseEnd...)
}

// LeaseUpdate publish events
func (e *Eventer) LeaseUpdate(i interface{}) error {
	return e.publish(i, e.leaseUpdate...)
}

// NewEventer creates a new instance of Eventer
func NewEventer(input NewEventerInput) (*Eventer, error) {
	newEventer := &Eventer{}

	//////////////////////////////////////////////////////////////////////
	// Account Eventing
	//////////////////////////////////////////////////////////////////////
	createAccount, err := NewSnsEvent(input.SnsClient, input.AccountCreatedTopicArn)
	if err != nil {
		return nil, err
	}

	resetAccount, err := NewSqsEvent(input.SqsClient, input.AccountResetQueueURL)
	if err != nil {
		return nil, err
	}

	deleteAccount, err := NewSnsEvent(input.SnsClient, input.AccountDeletedTopicArn)
	if err != nil {
		return nil, err
	}
	newEventer.accountCreate = []Publisher{
		createAccount,
		resetAccount,
	}
	newEventer.accountReset = []Publisher{
		resetAccount,
	}
	newEventer.accountDelete = []Publisher{
		deleteAccount,
		resetAccount,
	}
	newEventer.accountUpdate = []Publisher{}

	//////////////////////////////////////////////////////////////////////
	// Lease Eventing
	//////////////////////////////////////////////////////////////////////
	createLease, err := NewSnsEvent(input.SnsClient, input.LeaseAddedTopicArn)
	if err != nil {
		return nil, err
	}

	newEventer.leaseCreate = []Publisher{
		createLease,
	}
	newEventer.leaseEnd = []Publisher{
		resetAccount,
	}
	newEventer.leaseUpdate = []Publisher{}

	return newEventer, nil
}
