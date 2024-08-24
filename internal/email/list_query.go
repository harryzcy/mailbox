package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
)

// listQueryInput represents the inputs for listByYearMonth function
type listQueryInput struct {
	emailType        string
	year             string
	month            string
	order            string
	showTrash        string
	pageSize         int
	lastEvaluatedKey map[string]dynamodbTypes.AttributeValue
}

// unmarshalListOfMaps will be mocked during testing
var unmarshalListOfMaps = attributevalue.UnmarshalListOfMaps

// listQueryResult contains the items and lastEvaluatedKey returned from Query operation
type listQueryResult struct {
	items            []Item
	lastEvaluatedKey map[string]dynamodbTypes.AttributeValue
	hasMore          bool
}

// listByYearMonth returns a list of emails within a DynamoDB partition.
// This is an low level function call that directly uses AWS sdk.
func listByYearMonth(ctx context.Context, client api.QueryAPI, input listQueryInput) (listQueryResult, error) {
	typeYearMonth := input.emailType + "#" + input.year + "-" + input.month

	fmt.Println("querying for TypeYearMonth:", typeYearMonth)

	var limit *int32
	if input.pageSize > 0 {
		limit = aws.Int32(int32(input.pageSize))
	}

	queryInput := &dynamodb.QueryInput{
		TableName:              &env.TableName,
		IndexName:              &env.GsiIndexName,
		ExclusiveStartKey:      input.lastEvaluatedKey,
		KeyConditionExpression: aws.String("#tym = :val"),
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":val": &dynamodbTypes.AttributeValueMemberS{Value: typeYearMonth},
		},
		ExpressionAttributeNames: map[string]string{
			"#tym": "TypeYearMonth",
		},
		Limit:            limit,
		ScanIndexForward: aws.Bool(false), // reverse order
	}
	if input.showTrash == ShowTrashExclude {
		queryInput.FilterExpression = aws.String("attribute_not_exists(TrashedTime)")
	} else if input.showTrash == ShowTrashOnly {
		queryInput.FilterExpression = aws.String("attribute_exists(TrashedTime)")
	}

	resp, err := client.Query(ctx, queryInput)
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return listQueryResult{}, api.ErrTooManyRequests
		}

		return listQueryResult{}, err
	}

	var rawItems []RawEmailItem
	err = unmarshalListOfMaps(resp.Items, &rawItems)
	if err != nil {
		fmt.Printf("unmarshal failed: %v\n", err)
		return listQueryResult{}, err
	}

	items := make([]Item, len(rawItems))
	for i, rawItem := range rawItems {
		var item *Item
		item, err = rawItem.ToEmailItem()
		if err != nil {
			fmt.Printf("converting to time index failed: %v\n", err)
			return listQueryResult{}, err
		}
		items[i] = *item
	}

	return listQueryResult{
		items:            items,
		lastEvaluatedKey: resp.LastEvaluatedKey,
		hasMore:          len(resp.LastEvaluatedKey) > 0,
	}, nil
}
