package thread

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
)

func Trash(ctx context.Context, client api.UpdateItemAPI, threadID string) error {
	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: threadID},
		},
		UpdateExpression:    aws.String("SET TrashedTime = :val1"),
		ConditionExpression: aws.String("attribute_not_exists(TrashedTime)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val1": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		if apiErr := new(types.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return email.ErrAlreadyTrashed
		}
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return email.ErrTooManyRequests
		}

		return err
	}

	fmt.Println("trash thread finished successfully")
	return nil
}
