package mailing

import (
	"io/ioutil"
	"log"
	"net/mail"
)

// LogSender log emails.
type LogSender struct {
	From   mail.Address
	Logger *log.Logger
}

// NewLogSender implementation using the provided logger.
func NewLogSender(from string, l *log.Logger) *LogSender {
	if l == nil {
		l = log.New(ioutil.Discard, "", 0)
	}
	return &LogSender{
		From:   mail.Address{Address: from},
		Logger: l,
	}
}

// Send will just log the email.
func (s *LogSender) Send(to, subject, body string) error {
	toAddr := mail.Address{Address: to}
	msg := message(s.From, toAddr, subject, body)
	s.Logger.Printf("\n\n%s\n\n", msg)

	return nil
}
