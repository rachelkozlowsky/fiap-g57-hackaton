package email

import (
	"fmt"
	"log"
	"net/smtp"

	"notification-service/infra/utils"
)

type SMTPClient struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func InitSMTP() *SMTPClient {
	host := utils.GetEnv("SMTP_HOST", "smtp.gmail.com")
	port := utils.GetEnv("SMTP_PORT", "587")
	username := utils.GetEnv("SMTP_USER", "")
	password := utils.GetEnv("SMTP_PASSWORD", "")
	from := utils.GetEnv("SMTP_FROM", "noreply@g57.com")

	if username == "" || password == "" {
		log.Println("SMTP credentials not configured, emails will not be sent")
	} else {
		log.Println("SMTP client initialized")
	}

	return &SMTPClient{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPClient) SendEmail(to, subject, htmlBody string) error {
	if s.username == "" || s.password == "" {
		log.Printf("Skipping email to %s (SMTP not configured)", to)
		return nil
	}

	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	message := s.composeMessage(to, subject, htmlBody)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(message))
	
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent to %s", to)
	return nil
}

func (s *SMTPClient) composeMessage(to, subject, htmlBody string) string {
	headers := make(map[string]string)
	headers["From"] = s.from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	return message
}

func (s *SMTPClient) SendPlainEmail(to, subject, body string) error {
	if s.username == "" || s.password == "" {
		log.Printf("Skipping email to %s (SMTP not configured)", to)
		return nil
	}

	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	headers := make(map[string]string)
	headers["From"] = s.from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(message))
	
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent to %s", to)
	return nil
}
