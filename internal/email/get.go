package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/types"
)

// GetResult represents the result of get method
type GetResult struct {
	MessageID         string   `json:"messageID"`
	OriginalMessageID string   `json:"originalMessageID"`
	Type              string   `json:"type"`
	Subject           string   `json:"subject"`
	From              []string `json:"from"`
	To                []string `json:"to"`
	Text              string   `json:"text"`
	HTML              string   `json:"html"`
	ReplyTo           []string `json:"replyTo"`
	InReplyTo         string   `json:"inReplyTo"`
	References        string   `json:"references"` // space separated string
	ThreadID          string   `json:"threadID,omitempty"`
	IsThreadLatest    bool     `json:"isThreadLatest,omitempty"`

	// Inbox email attributes
	TimeReceived string        `json:"timeReceived,omitempty"`
	DateSent     string        `json:"dateSent,omitempty"`
	Source       string        `json:"source,omitempty"`
	Destination  []string      `json:"destination,omitempty"`
	ReturnPath   string        `json:"returnPath,omitempty"`
	Verdict      *EmailVerdict `json:"verdict,omitempty"`
	Unread       *bool         `json:"unread,omitempty"`

	// Draft email attributes
	TimeUpdated string   `json:"timeUpdated,omitempty"`
	Cc          []string `json:"cc,omitempty"`
	Bcc         []string `json:"bcc,omitempty"`

	// Sent email attributes
	TimeSent string `json:"timeSent,omitempty"`

	// Attachment attributes, currently only support
	Attachments *types.Files `json:"attachments,omitempty"`
	Inlines     *types.Files `json:"inlines,omitempty"`
}

type EmailVerdict struct {
	Spam  bool `json:"spam"`
	DKIM  bool `json:"dkim"`
	DMARC bool `json:"dmarc"`
	SPF   bool `json:"spf"`
	Virus bool `json:"virus"`
}

// Get returns the email and marks it as read
func GetAndRead(ctx context.Context, api GetEmailAPI, messageID string) (*GetResult, error) {
	result, err := Get(ctx, api, messageID)
	if err != nil {
		return nil, err
	}

	// mark email as read
	if result.Type == EmailTypeInbox && result.Unread != nil && *result.Unread {
		err = Read(ctx, api, messageID, ActionRead)
		if err != nil {
			return nil, err
		}
		fmt.Println("email marked as read")
	}

	return result, nil
}

// get returns the email
func Get(ctx context.Context, api GetItemAPI, messageID string) (*GetResult, error) {
	resp, err := api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
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

	// for backward compatibility, ReplyTo may be in string format,
	// then we need to convert it to string set
	if replyTo, ok := resp.Item["ReplyTo"]; ok {
		if _, ok := replyTo.(*dynamodbTypes.AttributeValueMemberS); ok {
			resp.Item["ReplyTo"] = &dynamodbTypes.AttributeValueMemberSS{
				Value: []string{replyTo.(*dynamodbTypes.AttributeValueMemberS).Value},
			}
		}
	}

	result, err := ParseGetResult(resp.Item)
	if err != nil {
		return nil, err
	}

	fmt.Println("get method finished successfully")
	return result, nil
}

func ParseGetResult(attributeValues map[string]dynamodbTypes.AttributeValue) (*GetResult, error) {
	result := new(GetResult)
	err := attributevalue.UnmarshalMap(attributeValues, result)
	if err != nil {
		return nil, err
	}

	var emailTime string
	result.Type, emailTime, err = UnmarshalGSI(attributeValues)
	if err != nil {
		return nil, err
	}

	if result.Type == EmailTypeInbox {
		result.TimeReceived = emailTime
		if result.Unread == nil {
			unread := false
			result.Unread = &unread
		}
	} else {
		if result.Type == EmailTypeDraft {
			result.TimeUpdated = emailTime
		} else if result.Type == EmailTypeSent {
			result.TimeSent = emailTime
		}
		result.Unread = nil
	}

	return result, nil
}
