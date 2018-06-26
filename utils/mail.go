package utils

import (
	"net/smtp"
	"strings"
)

type MailSender struct {
	smtpServer   string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	from         string
}

func NewMailSender(smtpServer, smtpUser, smtpPassword, from string) *MailSender {
	parts := strings.SplitN(smtpServer, ":", 2)
	server, port := parts[0], ""
	if len(parts) > 1 {
		port = ":" + parts[1]
	}
	return &MailSender{
		smtpServer:   server,
		smtpPort:     port,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		from:         from,
	}
}

func (m *MailSender) SendEmail(recipients []string, subject, message string) error {
	from := m.from
	if from == "" {
		from = m.smtpUser
	}

	msg := "From: " + m.from + "\n" +
		"To: " + strings.Join(recipients, ",") + "\n" +
		"Subject: " + subject + "\n\n" +
		message

	return smtp.SendMail(m.smtpServer+m.smtpPort, smtp.PlainAuth("", m.smtpUser, m.smtpPassword, m.smtpServer), m.smtpUser, recipients, []byte(msg))
}
