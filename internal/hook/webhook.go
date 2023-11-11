package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/harryzcy/mailbox/internal/env"
)

func webhookEnabled() bool {
	return env.WebhookURL != ""
}

// SendWebhook sends a webhook to the configured URL, if webhook is enabled.
// Otherwise, it does nothing.
func SendWebhook(ctx context.Context, data *Webhook) error {
	if !webhookEnabled() {
		return nil
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, env.WebhookURL, bytes.NewReader(body))
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
