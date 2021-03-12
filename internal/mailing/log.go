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
func (s *LogSender) Send(to, subject, html, text string) error {
	toAddr := mail.Address{Address: to}
	b, err := buildBody(s.From, toAddr, subject, html, text)
	if err != nil {
		return err
	}

	s.Logger.Logf("\n\n%s\n\n", string(b))
	return nil
}
