package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

// CreateInput represents the input of create method
type CreateInput struct {
	EmailInput
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

// Create adds an email as draft in DynamoDB
func Create(ctx context.Context, api PutItemAPI, input CreateInput) (*CreateResult, error) {
	messageID := generateMessageID()
	now := getUpdatedTime()
	typeYearMonth := EmailTypeDraft + "#" + now.Format("2006-01")
	dateTime := now.Format("02-15:04:05")

	input.MessageID = messageID
	item := input.GenerateAttributes(typeYearMonth, dateTime)

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
