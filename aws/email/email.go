package email

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SendEmailAPI defines the interface for the SES V2 client used by Sender.
type SendEmailAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// Sender provides a simplified interface for sending emails via AWS SES v2.
type Sender struct {
	client SendEmailAPI
	from   string
}

// NewSender creates a new email sender for the specified from address.
func NewSender(cfg aws.Config, from string) *Sender {
	return &Sender{
		client: sesv2.NewFromConfig(cfg),
		from:   from,
	}
}

// NewSenderFromAPI creates a new email sender using the provided SendEmailAPI.
// This is primarily useful for testing.
func NewSenderFromAPI(api SendEmailAPI, from string) *Sender {
	return &Sender{
		client: api,
		from:   from,
	}
}

// Send sends an email to the specified recipients.
// It allows setting both a text and HTML body. Either textBody or htmlBody can be empty,
// but not both.
func (s *Sender) Send(ctx context.Context, to []string, subject, textBody, htmlBody string) error {
	var body *types.Body
	if textBody != "" && htmlBody != "" {
		body = &types.Body{
			Text: &types.Content{
				Data: aws.String(textBody),
			},
			Html: &types.Content{
				Data: aws.String(htmlBody),
			},
		}
	} else if textBody != "" {
		body = &types.Body{
			Text: &types.Content{
				Data: aws.String(textBody),
			},
		}
	} else if htmlBody != "" {
		body = &types.Body{
			Html: &types.Content{
				Data: aws.String(htmlBody),
			},
		}
	}

	content := &types.EmailContent{
		Simple: &types.Message{
			Body: body,
			Subject: &types.Content{
				Data: aws.String(subject),
			},
		},
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.from),
		Destination: &types.Destination{
			ToAddresses: to,
		},
		Content: content,
	}

	_, err := s.client.SendEmail(ctx, input)
	return err
}
