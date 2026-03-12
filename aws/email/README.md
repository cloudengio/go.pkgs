# Package [cloudeng.io/aws/email](https://pkg.go.dev/cloudeng.io/aws/email?tab=doc)

```go
import cloudeng.io/aws/email
```


## Types
### Type SendEmailAPI
```go
type SendEmailAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}
```
SendEmailAPI defines the interface for the SES V2 client used by Sender.


### Type Sender
```go
type Sender struct {
	// contains filtered or unexported fields
}
```
Sender provides a simplified interface for sending emails via AWS SES v2.

### Functions

```go
func NewSender(cfg aws.Config, from string) *Sender
```
NewSender creates a new email sender for the specified from address.


```go
func NewSenderFromAPI(api SendEmailAPI, from string) *Sender
```
NewSenderFromAPI creates a new email sender using the provided SendEmailAPI.
This is primarily useful for testing.



### Methods

```go
func (s *Sender) Send(ctx context.Context, to []string, subject, textBody, htmlBody string) error
```
Send sends an email to the specified recipients. It allows setting both a
text and HTML body. Either textBody or htmlBody can be empty, but not both.







