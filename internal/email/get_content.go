package email

import (
	"context"

	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/platform"
)

// Get returns the email
func GetContent(ctx context.Context, client platform.GetItemContentAPI, messageID, disposition, contentID string) (*storage.GetEmailContentResult, error) {
	return storage.S3.GetEmailContent(ctx, client, messageID, disposition, contentID)
}
