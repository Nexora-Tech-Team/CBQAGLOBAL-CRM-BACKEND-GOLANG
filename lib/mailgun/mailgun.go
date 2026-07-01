package mailgun

import (
	"context"
	"net/http"
	"time"

	"github.com/mailgun/mailgun-go/v4"

	"erp-cbqa-global/lib/env"
)

type EmailClient struct {
	Body      string
	Recipient string
	Subject   string
}

func (emailClient *EmailClient) Send() (int, error) {
	mg := mailgun.NewMailgun(env.String("MAILGUN_DOMAIN", ""), env.String("MAILGUN_KEY", ""))
	message := mg.NewMessage(env.String("MAILGUN_SENDER", "do-not-reply@gurih-mart.com"), emailClient.Subject, "", emailClient.Recipient)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	message.SetHtml(emailClient.Body)
	defer cancel()
	if _, _, err := mg.Send(ctx, message); err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusAccepted, nil
}
