package mailing

import (
	"net/mail"
)

// Logger interface.
type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

// LogSender log emails.
type LogSender struct {
	From   mail.Address
	Logger Logger
}

// NewLogSender implementation using the provided logger.
func NewLogSender(from string, l Logger) *LogSender {
	return &LogSender{
		From:   mail.Address{Address: from},
		Logger: l,
	}
}

// Send will just log the email.
func (s *LogSender) Send(to, subject, body string) error {
	toAddr := mail.Address{Address: to}
	msg := message(s.From, toAddr, subject, body)
	s.Logger.Logf("\n\n%s\n\n", msg)

	return nil
}
