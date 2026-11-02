package email

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/model"
	"github.com/harryzcy/mailbox/internal/platform"
)

// Delete deletes an email from S3 and DynamoDB.
// Drafts can always be deleted, other emails only if they're trashed and not part of a thread.
func Delete(ctx context.Context, client platform.DeleteEmailAPI, messageID string) error {
	deletable, err := isDeletable(ctx, client, messageID)
	if err != nil {
		return err
	}
	if !deletable {
		return &platform.NotTrashedError{Type: "email"}
	}

	err = storage.S3.DeleteEmail(ctx, client, messageID)
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return platform.ErrTooManyRequests
		}
		return err
	}

	// the condition is checked again, in case the email was untrashed since
	resp, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
		ConditionExpression: aws.String("begins_with(TypeYearMonth, :v_type) OR (attribute_exists(TrashedTime) AND " + NotInThreadCondition + ")"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":v_type": &dynamodbTypes.AttributeValueMemberS{Value: model.EmailTypeDraft},
			":zero":   &dynamodbTypes.AttributeValueMemberN{Value: "0"},
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
	if threadID, ok := resp.Attributes["ThreadID"].(*dynamodbTypes.AttributeValueMemberS); ok && threadID.Value != "" {
		if err = removeDraftFromThread(ctx, client, threadID.Value, messageID); err != nil {
			return err
		}
	}

	fmt.Println("delete method finished successfully")
	return nil
}

// isDeletable reports whether the email meets the delete condition, reading strongly consistently
func isDeletable(ctx context.Context, client platform.GetItemAPI, messageID string) (bool, error) {
	resp, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String("TypeYearMonth, TrashedTime, ThreadID"),
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return false, platform.ErrTooManyRequests
		}
		return false, err
	}

	if typeYearMonth, ok := resp.Item["TypeYearMonth"].(*dynamodbTypes.AttributeValueMemberS); ok &&
		strings.HasPrefix(typeYearMonth.Value, model.EmailTypeDraft) {
		return true, nil
	}
	if _, trashed := resp.Item["TrashedTime"]; !trashed {
		return false, nil
	}
	threadID, ok := resp.Item["ThreadID"].(*dynamodbTypes.AttributeValueMemberS)
	return !ok || threadID.Value == "", nil
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
