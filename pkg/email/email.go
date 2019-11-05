package email

import (
	"bytes"

	"github.com/Optum/dce/pkg/awsiface"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"gopkg.in/gomail.v2"
)

//go:generate mockery -name Service
type Service interface {
	SendEmail(input *SendEmailInput) error
	SendRawEmailWithAttachment(input *SendEmailWithAttachmentInput) error
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

type SendEmailWithAttachmentInput struct {
	FromArn            string
	FromAddress        string
	ToAddresses        []string
	CCAddresses        []string
	BCCAddresses       []string
	Subject            string
	BodyHTML           string
	BodyText           string
	AttachmentFileName string
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

// SendRawEmailWithAttachment sends SES raw email with attachment
func (svc *SESEmailService) SendRawEmailWithAttachment(input *SendEmailWithAttachmentInput) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", input.FromAddress)
	msg.SetHeader("To", input.ToAddresses[0])
	msg.SetHeader("Subject", input.Subject)
	msg.SetBody("text/html", input.BodyHTML)
	msg.Attach(input.AttachmentFileName)

	var emailRaw bytes.Buffer
	msg.WriteTo(&emailRaw)

	message := ses.RawMessage{Data: emailRaw.Bytes()}
	emailInput := &ses.SendRawEmailInput{
		FromArn:    aws.String(input.FromArn),
		RawMessage: &message,
	}

	_, err := svc.SES.SendRawEmail(emailInput)

	return err
}
