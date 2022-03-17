package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// CreateInput represents the input of create method
type CreateInput struct {
	Subject string   `json:"subject"`
	From    []string `json:"from"`
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	ReplyTo []string `json:"replyTo"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
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

var getCreatedTime = func() time.Time {
	return time.Now().UTC()
}

// Create adds an email as draft in DynamoDB
func Create(ctx context.Context, api PutItemAPI, input CreateInput) (*CreateResult, error) {
	messageID := generateMessageID()
	now := getCreatedTime()
	typeYearMonth := EmailTypeDraft + "#" + now.Format("2006-01")
	dateTime := now.Format("02-15:04:05")

	item := map[string]types.AttributeValue{
		"MessageID":     &types.AttributeValueMemberS{Value: messageID},
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
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return nil, err
	}

	result := &CreateResult{
		TimeIndex: TimeIndex{
			MessageID:   messageID,
			Type:        EmailTypeDraft,
			TimeCreated: now.Format(time.RFC3339),
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
