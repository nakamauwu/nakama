//go:generate moq -out sender_mock.go . Sender

package mailing

import (
	"fmt"
	"net/mail"
)

// Sender sends mails.
type Sender interface {
	Send(to, subject, body string) error
}

func message(from, to mail.Address, subject, body string) string {
	return fmt.Sprintf("From: %s\r\n", from.String()) +
		fmt.Sprintf("To: %s\r\n", to.String()) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		body
}
