package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Trash marks an email as trashed
func Trash(ctx context.Context, api UpdateItemAPI, messageID string) error {
	_, err := api.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		UpdateExpression: aws.String("SET TrashedTime = :val1"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val1": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_not_exists(TrashedTime)"),
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return ErrAlreadyTrashed
		}
		return err
	}

	fmt.Println("trash method finished successfully")
	return nil
}
