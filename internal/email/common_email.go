package email

import (
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Input struct {
	MessageID  string   `json:"messageID"`
	Subject    string   `json:"subject"`
	From       []string `json:"from"`
	To         []string `json:"to"`
	Cc         []string `json:"cc"`
	Bcc        []string `json:"bcc"`
	ReplyTo    []string `json:"replyTo"`
	InReplyTo  string
	References string
	Text       string `json:"text"`
	HTML       string `json:"html"`
	ThreadID   string `json:"threadID,omitempty"`
}

// GenerateAttributes generates DynamoDB AttributeValues
func (e Input) GenerateAttributes(typeYearMonth, dateTime string) map[string]dynamodbTypes.AttributeValue {
	item := map[string]dynamodbTypes.AttributeValue{
		"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: e.MessageID},
		"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: typeYearMonth},
		"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: dateTime},
		"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: e.Subject},
		"Text":          &dynamodbTypes.AttributeValueMemberS{Value: e.Text},
		"HTML":          &dynamodbTypes.AttributeValueMemberS{Value: e.HTML},
	}

	if len(e.From) > 0 {
		item["From"] = &dynamodbTypes.AttributeValueMemberSS{Value: e.From}
	}
	if len(e.To) > 0 {
		item["To"] = &dynamodbTypes.AttributeValueMemberSS{Value: e.To}
	}
	if len(e.Cc) > 0 {
		item["Cc"] = &dynamodbTypes.AttributeValueMemberSS{Value: e.Cc}
	}
	if len(e.Bcc) > 0 {
		item["Bcc"] = &dynamodbTypes.AttributeValueMemberSS{Value: e.Bcc}
	}
	if len(e.ReplyTo) > 0 {
		item["ReplyTo"] = &dynamodbTypes.AttributeValueMemberSS{Value: e.ReplyTo}
	}
	if e.InReplyTo != "" {
		item["InReplyTo"] = &dynamodbTypes.AttributeValueMemberS{Value: e.InReplyTo}
	}
	if e.References != "" {
		item["References"] = &dynamodbTypes.AttributeValueMemberS{Value: e.References}
	}
	if e.ThreadID != "" {
		item["ThreadID"] = &dynamodbTypes.AttributeValueMemberS{Value: e.ThreadID}
	}

	return item
}
