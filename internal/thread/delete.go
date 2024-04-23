package thread

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/types"
)

// Delete deletes a trashed thread as well as its emails from DynamoDB and S3.
// It will return an error if the thread is not trashed.
func Delete(ctx context.Context, client api.DeleteThreadAPI, messageID string) error {
	thread, err := GetThread(ctx, client, messageID)
	if err != nil {
		return err
	}
	if thread.TrashedTime != nil {
		return &api.NotTrashedError{Type: "thread"}
	}

	transactWriteItems := make([]dynamodbTypes.TransactWriteItem, len(thread.EmailIDs)+1)
	// delete thread
	transactWriteItems[0] = dynamodbTypes.TransactWriteItem{
		Delete: &dynamodbTypes.Delete{
			TableName: aws.String(env.TableName),
			Key: map[string]dynamodbTypes.AttributeValue{
				"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
			},
			ConditionExpression: aws.String("(attribute_exists(TrashedTime)"),
		},
	}

	// delete emails
	for i, emailID := range thread.EmailIDs {
		transactWriteItems[i+1] = dynamodbTypes.TransactWriteItem{
			Delete: &dynamodbTypes.Delete{
				TableName: aws.String(env.TableName),
				Key: map[string]dynamodbTypes.AttributeValue{
					"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: emailID},
				},
				ConditionExpression: aws.String("(attribute_exists(TrashedTime) OR begins_with(TypeYearMonth, :v_type)) AND attribute_exists(ThreadID)"),
				ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
					":v_type": &dynamodbTypes.AttributeValueMemberS{Value: types.EmailTypeDraft},
				},
			},
		}
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactWriteItems,
	})
	if err != nil {
		var condFailedErr *dynamodbTypes.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			// TODO: more specific error checking
			return &api.NotTrashedError{Type: "thread"}
		}
		return err
	}

	err = storage.S3.DeleteEmail(ctx, client, messageID)
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return api.ErrTooManyRequests
		}
		return err
	}

	fmt.Println("delete thread finished successfully")
	return nil
}
