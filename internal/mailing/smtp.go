package mailing

import (
	"net"
	"net/mail"
	"net/smtp"
)

// SMTPSender sends emails using an SMTP service.
type SMTPSender struct {
	From mail.Address
	Addr string
	Auth smtp.Auth
}

// NewSMTPSender implementation using an SMTP server.
func NewSMTPSender(from, host, port, username, password string) *SMTPSender {
	return &SMTPSender{
		From: mail.Address{Address: from},
		Addr: net.JoinHostPort(host, port),
		Auth: smtp.PlainAuth("", username, password, host),
	}
}

// Send an email to the given email address.
func (s *SMTPSender) Send(to, subject, body string) error {
	toAddr := mail.Address{Address: to}
	msg := message(s.From, toAddr, subject, body)

	return smtp.SendMail(
		s.Addr,
		s.Auth,
		s.From.Address,
		[]string{toAddr.Address},
		[]byte(msg),
	)
}
