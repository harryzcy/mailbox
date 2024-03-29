package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
)

// Delete deletes an trashed email from DynamoDB and S3.
// This action won't be successful if it's not trashed.
func Delete(ctx context.Context, client api.DeleteItemAPI, messageID string) error {
	_, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		ConditionExpression: aws.String("(attribute_exists(TrashedTime) OR begins_with(TypeYearMonth, :v_type)) AND attribute_not_exists(ThreadID)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":v_type": &types.AttributeValueMemberS{Value: EmailTypeDraft},
		},
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return &api.NotTrashedError{Type: "email"}
		}
		return err
	}

	err = storage.S3.DeleteEmail(ctx, client, messageID)
	if err != nil {
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return api.ErrTooManyRequests
		}
		return err
	}

	fmt.Println("delete method finished successfully")
	return nil
}
