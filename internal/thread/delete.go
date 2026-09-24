package thread

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/platform"
)

// Delete deletes a trashed thread as well as its emails from DynamoDB and S3.
// It will return an error if the thread is not trashed.
func Delete(ctx context.Context, client platform.DeleteThreadAPI, messageID string) error {
	thread, err := GetThread(ctx, client, messageID)
	if err != nil {
		return err
	}
	if thread.TrashedTime == nil {
		return &platform.NotTrashedError{Type: "thread"}
	}

	emailIDs := thread.EmailIDs
	if thread.DraftID != "" {
		emailIDs = append(emailIDs, thread.DraftID)
	}

	transactWriteItems := make([]dynamodbTypes.TransactWriteItem, len(emailIDs)+1)
	// delete thread
	transactWriteItems[0] = dynamodbTypes.TransactWriteItem{
		Delete: &dynamodbTypes.Delete{
			TableName: aws.String(env.TableName),
			Key: map[string]dynamodbTypes.AttributeValue{
				"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
			},
			ConditionExpression: aws.String("attribute_exists(TrashedTime)"),
		},
	}

	// delete emails, which are not trashed individually when the thread is trashed
	for i, emailID := range emailIDs {
		transactWriteItems[i+1] = dynamodbTypes.TransactWriteItem{
			Delete: &dynamodbTypes.Delete{
				TableName: aws.String(env.TableName),
				Key: map[string]dynamodbTypes.AttributeValue{
					"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: emailID},
				},
				ConditionExpression: aws.String("ThreadID = :threadID"),
				ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
					":threadID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
				},
			},
		}
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactWriteItems,
	})
	if err != nil {
		var canceledErr *dynamodbTypes.TransactionCanceledException
		if errors.As(err, &canceledErr) && len(canceledErr.CancellationReasons) > 0 &&
			aws.ToString(canceledErr.CancellationReasons[0].Code) == "ConditionalCheckFailed" {
			return &platform.NotTrashedError{Type: "thread"}
		}
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return platform.ErrTooManyRequests
		}
		return err
	}

	for _, emailID := range emailIDs {
		err = storage.S3.DeleteEmail(ctx, client, emailID)
		if err != nil {
			return err
		}
	}

	fmt.Println("delete thread finished successfully")
	return nil
}
