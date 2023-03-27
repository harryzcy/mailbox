package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/harryzcy/mailbox/internal/util/htmlutil"
)

// CreateInput represents the input of create method
type CreateInput struct {
	EmailInput
	GenerateText string `json:"generateText"` // on, off, or auto (default)
	Send         bool   `json:"send"`         // send email immediately
	ReplyEmailID string `json:"replyEmailID"` // reply to an email, empty if not reply
}

// CreateResult represents the result of create method
type CreateResult struct {
	TimeIndex
	Subject string   `json:"subject"`
	From    []string `json:"from"`
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	ReplyTo []string `json:"replyTo"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
}

func generateMessageID() string {
	rawID := uuid.New()
	messageID := "draft-" + strings.ReplaceAll(rawID.String(), "-", "")
	return messageID
}

var generateText = htmlutil.GenerateText

// Create adds an email as draft in DynamoDB
func Create(ctx context.Context, api CreateAndSendEmailAPI, input CreateInput) (*CreateResult, error) {
	input.MessageID = generateMessageID()
	now := getUpdatedTime()
	typeYearMonth, err := format.FormatTypeYearMonth(EmailTypeDraft, now)
	if err != nil {
		return nil, err
	}
	dateTime := now.Format("02-15:04:05")

	if (input.GenerateText == "on") || (input.GenerateText == "auto" && input.Text == "") {
		var err error
		input.Text, err = generateText(input.HTML)
		if err != nil {
			return nil, err
		}
	}

	item := input.GenerateAttributes(typeYearMonth, dateTime)

	isThread := input.ReplyEmailID != ""
	isExistingThread := false
	if isThread {
		// is part of the thread
		fmt.Println("the new email should be part of the thread, determining the thread info")
		info, err := getThreadInfo(ctx, api, input.ReplyEmailID)
		if err != nil {
			return nil, err
		}

		if info.ThreadID != "" {
			isExistingThread = true
			item["ThreadID"] = &types.AttributeValueMemberS{Value: info.ThreadID}
		}

		if isExistingThread {
			fmt.Println("found existing thread")

			// for existing thread, we need to put the email and add MessageID to thread as DraftID attribute
			_, err = api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
				TransactItems: []types.TransactWriteItem{
					{
						Put: &types.Put{
							TableName: aws.String(tableName),
							Item:      item,
						},
					},
					{
						Update: &types.Update{
							TableName: aws.String(tableName),
							Key: map[string]types.AttributeValue{
								"MessageID": item["ThreadID"],
							},
							UpdateExpression: aws.String("SET DraftID = :draftID"),
							ExpressionAttributeValues: map[string]types.AttributeValue{
								":draftID": item["MessageID"],
							},
						},
					},
				},
			})
			if err != nil {
				if apiErr := new(types.TransactionCanceledException); errors.As(err, &apiErr) {
					return nil, ErrTooManyRequests
				}
				return nil, err
			}
		} else {
			fmt.Println("thread does not exist, create a new thread")
			// for new thread, we need to
			// 1) put the email,
			// 2) create a new thread with DraftID,
			// 3) add ThreadID to the previous email
			threadID := generateThreadID()
			item["ThreadID"] = &types.AttributeValueMemberS{Value: threadID}

			t := time.Now().UTC()
			var threadTypeYearMonth string
			threadTypeYearMonth, err = format.FormatTypeYearMonth(EmailTypeThread, t)
			if err != nil {
				return nil, err
			}

			thread := map[string]types.AttributeValue{
				"MessageID":     &types.AttributeValueMemberS{Value: threadID},
				"TypeYearMonth": &types.AttributeValueMemberS{Value: threadTypeYearMonth},
				"Subject":       &types.AttributeValueMemberS{Value: info.CreatingSubject},
				"EmailIDs": &types.AttributeValueMemberL{
					Value: []types.AttributeValue{
						&types.AttributeValueMemberS{Value: info.CreatingEmailID},
					},
				},
				"TimeUpdated": &types.AttributeValueMemberS{Value: format.FormatRFC3399(t)},
				"DraftID":     item["MessageID"],
			}
			_, err = api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
				TransactItems: []types.TransactWriteItem{
					{
						Put: &types.Put{
							TableName: aws.String(tableName),
							Item:      item,
						},
					},
					{
						Put: &types.Put{
							TableName: aws.String(tableName),
							Item:      thread,
						},
					},
					{
						Update: &types.Update{
							TableName: aws.String(tableName),
							Key: map[string]types.AttributeValue{
								"MessageID": &types.AttributeValueMemberS{Value: info.CreatingEmailID},
							},
							UpdateExpression: aws.String("SET #threadID = :threadID, #isThreadLatest = :isThreadLatest"),
							ExpressionAttributeNames: map[string]string{
								"#threadID":       "ThreadID",
								"#isThreadLatest": "IsThreadLatest",
							},
							ExpressionAttributeValues: map[string]types.AttributeValue{
								":threadID":       &types.AttributeValueMemberS{Value: threadID},
								":isThreadLatest": &types.AttributeValueMemberBOOL{Value: true},
							},
						},
					},
				},
			})
			if err != nil {
				if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
					return nil, ErrTooManyRequests
				}
				return nil, err
			}
		}
	} else {
		// is not part of the thread, so we can just put the email
		_, err = api.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err != nil {
			if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
				return nil, ErrTooManyRequests
			}
			return nil, err
		}
	}

	emailType := EmailTypeDraft
	if input.Send {
		email := &EmailInput{
			MessageID: input.MessageID,
			Subject:   input.Subject,
			From:      input.From,
			To:        input.To,
			Cc:        input.Cc,
			Bcc:       input.Bcc,
			ReplyTo:   input.ReplyTo,
			Text:      input.Text,
			HTML:      input.HTML,
		}

		var newMessageID string
		if newMessageID, err = sendEmailViaSES(ctx, api, email); err != nil {
			return nil, err
		}
		email.MessageID = newMessageID

		if err = markEmailAsSent(ctx, api, input.MessageID, email); err != nil {
			return nil, err
		}
		input.MessageID = newMessageID
		emailType = EmailTypeSent
	}

	result := &CreateResult{
		TimeIndex: TimeIndex{
			MessageID:   input.MessageID,
			Type:        emailType,
			TimeUpdated: now.Format(time.RFC3339),
		},
		Subject: input.Subject,
		From:    input.From,
		To:      input.To,
		Cc:      input.Cc,
		Bcc:     input.Bcc,
		ReplyTo: input.ReplyTo,
		Text:    input.Text,
		HTML:    input.HTML,
	}

	fmt.Println("create method finished successfully")
	return result, nil
}

type ThreadInfo struct {
	ThreadID string

	References string // used by email reply

	// used to create a new thread
	CreatingEmailID string
	CreatingSubject string
}

func getThreadInfo(ctx context.Context, api CreateAndSendEmailAPI, replyEmailID string) (*ThreadInfo, error) {
	fmt.Println("getting email to reply to")
	email, err := Get(ctx, api, replyEmailID)
	if err != nil {
		return nil, err
	}

	return &ThreadInfo{
		ThreadID:        email.ThreadID,
		References:      email.References,
		CreatingEmailID: email.MessageID,
		CreatingSubject: email.Subject,
	}, nil
}
