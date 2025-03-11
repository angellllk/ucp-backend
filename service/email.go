package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gopkg.in/gomail.v2"
)

type EmailService struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
}

func NewEmailService(host string, port int, username, password string, from string) *EmailService {
	return &EmailService{
		SMTPHost: host,
		SMTPPort: port,
		Username: username,
		Password: password,
		From:     from,
	}
}

func (e *EmailService) SendEmail(to, subject, body string) error {
	mail := gomail.NewMessage()
	mail.SetHeader("From", e.From)
	mail.SetHeader("To", to)
	mail.SetHeader("Subject", subject)
	mail.SetBody("text/html", body)

	dialer := gomail.NewDialer(e.SMTPHost, e.SMTPPort, e.Username, e.Password)
	dialer.SSL = true

	return dialer.DialAndSend(mail)
}

func GenerateToken(email string, timestamp int64) string {
	secretKey := "$ecretKey_).r0@+"
	data := fmt.Sprintf("%s:%d", email, timestamp)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func ValidateToken(email, token string, timestamp int64) bool {
	expectedToken := GenerateToken(email, timestamp)
	return hmac.Equal([]byte(expectedToken), []byte(token))
}
