package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
)

// Reparse re-parse an email from S3 and update the DynamoDB record
func Reparse(ctx context.Context, client platform.ReparseEmailAPI, messageID string) error {
	item := make(map[string]types.AttributeValue)

	emailResult, err := storage.S3.GetEmail(ctx, client, messageID)
	if err != nil {
		return err
	}
	item["Text"] = &types.AttributeValueMemberS{Value: emailResult.Text}
	item["HTML"] = &types.AttributeValueMemberS{Value: emailResult.HTML}
	item["Attachments"] = emailResult.Attachments.ToAttributeValue()
	item["Inlines"] = emailResult.Inlines.ToAttributeValue()
	item["OtherParts"] = emailResult.OtherParts.ToAttributeValue()

	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		UpdateExpression: aws.String("SET #tx = :text, HTML = :html, Attachments = :attachments, Inlines = :inlines, OtherParts = :others"),
		ExpressionAttributeNames: map[string]string{
			"#tx": "Text",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":text":        &types.AttributeValueMemberS{Value: emailResult.Text},
			":html":        &types.AttributeValueMemberS{Value: emailResult.HTML},
			":attachments": emailResult.Attachments.ToAttributeValue(),
			":inlines":     emailResult.Inlines.ToAttributeValue(),
			":others":      emailResult.OtherParts.ToAttributeValue(),
		},
	})
	if err != nil {
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return platform.ErrTooManyRequests
		}

		return err
	}

	fmt.Println("read method finished successfully")
	return nil
}
