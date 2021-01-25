package email

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var tableName = os.Getenv("DYNAMODB_TABLE")

// TimeIndex represents the index info of an email
type TimeIndex struct {
	MessageID    string `json:"messageID"`
	TimeReceived string `json:"timeReceived"`
}

// ListResult represents the result of list method
type ListResult struct {
	Count            int32       `json:"count"`
	Items            []TimeIndex `json:"items"`
	LastEvaluatedKey string      `json:"lastEvaluatedKey"`
}

// GetResult represents the result of get method
type GetResult struct {
	MessageID    string   `json:"messageID"`
	Subject      string   `json:"subject"`
	DateSent     string   `json:"dateSent"`
	TimeReceived string   `json:"timeReceived"`
	Source       string   `json:"source"`
	Destination  []string `json:"destination"`
	From         []string `json:"from"`
	To           []string `json:"to"`
	ReturnPath   string   `json:"returnPath"`
	Text         string   `json:"text"`
	HTML         string   `json:"html"`
}

// List lists emails in DynamoDB
func List(cfg aws.Config, year, month int) (*ListResult, error) {
	ym := fmt.Sprintf("%04d-%02d", year, month)
	fmt.Println("requesting for year-month:", ym)
	indexName := "timeIndex"
	keyConditionExpression := "ymReceived = :ym"
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeValues[":ym"] = &types.AttributeValueMemberS{Value: ym}

	svc := dynamodb.NewFromConfig(cfg)
	resp, err := svc.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:                 &tableName,
		IndexName:                 &indexName,
		KeyConditionExpression:    &keyConditionExpression,
		ExpressionAttributeValues: expressionAttributeValues,
	})
	if err != nil {
		return nil, err
	}
	result := &ListResult{Count: resp.Count, Items: make([]TimeIndex, resp.Count)}
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &result.Items)
	if err != nil {
		fmt.Println("unmarshal failed,", err)
		return nil, err
	}

	fmt.Printf("Count: %d\n", resp.Count)
	fmt.Printf("LastEvaluatedKey: %+v\n", resp.LastEvaluatedKey)

	return result, nil
}

// Get returns the email
func Get(cfg aws.Config, messageID string) (*GetResult, error) {
	svc := dynamodb.NewFromConfig(cfg)
	key := make(map[string]types.AttributeValue)
	key["messageID"] = &types.AttributeValueMemberS{Value: messageID}
	resp, err := svc.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: &tableName,
		Key:       key,
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

	fmt.Println("get method finished successfully")
	return result, nil
}
