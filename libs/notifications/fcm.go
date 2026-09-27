package notifications

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMSender struct{ client *messaging.Client }

func NewFCMSender(ctx context.Context) (*FCMSender, error) {
	credentials := os.Getenv("FIREBASE_CREDENTIALS_FILE")
	if credentials == "" {
		return nil, nil
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: os.Getenv("FIREBASE_PROJECT_ID")}, option.WithCredentialsFile(credentials))
	if err != nil {
		return nil, fmt.Errorf("initialize firebase: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase messaging: %w", err)
	}
	return &FCMSender{client: client}, nil
}

func (s *FCMSender) Send(ctx context.Context, token, title, body, deeplink, webURL, mobileRoute, entityID string) (string, error) {
	if s == nil || s.client == nil || token == "" {
		return "", nil
	}
	return s.client.Send(ctx, &messaging.Message{Token: token, Notification: &messaging.Notification{Title: title, Body: body}, Data: map[string]string{"deeplink": deeplink, "web_url": webURL, "mobile_route": mobileRoute, "entity_id": entityID}})
}
