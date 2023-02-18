package email

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Thread struct {
	MessageID   string   `json:"messageID"`
	Type        string   `json:"type"`    // always "thread"
	Subject     string   `json:"subject"` // The subject of the first email in the thread
	EmailIDs    []string `json:"emailIDs"`
	TimeUpdated string   `json:"timeUpdated"` // The time the last email is received

	Emails []GetResult `json:"emails"`
}

func GetThread(ctx context.Context, api GetItemAPI, messageID string) (*Thread, error) {
	resp, err := api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, ErrTooManyRequests
		}
		return nil, err
	}
	if len(resp.Item) == 0 {
		return nil, ErrNotFound
	}

	emailType, _, err := unmarshalGSI(resp.Item)
	if err != nil {
		return nil, err
	}
	if emailType != "thread" {
		return nil, ErrNotFound
	}

	result := &Thread{
		Type: "thread",
	}
	err = attributevalue.UnmarshalMap(resp.Item, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetThreadWithEmails(ctx context.Context, api GetThreadWithEmailsAPI, messageID string) (*Thread, error) {
	thread, err := GetThread(ctx, api, messageID)
	if err != nil {
		return nil, err
	}

	keys := []map[string]dynamodbTypes.AttributeValue{}
	for _, emailID := range thread.EmailIDs {
		keys = append(keys, map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: emailID},
		})
	}

	resp, err := api.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]dynamodbTypes.KeysAndAttributes{
			tableName: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, ErrTooManyRequests
		}
		return nil, err
	}

	for _, item := range resp.Responses[tableName] {
		email, err := parseGetResult(item)
		if err != nil {
			return nil, err
		}

		thread.Emails = append(thread.Emails, *email)
	}

	return thread, nil
}
