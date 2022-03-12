package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/harryzcy/mailbox/internal/util/format"
)

// GetResult represents the result of get method
type GetResult struct {
	TimeIndex
	Subject     string   `json:"subject"`
	DateSent    string   `json:"dateSent"`
	Source      string   `json:"source"`
	Destination []string `json:"destination"`
	From        []string `json:"from"`
	To          []string `json:"to"`
	ReturnPath  string   `json:"returnPath"`
	Text        string   `json:"text"`
	HTML        string   `json:"html"`
}

// Get returns the email
func Get(ctx context.Context, cfg aws.Config, messageID string) (*GetResult, error) {
	svc := dynamodb.NewFromConfig(cfg)
	resp, err := svc.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"messageID": &types.AttributeValueMemberS{Value: messageID},
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

	result.Type, result.TimeReceived, err = unmarshalGSI(resp.Item)
	if err != nil {
		return nil, err
	}

	fmt.Println("get method finished successfully")
	return result, nil
}

func unmarshalGSI(item map[string]types.AttributeValue) (emailType, timeReceived string, err error) {
	var typeYearMonth string
	var dt string // date-time
	err = attributevalue.Unmarshal(item["type-year-month"], &typeYearMonth)
	if err != nil {
		fmt.Printf("unmarshal type-year-month failed: %v", err)
		return
	}
	err = attributevalue.Unmarshal(item["date-time"], &dt)
	if err != nil {
		fmt.Printf("unmarshal date-time failed: %v", err)
		return
	}
	return parseGSI(typeYearMonth, dt)
}

func parseGSI(typeYearMonth, dt string) (emailType, timeReceived string, err error) {
	var ym string // YYYY-MM
	emailType, ym, err = format.ExtractTypeYearMonth(typeYearMonth)
	if err != nil {
		fmt.Printf("extract type-year-month failed: %v\n", err)
		return
	}
	timeReceived = format.RejoinDate(ym, dt)
	return
}
