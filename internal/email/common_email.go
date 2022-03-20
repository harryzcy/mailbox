package email

import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

type EmailInput struct {
	MessageID string   `json:"messageID"`
	Subject   string   `json:"subject"`
	From      []string `json:"from"`
	To        []string `json:"to"`
	Cc        []string `json:"cc"`
	Bcc       []string `json:"bcc"`
	ReplyTo   []string `json:"replyTo"`
	Text      string   `json:"text"`
	HTML      string   `json:"html"`
}

// GenerateAttributes generates DynamoDB AttributeValues
func (e EmailInput) GenerateAttributes(hasMessageID bool, typeYearMonth, dateTime string) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"TypeYearMonth": &types.AttributeValueMemberS{Value: typeYearMonth},
		"DateTime":      &types.AttributeValueMemberS{Value: dateTime},
		"Subject":       &types.AttributeValueMemberS{Value: e.Subject},
		"Text":          &types.AttributeValueMemberS{Value: e.Text},
		"HTML":          &types.AttributeValueMemberS{Value: e.HTML},
	}

	if hasMessageID {
		item["MessageID"] = &types.AttributeValueMemberS{Value: e.MessageID}
	}

	if e.From != nil && len(e.From) > 0 {
		item["From"] = &types.AttributeValueMemberSS{Value: e.From}
	}
	if e.To != nil && len(e.To) > 0 {
		item["To"] = &types.AttributeValueMemberSS{Value: e.To}
	}
	if e.Cc != nil && len(e.Cc) > 0 {
		item["Cc"] = &types.AttributeValueMemberSS{Value: e.Cc}
	}
	if e.Bcc != nil && len(e.Bcc) > 0 {
		item["Bcc"] = &types.AttributeValueMemberSS{Value: e.Bcc}
	}
	if e.ReplyTo != nil && len(e.ReplyTo) > 0 {
		item["ReplyTo"] = &types.AttributeValueMemberSS{Value: e.ReplyTo}
	}

	return item
}
