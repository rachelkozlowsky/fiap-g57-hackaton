package email

import (
	"crypto/tls"
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
	return s.dialAndSend(to, message, auth)
}

func (s *SMTPClient) dialAndSend(to, message string, auth smtp.Auth) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	
	// Dial the SMTP server
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial smtp: %w", err)
	}
	defer c.Quit()

	if ok, _ := c.Extension("STARTTLS"); ok {
		log.Printf("Iniciando STARTTLS para %s", s.host)
		config := &tls.Config{
			ServerName:         s.host,
			InsecureSkipVerify: true,
		}
		if err = c.StartTLS(config); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
		log.Println("STARTTLS concluído com sucesso")
	} else {
		log.Println("Aviso: Servidor não anunciou suporte a STARTTLS")
	}

	if auth != nil {
		log.Printf("Autenticando usuário %s no host %s", s.username, s.host)
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
		log.Println("Autenticação concluída")
	}

	if err = c.Mail(s.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err = c.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("failed to create data writer: %w", err)
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	log.Printf("Email sent successfully to %s", to)
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

	return s.dialAndSend(to, message, auth)
}
