package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	platform "github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/model"
)

// Delete deletes an trashed email from DynamoDB and S3.
// This action won't be successful if it's not trashed.
func Delete(ctx context.Context, client platform.DeleteItemAPI, messageID string) error {
	_, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
		ConditionExpression: aws.String("(attribute_exists(TrashedTime) OR begins_with(TypeYearMonth, :v_type)) AND attribute_not_exists(ThreadID)"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":v_type": &dynamodbTypes.AttributeValueMemberS{Value: model.EmailTypeDraft},
		},
	})
	if err != nil {
		var condFailedErr *dynamodbTypes.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return &platform.NotTrashedError{Type: "email"}
		}
		return err
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
