package email_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/aws/email"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

type mockSES struct {
	lastInput *sesv2.SendEmailInput
}

func (m *mockSES) SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	m.lastInput = params
	return &sesv2.SendEmailOutput{}, nil
}

func TestSender(t *testing.T) {
	ctx := context.Background()
	mock := &mockSES{}
	sender := email.NewSenderFromAPI(mock, "test@example.com")

	err := sender.Send(ctx, []string{"to@example.com"}, "Hello", "Text body", "HTML body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastInput == nil {
		t.Fatal("expected SendEmail to be called")
	}

	if got, want := *mock.lastInput.FromEmailAddress, "test@example.com"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := mock.lastInput.Destination.ToAddresses[0], "to@example.com"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := *mock.lastInput.Content.Simple.Subject.Data, "Hello"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := *mock.lastInput.Content.Simple.Body.Text.Data, "Text body"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := *mock.lastInput.Content.Simple.Body.Html.Data, "HTML body"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSenderTextOnly(t *testing.T) {
	ctx := context.Background()
	mock := &mockSES{}
	sender := email.NewSenderFromAPI(mock, "test@example.com")

	err := sender.Send(ctx, []string{"to@example.com"}, "Hello", "Text body", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastInput.Content.Simple.Body.Html != nil {
		t.Errorf("expected HTML body to be nil")
	}

	if got, want := *mock.lastInput.Content.Simple.Body.Text.Data, "Text body"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSenderHtmlOnly(t *testing.T) {
	ctx := context.Background()
	mock := &mockSES{}
	sender := email.NewSenderFromAPI(mock, "test@example.com")

	err := sender.Send(ctx, []string{"to@example.com"}, "Hello", "", "HTML body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastInput.Content.Simple.Body.Text != nil {
		t.Errorf("expected Text body to be nil")
	}

	if got, want := *mock.lastInput.Content.Simple.Body.Html.Data, "HTML body"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

var awsService *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsService, awstestutil.WithSES())
	os.Exit(0)
}

func TestSenderWithLocalStack(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	client := awsService.SESV2(awstestutil.DefaultAWSConfig())

	sender := email.NewSenderFromAPI(client, "test@example.com")
	err := sender.Send(ctx, []string{"to@example.com"}, "test subject", "test text body", "test html body")
	if err != nil {
		if !strings.Contains(err.Error(), "not yet implemented or pro feature") {
			t.Fatalf("unexpected error from LocalStack: %v", err)
		}
		t.Log("LocalStack returned expected 'not implemented' for SESv2")
	}
}
