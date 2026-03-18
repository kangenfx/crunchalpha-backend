package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"
)

type Sender struct {
	mode     string
	fromName string
	fromAddr string
	smtpHost string
	smtpPort string
	smtpUser string
	smtpPass string
}

func NewSender() *Sender {
	mode := os.Getenv("EMAIL_MODE")
	if mode == "" {
		mode = "mock"
	}
	return &Sender{
		mode:     mode,
		fromName: "CrunchAlpha",
		fromAddr: os.Getenv("SMTP_FROM"),
		smtpHost: os.Getenv("SMTP_HOST"),
		smtpPort: os.Getenv("SMTP_PORT"),
		smtpUser: os.Getenv("SMTP_USER"),
		smtpPass: os.Getenv("SMTP_PASS"),
	}
}

func (s *Sender) Send(req EmailRequest) error {
	if s.mode == "mock" {
		return s.sendMock(req)
	}
	return s.sendSMTP(req)
}

func (s *Sender) sendMock(req EmailRequest) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("📧 MOCK EMAIL [%s] To: %s | Subject: %s", timestamp, req.To, req.Subject)
	if req.IsHTML {
		htmlFile := fmt.Sprintf("/tmp/crunchalpha-email-%d.html", time.Now().Unix())
		os.WriteFile(htmlFile, []byte(req.Body), 0644)
		log.Printf("   📄 HTML saved: %s", htmlFile)
	}
	return nil
}

func (s *Sender) sendSMTP(req EmailRequest) error {
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPass, s.smtpHost)

	contentType := "text/plain"
	if req.IsHTML {
		contentType = "text/html"
	}

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s",
		s.fromName, s.fromAddr, req.To, req.Subject, contentType, req.Body)

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)

	// Use TLS for port 465
	if s.smtpPort == "465" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.smtpHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("❌ SMTP TLS dial error: %v", err)
			return s.sendMock(req)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.smtpHost)
		if err != nil {
			return s.sendMock(req)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			log.Printf("❌ SMTP auth error: %v", err)
			return s.sendMock(req)
		}
		if err = client.Mail(s.fromAddr); err != nil {
			return s.sendMock(req)
		}
		if err = client.Rcpt(req.To); err != nil {
			return s.sendMock(req)
		}
		w, err := client.Data()
		if err != nil {
			return s.sendMock(req)
		}
		_, err = fmt.Fprint(w, msg)
		if err != nil {
			return s.sendMock(req)
		}
		w.Close()
		log.Printf("✅ Email sent via SMTP to %s", req.To)
		return nil
	}

	// Port 587 - STARTTLS
	err := smtp.SendMail(addr, auth, s.fromAddr, []string{req.To}, []byte(msg))
	if err != nil {
		log.Printf("❌ SMTP send error: %v - falling back to mock", err)
		return s.sendMock(req)
	}
	log.Printf("✅ Email sent via SMTP to %s", req.To)
	return nil
}

func (s *Sender) SendPasswordReset(email, resetToken string) error {
	resetLink := fmt.Sprintf("http://45.32.118.117:5176/reset-password?token=%s", resetToken)
	data := PasswordResetData{UserEmail: email, ResetLink: resetLink, ExpiresIn: "1 hour"}
	return s.Send(EmailRequest{
		To: email, Subject: "Reset Your CrunchAlpha Password",
		Body: GetPasswordResetHTML(data), IsHTML: true,
	})
}

func (s *Sender) SendWelcome(email, name string) error {
	data := WelcomeData{UserEmail: email, UserName: name}
	return s.Send(EmailRequest{
		To: email, Subject: "Welcome to CrunchAlpha! 🎉",
		Body: GetWelcomeHTML(data), IsHTML: true,
	})
}
