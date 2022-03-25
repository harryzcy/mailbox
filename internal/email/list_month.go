package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// List lists emails in DynamoDB
func ListMonth(ctx context.Context, api QueryAPI, year, month string) (*ListResult, error) {
	if len(month) == 1 {
		month = "0" + month
	}
	if len(year) != 4 || len(month) != 2 {
		return nil, ErrInvalidInput
	}
	typeYearMonth := "inbox#" + year + "-" + month
	fmt.Println("querying for TypeYearMonth:", typeYearMonth)

	keyConditionExpression := "#tym = :val"
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeValues[":val"] = &types.AttributeValueMemberS{Value: typeYearMonth}
	projectionExpression := map[string]string{
		"#tym": "TypeYearMonth",
	}

	resp, err := api.Query(ctx, &dynamodb.QueryInput{
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

	result := &ListResult{Count: int(resp.Count), Items: items}
	fmt.Printf("Count: %d\n", resp.Count)
	fmt.Printf("LastEvaluatedKey: %+v\n", resp.LastEvaluatedKey)
	return result, nil
}
