package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/harryzcy/mailbox/internal/env"
)

// webhookEnabled returns true if webhook is enabled.
func webhookEnabled() bool {
	return env.WebhookURL != ""
}

// SendWebhook sends a webhook to the configured URL, if webhook is enabled.
// Otherwise, it does nothing.
func SendWebhook(ctx context.Context, data *Hook) error {
	if !webhookEnabled() {
		return nil
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, env.WebhookURL, body)
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
