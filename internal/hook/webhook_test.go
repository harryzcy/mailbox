package hook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harryzcy/mailbox/internal/env"
	"github.com/stretchr/testify/assert"
)

func TestSendWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var webhook Webhook
		err := json.NewDecoder(req.Body).Decode(&webhook)
		assert.Nil(t, err)
		assert.Equal(t, EventEmail, webhook.Event)
		assert.Equal(t, ActionReceived, webhook.Action)
		assert.Equal(t, "123", webhook.Email.ID)

		_, err = rw.Write([]byte("OK"))
		assert.Nil(t, err)
	}))
	defer server.Close()

	env.WebhookURL = server.URL
	err := SendWebhook(context.Background(), &Webhook{
		Event:  EventEmail,
		Action: ActionReceived,
		Email:  Email{ID: "123"},
	})
	assert.Nil(t, err)
}

func TestSendWebhook_Error(t *testing.T) {
	env.WebhookURL = ""
	err := SendWebhook(context.Background(), &Webhook{
		Event:  EventEmail,
		Action: ActionReceived,
		Email:  Email{ID: "123"},
	})
	assert.NotNil(t, err)
}
