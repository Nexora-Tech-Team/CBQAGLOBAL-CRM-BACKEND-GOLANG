package smtp

import (
	"erp-cbqa-global/lib/env"
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
)

type SendEmailSmtpRequest struct {
	To      []string
	Subject string
	Body    string
}

// SendEmailSmtp is
func SendEmailSmtp(emailReq SendEmailSmtpRequest) (int, error) {
	authEmail := smtp.PlainAuth("",
		env.String("EMAIL_AUTH_EMAIL", ""),
		env.String("EMAIL_AUTH_PASSWORD", ""),
		env.String("EMAIL_SMTP_HOST", "smtp.gmail.com"))
	addressEmail := fmt.Sprintf("%s:%s",
		env.String("EMAIL_SMTP_HOST", "smtp.gmail.com"),
		env.String("EMAIL_SMTP_PORT", "587"))
	fromEmail := mail.Address{
		Name: env.String("EMAIL_SENDER_NAME", "noreply@fulfillment.gurih.com"),
	}
	bodyEmail := "From: " + fromEmail.String() + "\n" +
		"To: " + strings.Join(emailReq.To, ",") + "\n" +
		"Subject: " + emailReq.Subject + "\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		emailReq.Body

	err := smtp.SendMail(addressEmail,
		authEmail, env.String("EMAIL_AUTH_EMAIL", ""),
		emailReq.To, []byte(bodyEmail))
	if err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusAccepted, nil
}
