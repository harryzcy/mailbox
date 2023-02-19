package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/harryzcy/mailbox/internal/util/format"
)

type Thread struct {
	MessageID   string   `json:"messageID"`
	Type        string   `json:"type"`    // always "thread"
	Subject     string   `json:"subject"` // The subject of the first email in the thread
	EmailIDs    []string `json:"emailIDs"`
	TimeUpdated string   `json:"timeUpdated"` // The time the last email is received

	Emails []GetResult `json:"emails,omitempty"`
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

type DetermineThreadInput struct {
	InReplyTo  string
	References string
}

type DetermineThreadOutput struct {
	ThreadID        string
	Exists          bool   // If true, the email belongs to an existing thread
	ShouldCreate    bool   // If true, a new thread should be created
	CreatingEmailID string // If ShouldCreate is true, the messageID of the first email in the thread
	CreatingSubject string // If ShouldCreate is true, the subject of the first email in the thread
	CreatingTime    string // If ShouldCreate is true, the time the first email is received
}

// DetermineThread determines which thread an incoming email belongs to.
// If a thread already exists, the ThreadID is returned and Exists is true.
// If a thread does not exist and a new thread should be created, the ThreadID is randomly generated and ShouldCreate is true.
func DetermineThread(ctx context.Context, api GetItemAPI, input *DetermineThreadInput) (*DetermineThreadOutput, error) {
	searchMessageID := ""
	if len(input.InReplyTo) > 0 {
		searchMessageID = input.InReplyTo
	} else if len(input.References) > 0 {
		references := strings.Split(input.References, " ")
		searchMessageID = references[len(references)-1] // The last messageID in the References header
	}

	if searchMessageID == "" {
		return nil, nil
	}

	// TODO: fix: searchMessageID should be OriginalMessageID
	previousEmail, err := Get(ctx, api, searchMessageID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if previousEmail.ThreadID == "" {
		threadID := generateThreadID()
		return &DetermineThreadOutput{
			ThreadID:        threadID,
			ShouldCreate:    true,
			CreatingEmailID: previousEmail.MessageID,
			CreatingSubject: previousEmail.Subject,
			CreatingTime:    previousEmail.TimeReceived,
		}, nil
	}

	return &DetermineThreadOutput{
		ThreadID: previousEmail.ThreadID,
		Exists:   true,
	}, nil
}

func generateThreadID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

type StoreEmailWithExistingThreadInput struct {
	ThreadID string
	Email    map[string]dynamodbTypes.AttributeValue
}

// StoreEmailWithExistingThread stores the email and updates the thread.
func StoreEmailWithExistingThread(ctx context.Context, api TransactWriteItemsAPI, input *StoreEmailWithExistingThreadInput) error {
	resp, err := api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbTypes.TransactWriteItem{
			{
				Put: &dynamodbTypes.Put{
					TableName: aws.String(tableName),
					Item:      input.Email,
				},
			},
			{
				Update: &dynamodbTypes.Update{
					TableName: aws.String(tableName),
					Key: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.ThreadID},
					},
					UpdateExpression: aws.String("SET #emails = list_append(#emails, :emails), #timeUpdated = :timeUpdated"),
					ExpressionAttributeNames: map[string]string{
						"#emails":      "EmailIDs",
						"#timeUpdated": "TimeUpdated",
					},
					ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
						":emails":      &dynamodbTypes.AttributeValueMemberL{Value: []dynamodbTypes.AttributeValue{input.Email["MessageID"]}},
						":timeUpdated": input.Email["TimeReceived"],
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("DynamoDB returned metadata: %s", resp.ResultMetadata)

	return nil
}

type StoreEmailWithNewThreadInput struct {
	ThreadID        string
	Email           map[string]dynamodbTypes.AttributeValue
	CreatingEmailID string
	CreatingSubject string
	CreatingTime    string
}

// StoreEmailWithNewThread stores the email, creates a new thread, and add ThreadID to previous email
func StoreEmailWithNewThread(ctx context.Context, api TransactWriteItemsAPI, input *StoreEmailWithNewThreadInput) error {
	t, err := time.Parse(time.RFC3339, input.CreatingTime)
	if err != nil {
		return err
	}
	typeYearMonth, err := format.FormatTypeYearMonth("thread", t)
	if err != nil {
		return err
	}

	thread := map[string]dynamodbTypes.AttributeValue{
		"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: input.ThreadID},
		"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: typeYearMonth},
		"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: input.CreatingSubject},
		"EmailIDs": &dynamodbTypes.AttributeValueMemberL{
			Value: []dynamodbTypes.AttributeValue{
				&dynamodbTypes.AttributeValueMemberS{Value: input.CreatingEmailID},
				input.Email["MessageID"],
			},
		},
		"TimeUpdated": input.Email["TimeReceived"],
	}

	resp, err := api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbTypes.TransactWriteItem{
			{
				// Set ThreadID to previous email
				Update: &dynamodbTypes.Update{
					TableName: aws.String(tableName),
					Key: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.CreatingEmailID},
					},
					UpdateExpression: aws.String("SET #threadID = :threadID"),
					ExpressionAttributeNames: map[string]string{
						"#threadID": "ThreadID",
					},
					ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
						":threadID": &dynamodbTypes.AttributeValueMemberS{Value: input.ThreadID},
					},
				},
			},
			{
				// Store the new email
				Put: &dynamodbTypes.Put{
					TableName: aws.String(tableName),
					Item:      input.Email,
				},
			},
			{
				// Create the new thread
				Put: &dynamodbTypes.Put{
					TableName: aws.String(tableName),
					Item:      thread,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("DynamoDB returned metadata: %s", resp.ResultMetadata)
	return nil
}
