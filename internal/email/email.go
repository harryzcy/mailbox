package email

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

var (
	// tableName represents DynamoDB table name
	tableName = os.Getenv("DYNAMODB_TABLE")
	// gsiIndexName represents DynamoDB's GSI name
	gsiIndexName = os.Getenv("DYNAMODB_TIME_INDEX")
)

// TimeIndex represents time attributes of an email
type TimeIndex struct {
	MessageID    string `json:"messageID"`
	Type         string `json:"type"`
	TimeReceived string `json:"timeReceived"`
}

// GSIIndex represents Global Secondary Index of an email
type GSIIndex struct {
	MessageID     string `dynamodbav:"messageID"`
	TypeYearMonth string `dynamodbav:"type-year-month"`
	DateTime      string `dynamodbav:"date-time"`
}

// ToTimeIndex returns TimeIndex
func (gsi GSIIndex) ToTimeIndex() (*TimeIndex, error) {
	timeIndex := new(TimeIndex)
	var err error

	timeIndex.MessageID = gsi.MessageID
	timeIndex.Type, timeIndex.TimeReceived, err = parseGSI(gsi.TypeYearMonth, gsi.DateTime)
	return timeIndex, err
}

// ListResult represents the result of list method
type ListResult struct {
	Count            int32       `json:"count"`
	Items            []TimeIndex `json:"items"`
	LastEvaluatedKey string      `json:"lastEvaluatedKey"`
}

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

// List lists emails in DynamoDB
func List(ctx context.Context, cfg aws.Config, year, month string) (*ListResult, error) {
	if len(month) == 1 {
		month = "0" + month
	}
	if len(year) != 4 || len(month) != 2 {
		return nil, ErrInvalidInput
	}
	typeYearMonth := "inbox#" + year + "-" + month
	fmt.Println("querying for type-year-month:", typeYearMonth)

	keyConditionExpression := "#tym = :val"
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeValues[":val"] = &types.AttributeValueMemberS{Value: typeYearMonth}
	projectionExpression := map[string]string{
		"#tym": "type-year-month",
	}

	svc := dynamodb.NewFromConfig(cfg)
	resp, err := svc.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &tableName,
		IndexName:                 &gsiIndexName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  projectionExpression,
	})
	if err != nil {
		return nil, err
	}
	var rawItems []GSIIndex
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &rawItems)
	if err != nil {
		fmt.Printf("unmarshal failed: %v\n", err)
		return nil, err
	}

	items := make([]TimeIndex, len(rawItems))
	for i, rawItem := range rawItems {
		var item *TimeIndex
		item, err = rawItem.ToTimeIndex()
		if err != nil {
			fmt.Printf("converting to time index failed: %v\n", err)
			return nil, err
		}
		items[i] = *item
	}

	result := &ListResult{Count: resp.Count, Items: items}
	fmt.Printf("Count: %d\n", resp.Count)
	fmt.Printf("LastEvaluatedKey: %+v\n", resp.LastEvaluatedKey)
	return result, nil
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
