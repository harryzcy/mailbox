package thread

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/harryzcy/mailbox/internal/util/idutil"
)

type Thread struct {
	MessageID   string   `json:"messageID"`
	Type        string   `json:"type"`    // always "thread"
	Subject     string   `json:"subject"` // The subject of the first email in the thread
	EmailIDs    []string `json:"emailIDs"`
	DraftID     string   `json:"draftID,omitempty"`
	TimeUpdated string   `json:"timeUpdated"`           // The time the last email is received or sent
	TrashedTime *string  `json:"trashedTime,omitempty"` // Time in RFC3339 format

	Emails []email.GetResult `json:"emails,omitempty"`
	Draft  *email.GetResult  `json:"draft,omitempty"`
}

func GetThread(ctx context.Context, api email.GetItemAPI, messageID string) (*Thread, error) {
	resp, err := api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, email.ErrTooManyRequests
		}
		return nil, err
	}
	if len(resp.Item) == 0 {
		return nil, email.ErrNotFound
	}

	emailType, _, err := email.UnmarshalGSI(resp.Item)
	if err != nil {
		return nil, err
	}
	if emailType != "thread" {
		return nil, email.ErrNotFound
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

func GetThreadWithEmails(ctx context.Context, api email.GetThreadWithEmailsAPI, messageID string) (*Thread, error) {
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
	if thread.DraftID != "" {
		keys = append(keys, map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: thread.DraftID},
		})
	}

	resp, err := api.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]dynamodbTypes.KeysAndAttributes{
			env.TableName: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, email.ErrTooManyRequests
		}
		return nil, err
	}

	orderMap := map[string]int{}
	for i, emailID := range thread.EmailIDs {
		orderMap[emailID] = i
	}

	thread.Emails = make([]email.GetResult, len(thread.EmailIDs))

	for _, item := range resp.Responses[env.TableName] {
		email, err := email.ParseGetResult(item)
		if err != nil {
			return nil, err
		}

		if email.MessageID == thread.DraftID {
			thread.Draft = email
		} else {
			thread.Emails[orderMap[email.MessageID]] = *email
		}
	}

	return thread, nil
}

type DetermineThreadInput struct {
	InReplyTo  string
	References string
}

type DetermineThreadOutput struct {
	ThreadID          string
	Exists            bool   // If true, the email belongs to an existing thread
	PreviousMessageID string // If Exists is true, the messageID of the last email in the thread

	ShouldCreate    bool   // If true, a new thread should be created
	CreatingEmailID string // If ShouldCreate is true, the messageID of the first email in the thread
	CreatingSubject string // If ShouldCreate is true, the subject of the first email in the thread
	CreatingTime    string // If ShouldCreate is true, the time the first email is received
}

// DetermineThread determines which thread an incoming email belongs to.
// If a thread already exists, the ThreadID is returned and Exists is true.
// If a thread does not exist and a new thread should be created, the ThreadID is randomly generated and ShouldCreate is true.
func DetermineThread(ctx context.Context, api email.QueryAndGetItemAPI, input *DetermineThreadInput) (*DetermineThreadOutput, error) {
	fmt.Println("Determining thread...")
	originalMessageID := ""
	if len(input.InReplyTo) > 0 {
		originalMessageID = input.InReplyTo
	} else if len(input.References) > 0 {
		references := strings.Split(input.References, " ")
		originalMessageID = references[len(references)-1] // The last messageID in the References header
	}

	if originalMessageID == "" {
		return &DetermineThreadOutput{}, nil
	}

	sesDomain := env.Region + ".amazonses.com"
	var possibleSentID string
	if strings.HasSuffix(originalMessageID, "@"+sesDomain+">") {
		fmt.Println("incoming email is replying to a SES email")
		// If the messageID ends with @<SES domain>, it maybe a messageID of a sent email.
		// In this case, we need check if there's a corresponding sent email.
		possibleSentID = strings.TrimSuffix(originalMessageID, "@"+sesDomain+">")
		possibleSentID = strings.TrimPrefix(possibleSentID, "<")
	}

	var previousEmail *email.GetResult
	var err error
	isSentEmail := false
	if possibleSentID != "" {
		// Check if the messageID is a sent email first
		fmt.Println("checking possible sent email")
		previousEmail, err = email.Get(ctx, api, possibleSentID)
		if err != nil && !errors.Is(err, email.ErrNotFound) {
			return nil, err
		}
		isSentEmail = true
	}

	if previousEmail == nil {
		// If the messageID does not corresponded to a sent email, check if it's a received email
		fmt.Println("checking original messageID")
		var resp *dynamodb.QueryOutput
		resp, err = api.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(env.TableName),
			IndexName:              aws.String(env.GsiOriginalIndexName),
			KeyConditionExpression: aws.String("OriginalMessageID = :originalMessageID"),
			ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
				":originalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: originalMessageID},
			},
		})
		if err != nil {
			if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
				return nil, email.ErrTooManyRequests
			}
			return nil, err
		}
		// TODO: handle the case where len(resp.Items) > 1
		if len(resp.Items) == 0 {
			return &DetermineThreadOutput{}, nil
		}

		searchMessageID := resp.Items[0]["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value
		previousEmail, err = email.Get(ctx, api, searchMessageID)
		if err != nil {
			if errors.Is(err, email.ErrNotFound) {
				return &DetermineThreadOutput{}, nil
			}
			return nil, err
		}
	}

	if previousEmail.ThreadID == "" {
		// There's no thread for previousEmail, so we need to create a new thread
		fmt.Println("determining thread finished: new thread should be created")
		threadID := idutil.GenerateThreadID()
		output := &DetermineThreadOutput{
			ThreadID:        threadID,
			ShouldCreate:    true,
			CreatingEmailID: previousEmail.MessageID,
			CreatingSubject: previousEmail.Subject,
			CreatingTime:    previousEmail.TimeReceived,
		}
		if isSentEmail {
			output.CreatingTime = previousEmail.TimeSent
		}
		return output, nil
	}

	if previousEmail.IsThreadLatest {
		fmt.Println("determining thread finished: previous email is the latest email in the thread")
		return &DetermineThreadOutput{
			ThreadID:          previousEmail.ThreadID,
			Exists:            true,
			PreviousMessageID: previousEmail.MessageID,
		}, nil
	}

	fmt.Println("determining thread finished: previous email is not the latest email in the thread")
	thread, err := GetThread(ctx, api, previousEmail.ThreadID)
	if err != nil {
		return nil, err
	}
	fmt.Println("get existing thread finished")
	return &DetermineThreadOutput{
		ThreadID:          previousEmail.ThreadID,
		Exists:            true,
		PreviousMessageID: thread.EmailIDs[len(thread.EmailIDs)-1],
	}, nil
}

