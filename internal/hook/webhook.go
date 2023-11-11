package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/harryzcy/mailbox/internal/env"
)

const (
	EventEmail     = "email"
	ActionReceived = "received"
)

type Webhook struct {
	Event  string `json:"event"`
	Action string `json:"action"`
	Email  Email
}

type Email struct {
	ID string `json:"id"` // message id
}

func Enabled() bool {
	return env.WebhookURL != ""
}

func SendWebhook(ctx context.Context, data *Webhook) error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", env.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
