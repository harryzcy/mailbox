package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Untrash marks an trashed email as not trashed
func Untrash(ctx context.Context, api UpdateItemAPI, messageID string) error {
	_, err := api.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
		UpdateExpression:    aws.String("REMOVE TrashedTime"),
		ConditionExpression: aws.String("attribute_exists(TrashedTime) AND NOT begins_with(TypeYearMonth, :v_type)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":v_type": &types.AttributeValueMemberS{Value: EmailTypeDraft},
		},
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return ErrNotTrashed
		}
		return err
	}

	fmt.Println("trash method finished successfully")
	return nil
}
