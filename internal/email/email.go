package email

import (
	"errors"

	"github.com/resend/resend-go/v2"
)

type Mailer struct {
	client   *resend.Client
	fromName string
	fromAddr string
}

func NewMailer(apiKey, fromName, fromAddr string) Mailer {
	client := resend.NewClient(apiKey)
	return Mailer{
		client:   client,
		fromName: fromName,
		fromAddr: fromAddr,
	}
}
func (m Mailer) Send(to, subject, htmlBody string) error {
	params := &resend.SendEmailRequest{
		From:    m.fromName + "<" + m.fromAddr + ">",
		To:      []string{to},
		Html:    htmlBody,
		Subject: subject,
	}

	sent, err := m.client.Emails.Send(params)
	if err != nil {
		return err
	}

	if sent.Id == "" {
		return errors.New("failed to send email via Resend")
	}

	return nil
}
