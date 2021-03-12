package mailing

import (
	"fmt"
	"net/mail"

	"github.com/sendgrid/sendgrid-go"
	sendgridmail "github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendgridSender struct {
	From   mail.Address
	APIKey string
}

func NewSendgridSender(from, apiKey string) *SendgridSender {
	return &SendgridSender{
		From:   mail.Address{Name: "nakama", Address: from},
		APIKey: apiKey,
	}
}

func (s *SendgridSender) Send(to, subject, html, text string) error {
	m := sendgridmail.NewSingleEmail(
		sendgridmail.NewEmail(s.From.Name, s.From.Address),
		subject,
		sendgridmail.NewEmail("", to),
		text,
		html,
	)
	c := sendgrid.NewSendClient(s.APIKey)
	c.Body = sendgridmail.GetRequestBody(m)
	resp, err := sendgrid.MakeRequestRetry(c.Request)
	if err != nil {
		return fmt.Errorf("could not send mail: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("could not send mail; got status_code=%d headers=%+v body=%v", resp.StatusCode, resp.Headers, resp.Body)
	}

	return nil
}
