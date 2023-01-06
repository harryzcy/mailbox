package email

import (
	"context"

	"github.com/harryzcy/mailbox/internal/datasource/storage"
)

// Get returns the email
func GetContent(ctx context.Context, api GetItemContentAPI, messageID, disposition, contentID string) (*storage.GetEmailContentResult, error) {
	return storage.S3.GetEmailContent(ctx, api, messageID, disposition, contentID)
}