type StoreEmailWithExistingThreadInput struct {
	ThreadID          string
	Email             map[string]dynamodbTypes.AttributeValue
	TimeReceived      string
	PreviousMessageID string
}

// StoreEmailWithExistingThread stores the email and updates the thread.
func StoreEmailWithExistingThread(ctx context.Context, api email.TransactWriteItemsAPI, input *StoreEmailWithExistingThreadInput) error {
	input.Email["IsThreadLatest"] = &dynamodbTypes.AttributeValueMemberBOOL{Value: true}
	_, err := api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbTypes.TransactWriteItem{
			{
				// Store new email
				Put: &dynamodbTypes.Put{
					TableName: aws.String(env.TableName),
					Item:      input.Email,
				},
			},
			{
				// Update the thread
				Update: &dynamodbTypes.Update{
					TableName: aws.String(env.TableName),
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
						":timeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: input.TimeReceived},
					},
				},
			},
			{
				// Remove IsThreadLatest from the previous email
				Update: &dynamodbTypes.Update{
					TableName: aws.String(env.TableName),
					Key: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.PreviousMessageID},
					},
					UpdateExpression: aws.String("REMOVE IsThreadLatest"),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

type StoreEmailWithNewThreadInput struct {
	ThreadID        string
	Email           map[string]dynamodbTypes.AttributeValue
	TimeReceived    string
	CreatingEmailID string
	CreatingSubject string
	CreatingTime    string
}

// StoreEmailWithNewThread stores the email, creates a new thread, and add ThreadID to previous email
func StoreEmailWithNewThread(ctx context.Context, api email.TransactWriteItemsAPI, input *StoreEmailWithNewThreadInput) error {
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
		"TimeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: input.TimeReceived},
	}

	input.Email["IsThreadLatest"] = &dynamodbTypes.AttributeValueMemberBOOL{Value: true}
	_, err = api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbTypes.TransactWriteItem{
			{
				// Set ThreadID to previous email
				Update: &dynamodbTypes.Update{
					TableName: aws.String(env.TableName),
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
					TableName: aws.String(env.TableName),
					Item:      input.Email,
				},
			},
			{
				// Create the new thread
				Put: &dynamodbTypes.Put{
					TableName: aws.String(env.TableName),
					Item:      thread,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

type StoreEmailInput struct {
	InReplyTo    string
	References   string
	Item         map[string]dynamodbTypes.AttributeValue
	TimeReceived string // RFC3339
}

// StoreEmail attempts to store the email. If error occurs, it will be logged and the function will return.
func StoreEmail(ctx context.Context, api email.StoreEmailAPI, input *StoreEmailInput) {
	output, err := DetermineThread(ctx, api, &DetermineThreadInput{
		InReplyTo:  input.InReplyTo,
		References: input.References,
	})
	if err != nil {
		log.Printf("failed to determine thread, %v\n", err)
		// continue
	} else {
		input.Item["ThreadID"] = &dynamodbTypes.AttributeValueMemberS{Value: output.ThreadID}
	}

	if output != nil && output.Exists {
		err = StoreEmailWithExistingThread(ctx, api, &StoreEmailWithExistingThreadInput{
			ThreadID:          output.ThreadID,
			Email:             input.Item,
			PreviousMessageID: output.PreviousMessageID,
		})
		if err != nil {
			log.Fatalf("failed to store email with existing thread, %v", err)
		}
		return
	}

	if output != nil && output.ShouldCreate {
		err = StoreEmailWithNewThread(ctx, api, &StoreEmailWithNewThreadInput{
			ThreadID:        output.ThreadID,
			Email:           input.Item,
			TimeReceived:    input.TimeReceived,
			CreatingEmailID: output.CreatingEmailID,
			CreatingSubject: output.CreatingSubject,
			CreatingTime:    output.CreatingTime,
		})
		if err != nil {
			log.Fatalf("failed to store email with new thread, %v", err)
		}
		return
	}

	err = storage.DynamoDB.Store(ctx, api, input.Item)
	if err != nil {
		log.Fatalf("failed to store item in DynamoDB, %v", err)
	}
}
