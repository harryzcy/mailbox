package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
)

// Trash marks an email as trashed
func Trash(ctx context.Context, client api.UpdateItemAPI, messageID string) error {
	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		UpdateExpression:    aws.String("SET TrashedTime = :val1"),
		ConditionExpression: aws.String("attribute_not_exists(TrashedTime) AND NOT begins_with(TypeYearMonth, :v_type)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val1":   &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			":v_type": &types.AttributeValueMemberS{Value: EmailTypeDraft},
		},
	})
	if err != nil {
		if apiErr := new(types.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return api.ErrAlreadyTrashed
		}
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return api.ErrTooManyRequests
		}

		return err
	}

	fmt.Println("trash method finished successfully")
	return nil
}
