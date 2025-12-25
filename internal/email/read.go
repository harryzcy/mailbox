package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/model"
	"github.com/harryzcy/mailbox/internal/platform"
)

const (
	ActionRead   = "read"
	ActionUnread = "unread"
)

// Read marks an email as read or unread
func Read(ctx context.Context, client platform.UpdateItemAPI, messageID, action string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":v_type": &dynamodbTypes.AttributeValueMemberS{Value: model.EmailTypeInbox},
		},
	}
	if action == ActionRead {
		input.UpdateExpression = aws.String("REMOVE Unread")
		input.ConditionExpression = aws.String("attribute_exists(Unread) AND begins_with(TypeYearMonth, :v_type)")
	} else {
		input.UpdateExpression = aws.String("SET Unread = :val1")
		input.ConditionExpression = aws.String("attribute_not_exists(Unread) AND begins_with(TypeYearMonth, :v_type)")
		input.ExpressionAttributeValues[":val1"] = &dynamodbTypes.AttributeValueMemberBOOL{Value: true}
	}

	_, err := client.UpdateItem(ctx, input)
	if err != nil {
		if apiErr := new(dynamodbTypes.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return platform.ErrReadActionFailed
		}
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return platform.ErrTooManyRequests
		}

		return err
	}

	fmt.Println("read method finished successfully")
	return nil
}
