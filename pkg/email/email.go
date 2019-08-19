package email

import (
	"github.com/Optum/Redbox/pkg/awsiface"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
)

//go:generate mockery -name Service
type Service interface {
	SendEmail(input *SendEmailInput) error
}

type SendEmailInput struct {
	FromAddress  string
	ToAddresses  []string
	CCAddresses  []string
	BCCAddresses []string
	Subject      string
	BodyHTML     string
	BodyText     string
}

type SESEmailService struct {
	SES awsiface.SESAPI
}

func (svc *SESEmailService) SendEmail(input *SendEmailInput) error {
	emailInput := ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses:  aws.StringSlice(input.ToAddresses),
			CcAddresses:  aws.StringSlice(input.CCAddresses),
			BccAddresses: aws.StringSlice(input.BCCAddresses),
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(input.BodyHTML),
				},
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(input.BodyText),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(input.Subject),
			},
		},
		Source: aws.String(input.FromAddress),
	}
	_, err := svc.SES.SendEmail(&emailInput)

	return err
}
