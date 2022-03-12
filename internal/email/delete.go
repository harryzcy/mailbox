package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/harryzcy/mailbox/internal/datasource/storage"
)

// Delete deletes an trashed email from DynamoDB and S3.
// This action won't be successful if it's not trashed.
func Delete(ctx context.Context, cfg aws.Config, messageID string) error {
	svc := dynamodb.NewFromConfig(cfg)

	_, err := svc.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		ConditionExpression: aws.String("attribute_exists(TrashedTime)"),
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return ErrNotTrashed
		}
		return err
	}

	err = storage.S3.DeleteEmail(ctx, cfg, messageID)
	if err != nil {
		return err
	}

	fmt.Println("delete method finished successfully")
	return nil
}
