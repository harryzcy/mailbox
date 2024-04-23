package email

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

// TimeIndex represents the index attributes of an email
type TimeIndex struct {
	MessageID string `json:"messageID"`
	Type      string `json:"type"`

	// TimeReceived is used by inbox emails
	TimeReceived string `json:"timeReceived,omitempty"`

	// TimeUpdated is used by draft emails
	TimeUpdated string `json:"timeUpdated,omitempty"`

	// TimeSent is used by sent emails
	TimeSent string `json:"timeSent,omitempty"`
}

// GSIIndex represents Global Secondary Index of an email
type GSIIndex struct {
	MessageID     string `dynamodbav:"MessageID"`
	TypeYearMonth string `dynamodbav:"TypeYearMonth"`
	DateTime      string `dynamodbav:"DateTime"`
}

// ToTimeIndex returns TimeIndex
func (gsi GSIIndex) ToTimeIndex() (*TimeIndex, error) {
	index := &TimeIndex{
		MessageID: gsi.MessageID,
	}

	var emailTime string
	var err error
	index.Type, emailTime, err = parseGSI(gsi.TypeYearMonth, gsi.DateTime)
	if err != nil {
		return nil, err
	}

	switch index.Type {
	case types.EmailTypeInbox:
		index.TimeReceived = emailTime
	case types.EmailTypeSent:
		index.TimeSent = emailTime
	case types.EmailTypeDraft:
		index.TimeUpdated = emailTime
	}
	return index, nil
}

func UnmarshalGSI(item map[string]dynamodbTypes.AttributeValue) (emailType, emailTime string, err error) {
	var typeYearMonth string
	var dt string // date-time
	err = attributevalue.Unmarshal(item["TypeYearMonth"], &typeYearMonth)
	if err != nil {
		fmt.Printf("unmarshal TypeYearMonth failed: %v", err)
		return
	}
	err = attributevalue.Unmarshal(item["DateTime"], &dt)
	if err != nil {
		fmt.Printf("unmarshal DateTime failed: %v", err)
		return
	}
	return parseGSI(typeYearMonth, dt)
}

func parseGSI(typeYearMonth, dt string) (emailType, emailTime string, err error) {
	var ym string // YYYY-MM
	emailType, ym, err = format.ExtractTypeYearMonth(typeYearMonth)
	if err != nil {
		fmt.Printf("extract TypeYearMonth failed: %v\n", err)
		return
	}
	emailTime = format.RejoinDate(ym, dt)
	return
}

type Item struct {
	TimeIndex
	Subject        string   `json:"subject"`
	From           []string `json:"from"`
	To             []string `json:"to"`
	Unread         *bool    `json:"unread,omitempty"`
	ThreadID       string   `json:"threadID,omitempty"`
	IsThreadLatest bool     `json:"isThreadLatest,omitempty"`
}

type RawEmailItem struct {
	GSIIndex
	Subject        string
	From           []string `json:"from"`
	To             []string `json:"to"`
	Unread         *bool    `json:"unread,omitempty"`
	ThreadID       string   `json:"threadID,omitempty"`
	IsThreadLatest bool     `json:"isThreadLatest,omitempty"`
}

func (raw RawEmailItem) ToEmailItem() (*Item, error) {
	index, err := raw.GSIIndex.ToTimeIndex()
	if err != nil {
		return nil, err
	}
	item := &Item{
		TimeIndex:      *index,
		Subject:        raw.Subject,
		From:           raw.From,
		To:             raw.To,
		Unread:         raw.Unread,
		ThreadID:       raw.ThreadID,
		IsThreadLatest: raw.IsThreadLatest,
	}
	if item.Unread == nil && item.Type == types.EmailTypeInbox {
		item.Unread = new(bool)
	}

	return item, nil
}

type OriginalMessageIDIndex struct {
	MessageID         string
	OriginalMessageID string
}
