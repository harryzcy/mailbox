package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// listQueryInput represents the inputs for listByYearMonth function
type listQueryInput struct {
	emailType        string
	year             string
	month            string
	order            string
	pageSize         int
	lastEvaluatedKey map[string]types.AttributeValue
}

// unmarshalListOfMaps will be mocked during testing
var unmarshalListOfMaps = attributevalue.UnmarshalListOfMaps

// listQueryResult contains the items and lastEvaluatedKey returned from Query operation
type listQueryResult struct {
	items            []TimeIndex
	lastEvaluatedKey map[string]types.AttributeValue
	hasMore          bool
}

// listByYearMonth returns a list of emails within a DynamoDB partition.
// This is an low level function call that directly uses AWS sdk.
func listByYearMonth(ctx context.Context, api QueryAPI, input listQueryInput) (listQueryResult, error) {
	typeYearMonth := input.emailType + "#" + input.year + "-" + input.month

	fmt.Println("querying for TypeYearMonth:", typeYearMonth)

	var limit *int32
	if input.pageSize > 0 {
		limit = aws.Int32(int32(input.pageSize))
	}

	resp, err := api.Query(ctx, &dynamodb.QueryInput{
		TableName:              &tableName,
		IndexName:              &gsiIndexName,
		ExclusiveStartKey:      input.lastEvaluatedKey,
		KeyConditionExpression: aws.String("#tym = :val"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val": &types.AttributeValueMemberS{Value: typeYearMonth},
		},
		ExpressionAttributeNames: map[string]string{
			"#tym": "TypeYearMonth",
		},
		Limit:            limit,
		ScanIndexForward: aws.Bool(false), // reverse order
	})
	if err != nil {
		if apiErr := new(types.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return listQueryResult{}, ErrTooManyRequests
		}

		return listQueryResult{}, err
	}

	var rawItems []GSIIndex
	err = unmarshalListOfMaps(resp.Items, &rawItems)
	if err != nil {
		fmt.Printf("unmarshal failed: %v\n", err)
		return listQueryResult{}, err
	}

	items := make([]TimeIndex, len(rawItems))
	for i, rawItem := range rawItems {
		var item *TimeIndex
		item, err = rawItem.ToTimeIndex()
		if err != nil {
			fmt.Printf("converting to time index failed: %v\n", err)
			return listQueryResult{}, err
		}
		items[i] = *item
	}

	return listQueryResult{
		items:            items,
		lastEvaluatedKey: resp.LastEvaluatedKey,
		hasMore:          resp.LastEvaluatedKey != nil && len(resp.LastEvaluatedKey) > 0,
	}, nil
}
