package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetResult represents the result of get method
type GetResult struct {
	MessageID string   `json:"messageID"`
	Type      string   `json:"type"`
	Subject   string   `json:"subject"`
	From      []string `json:"from"`
	To        []string `json:"to"`
	Text      string   `json:"text"`
	HTML      string   `json:"html"`

	// Inbox email attributes
	TimeReceived string   `json:"timeReceived,omitempty"`
	DateSent     string   `json:"dateSent,omitempty"`
	Source       string   `json:"source,omitempty"`
	Destination  []string `json:"destination,omitempty"`
	ReturnPath   string   `json:"returnPath,omitempty"`

	// Draft email attributes
	TimeUpdated string   `json:"timeUpdated,omitempty"`
	Cc          []string `json:"cc,omitempty"`
	Bcc         []string `json:"bcc,omitempty"`
	ReplyTo     []string `json:"replyTo,omitempty"`
}

// Get returns the email
func Get(ctx context.Context, api GetItemAPI, messageID string) (*GetResult, error) {
	resp, err := api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Item) == 0 {
		return nil, ErrNotFound
	}
	result := new(GetResult)
	err = attributevalue.UnmarshalMap(resp.Item, result)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var emailTime string
	result.Type, emailTime, err = unmarshalGSI(resp.Item)
	if err != nil {
		return nil, err
	}

	if result.Type == EmailTypeInbox {
		result.TimeReceived = emailTime
	} else {
		result.TimeUpdated = emailTime
	}

	fmt.Println("get method finished successfully")
	return result, nil
}
