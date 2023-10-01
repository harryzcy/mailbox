package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
)

const (
	ActionRead   = "read"
	ActionUnread = "unread"
)

// Read marks an email as read or unread
func Read(ctx context.Context, client api.UpdateItemAPI, messageID, action string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":v_type": &types.AttributeValueMemberS{Value: EmailTypeInbox},
		},
	}
	if action == ActionRead {
		input.UpdateExpression = aws.String("REMOVE Unread")
		input.ConditionExpression = aws.String("attribute_exists(Unread) AND begins_with(TypeYearMonth, :v_type)")
	} else {
		input.UpdateExpression = aws.String("SET Unread = :val1")
		input.ConditionExpression = aws.String("attribute_not_exists(Unread) AND begins_with(TypeYearMonth, :v_type)")
		input.ExpressionAttributeValues[":val1"] = &types.AttributeValueMemberBOOL{Value: true}
	}

	_, err := client.UpdateItem(ctx, input)
	if err != nil {
		if apiErr := new(types.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return ErrReadActionFailed
		}
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return ErrTooManyRequests
		}

		return err
	}

	fmt.Println("read method finished successfully")
	return nil
}
