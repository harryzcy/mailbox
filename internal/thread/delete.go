package thread

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
)

// Delete deletes a trashed thread as well as its emails from DynamoDB and S3.
// It will return an error if the thread is not trashed.
func Delete(ctx context.Context, client api.DeleteThreadAPI, messageID string) error {
	thread, err := GetThread(ctx, client, messageID)
	if err != nil {
		return err
	}
	if thread.TrashedTime != nil {
		return email.ErrNotTrashed
	}

	transactWriteItems := make([]types.TransactWriteItem, len(thread.EmailIDs)+1)
	// delete thread
	transactWriteItems[0] = types.TransactWriteItem{
		Delete: &types.Delete{
			TableName: aws.String(env.TableName),
			Key: map[string]types.AttributeValue{
				"MessageID": &types.AttributeValueMemberS{Value: messageID},
			},
			ConditionExpression: aws.String("(attribute_exists(TrashedTime)"),
		},
	}

	// delete emails
	for i, emailID := range thread.EmailIDs {
		transactWriteItems[i+1] = types.TransactWriteItem{
			Delete: &types.Delete{
				TableName: aws.String(env.TableName),
				Key: map[string]types.AttributeValue{
					"MessageID": &types.AttributeValueMemberS{Value: emailID},
				},
				ConditionExpression: aws.String("(attribute_exists(TrashedTime) OR begins_with(TypeYearMonth, :v_type)) AND attribute_exists(ThreadID)"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":v_type": &types.AttributeValueMemberS{Value: email.EmailTypeDraft},
				},
			},
		}
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactWriteItems,
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			// TODO: more specific error checking
			return email.ErrNotTrashed
		}
		return err
	}

	err = storage.S3.DeleteEmail(ctx, client, messageID)
	if err != nil {
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return email.ErrTooManyRequests
		}
		return err
	}

	fmt.Println("delete thread finished successfully")
	return nil
}
