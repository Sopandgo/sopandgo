package mail

import (
	"net/smtp"
	"strings"
)

func NewSmtpSender(host, port, username, password, from string) *SmtpSender {
	return &SmtpSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SmtpSender) Send(to []string, subject string, body string) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	addr := s.host + ":" + s.port

	// Standard SMTP message format
	msg := []byte("To: " + strings.Join(to, ",") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	return smtp.SendMail(addr, auth, s.from, to, msg)
}
