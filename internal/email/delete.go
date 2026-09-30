package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/model"
	"github.com/harryzcy/mailbox/internal/platform"
)

// Delete deletes an email from DynamoDB and S3.
// Drafts can always be deleted, other emails only if they're trashed and not part of a thread.
func Delete(ctx context.Context, client platform.DeleteEmailAPI, messageID string) error {
	resp, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
		ConditionExpression: aws.String("begins_with(TypeYearMonth, :v_type) OR (attribute_exists(TrashedTime) AND attribute_not_exists(ThreadID))"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":v_type": &dynamodbTypes.AttributeValueMemberS{Value: model.EmailTypeDraft},
		},
		ReturnValues: dynamodbTypes.ReturnValueAllOld,
	})
	if err != nil {
		var condFailedErr *dynamodbTypes.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return &platform.NotTrashedError{Type: "email"}
		}
		return err
	}

	// only drafts can be deleted while part of a thread, i.e. reply drafts
	if threadID, ok := resp.Attributes["ThreadID"].(*dynamodbTypes.AttributeValueMemberS); ok {
		if err = removeDraftFromThread(ctx, client, threadID.Value, messageID); err != nil {
			return err
		}
	}

	err = storage.S3.DeleteEmail(ctx, client, messageID)
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return platform.ErrTooManyRequests
		}
		return err
	}

	fmt.Println("delete method finished successfully")
	return nil
}

func removeDraftFromThread(ctx context.Context, client platform.UpdateItemAPI, threadID, draftID string) error {
	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: threadID},
		},
		UpdateExpression:    aws.String("REMOVE DraftID"),
		ConditionExpression: aws.String("DraftID = :draftID"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":draftID": &dynamodbTypes.AttributeValueMemberS{Value: draftID},
		},
	})
	if err != nil {
		// the thread already points to a newer draft
		var condFailedErr *dynamodbTypes.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return nil
		}
		return err
	}
	return nil
}
