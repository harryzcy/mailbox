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
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/harryzcy/mailbox/internal/util/htmlutil"
	"github.com/harryzcy/mailbox/internal/util/idutil"
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
	Subject  string   `json:"subject"`
	From     []string `json:"from"`
	To       []string `json:"to"`
	Cc       []string `json:"cc"`
	Bcc      []string `json:"bcc"`
	ReplyTo  []string `json:"replyTo"`
	Text     string   `json:"text"`
	HTML     string   `json:"html"`
	ThreadID string   `json:"threadID,omitempty"`
}

func generateDraftID() string {
	rawID := uuid.New()
	messageID := "draft-" + strings.ReplaceAll(rawID.String(), "-", "")
	return messageID
}

var generateText = htmlutil.GenerateText

// Create adds an email as draft in DynamoDB
func Create(ctx context.Context, client api.CreateAndSendEmailAPI, input CreateInput) (*CreateResult, error) {
	input.MessageID = generateDraftID()
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
	var threadID, inReplyTo, references string
	if isThread {
		// the draft email is a reply, so it should be part of the thread
		// next we need to determine if there's an existing thread, or we need to create a new thread
		fmt.Println("the new email should be part of the thread, determining the thread info")
		info, err := getThreadInfo(ctx, client, input.ReplyEmailID)
		if err != nil {
			return nil, err
		}

		if info.ThreadID != "" {
			// if the thread ID is not empty, then there's an existing thread
			isExistingThread = true
			threadID = info.ThreadID
			item["ThreadID"] = &types.AttributeValueMemberS{Value: info.ThreadID}
		}

		// The In-Reply-To header field contains the Message-ID of the message being replied to,
		// and the References header contains a list of Message-IDs of all messages in the thread,
		// according to RFC 5332 3.6.4 in-reply-to and references.
		inReplyTo = info.ReplyToMessageID
		if inReplyTo == "" {
			return nil, errors.New("in-reply-to is empty")
		}
		if !strings.HasPrefix(inReplyTo, "<") && !strings.HasSuffix(inReplyTo, ">") {
			inReplyTo = "<" + inReplyTo + ">" // RFC 5332 3.6.4 msg-id, Message-ID must be enclosed in angle brackets
		}
		references = info.References + " " + inReplyTo
		item["InReplyTo"] = &types.AttributeValueMemberS{Value: inReplyTo}
		item["References"] = &types.AttributeValueMemberS{Value: references}

		if isExistingThread {
			fmt.Println("found existing thread")
			// for existing thread, we need to put the email and add MessageID to thread as DraftID attribute
			_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
				TransactItems: []types.TransactWriteItem{
					{
						Put: &types.Put{
							TableName: aws.String(env.TableName),
							Item:      item,
						},
					},
					{
						Update: &types.Update{
							TableName: aws.String(env.TableName),
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
			fmt.Println("thread does not exist, creating a new thread")
			// for new thread, we need to:
			// 1) put the email,
			// 2) create a new thread with DraftID,
			// 3) add ThreadID to the previous email
			threadID = idutil.GenerateThreadID()
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
			_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
				TransactItems: []types.TransactWriteItem{
					{
						Put: &types.Put{
							TableName: aws.String(env.TableName),
							Item:      item,
						},
					},
					{
						Put: &types.Put{
							TableName: aws.String(env.TableName),
							Item:      thread,
						},
					},
					{
						Update: &types.Update{
							TableName: aws.String(env.TableName),
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
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(env.TableName),
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
			MessageID:  input.MessageID,
			Subject:    input.Subject,
			From:       input.From,
			To:         input.To,
			Cc:         input.Cc,
			Bcc:        input.Bcc,
			ReplyTo:    input.ReplyTo,
			Text:       input.Text,
			HTML:       input.HTML,
			ThreadID:   threadID,
			InReplyTo:  inReplyTo,
			References: references,
		}

		var newMessageID string
		if newMessageID, err = sendEmailViaSES(ctx, client, email); err != nil {
			return nil, err
		}
		email.MessageID = newMessageID

		if err = markEmailAsSent(ctx, client, input.MessageID, email); err != nil {
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
		Subject:  input.Subject,
		From:     input.From,
		To:       input.To,
		Cc:       input.Cc,
		Bcc:      input.Bcc,
		ReplyTo:  input.ReplyTo,
		Text:     input.Text,
		HTML:     input.HTML,
		ThreadID: threadID,
	}

	fmt.Println("create method finished successfully")
	return result, nil
}

type ThreadInfo struct {
	ThreadID string

	References string // used by email reply

	// used to create a new thread
	CreatingEmailID  string
	CreatingSubject  string
	ReplyToMessageID string // the original message id from the sender, rather than the one generated by SES
}

func getThreadInfo(ctx context.Context, client api.CreateAndSendEmailAPI, replyEmailID string) (*ThreadInfo, error) {
	fmt.Println("getting email to reply to")
	email, err := Get(ctx, client, replyEmailID)
	if err != nil {
		return nil, err
	}
	var replyToMessageID string
	if email.Type == EmailTypeInbox {
		replyToMessageID = email.OriginalMessageID
	} else if email.Type == EmailTypeSent {
		replyToMessageID = fmt.Sprintf("%s@%s.amazonses.com", email.MessageID, env.Region)
	} else {
		return nil, errors.New("invalid email type")
	}

	return &ThreadInfo{
		ThreadID:         email.ThreadID,
		References:       email.References,
		CreatingEmailID:  email.MessageID,
		CreatingSubject:  email.Subject,
		ReplyToMessageID: replyToMessageID,
	}, nil
}
