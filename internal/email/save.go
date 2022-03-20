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
	"github.com/harryzcy/mailbox/internal/util/format"
)

// SaveInput represents the input of save method
type SaveInput struct {
	MessageID string `json:"messageID"`
	EmailInput
}

// SaveResult represents the result of save method
type SaveResult struct {
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

var getUpdatedTime = func() time.Time {
	return time.Now().UTC()
}

// Save puts an email as draft in DynamoDB
func Save(ctx context.Context, api PutItemAPI, input SaveInput) (*SaveResult, error) {
	if !strings.HasPrefix(input.MessageID, "draft-") {
		return nil, ErrEmailIsNotDraft
	}

	now := getUpdatedTime()
	typeYearMonth, _ := format.FormatTypeYearMonth(EmailTypeDraft, now)
	dateTime := format.FormatDateTime(now)

	item := map[string]types.AttributeValue{
		"MessageID":     &types.AttributeValueMemberS{Value: input.MessageID},
		"TypeYearMonth": &types.AttributeValueMemberS{Value: typeYearMonth},
		"DateTime":      &types.AttributeValueMemberS{Value: dateTime},
		"Subject":       &types.AttributeValueMemberS{Value: input.Subject},
		"Text":          &types.AttributeValueMemberS{Value: input.Text},
		"HTML":          &types.AttributeValueMemberS{Value: input.HTML},
	}
	if input.From != nil && len(input.From) > 0 {
		item["From"] = &types.AttributeValueMemberSS{Value: input.From}
	}
	if input.To != nil && len(input.To) > 0 {
		item["To"] = &types.AttributeValueMemberSS{Value: input.To}
	}
	if input.Cc != nil && len(input.Cc) > 0 {
		item["Cc"] = &types.AttributeValueMemberSS{Value: input.Cc}
	}
	if input.Bcc != nil && len(input.Bcc) > 0 {
		item["Bcc"] = &types.AttributeValueMemberSS{Value: input.Bcc}
	}
	if input.ReplyTo != nil && len(input.ReplyTo) > 0 {
		item["ReplyTo"] = &types.AttributeValueMemberSS{Value: input.ReplyTo}
	}

	_, err := api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("MessageID = :messageID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":messageID": &types.AttributeValueMemberS{Value: input.MessageID},
		},
	})
	if err != nil {
		var condFailedErr *types.ConditionalCheckFailedException
		if errors.As(err, &condFailedErr) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := &SaveResult{
		TimeIndex: TimeIndex{
			MessageID:   input.MessageID,
			Type:        EmailTypeDraft,
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

	fmt.Println("save method finished successfully")
	return result, nil
}
