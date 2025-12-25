package email

import (
	"context"

	platform "github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
)

// Get returns the email
func GetContent(ctx context.Context, client platform.GetItemContentAPI, messageID, disposition, contentID string) (*storage.GetEmailContentResult, error) {
	return storage.S3.GetEmailContent(ctx, client, messageID, disposition, contentID)
}
