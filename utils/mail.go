package utils

import (
	"net/smtp"
	"strings"
)

type MailSender struct {
	smtpServer string
	smtpPort   string
	user       string
	password   string
}

func NewMailSender(smtpServer, user, password string) *MailSender {
	parts := strings.SplitN(smtpServer, ":", 2)
	server, port := parts[0], ""
	if len(parts) > 1 {
		port = ":" + parts[1]
	}
	return &MailSender{
		smtpServer: server,
		smtpPort:   port,
		user:       user,
		password:   password,
	}
}

func (m *MailSender) SendEmail(recipients []string, subject, message string) error {
	msg := "From: " + m.user + "\n" +
		"To: " + strings.Join(recipients, ",") + "\n" +
		"Subject: " + subject + "\n\n" +
		message

	return smtp.SendMail(m.smtpServer+m.smtpPort, smtp.PlainAuth("", m.user, m.password, m.smtpServer), m.user, recipients, []byte(msg))
}
