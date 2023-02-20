package integration

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/stretchr/testify/assert"
)

func TestTrivial(t *testing.T) {
	assert.True(t, true)
}

func TestStoreEmails(t *testing.T) {
	testStoreEmails_NoThread(t)
}

func testEmptyTable(t *testing.T) int {
	resp, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Items))
	return len(resp.Items)
}

func testStoreEmails_NoThread(t *testing.T) {
	defer deleteAllItems()

	if num := testEmptyTable(t); num != 0 {
		return
	}

	// first email
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "1"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "3@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01T00:00:00Z"},
		},
	})
	// second email, no In-Reply-To or References
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "2"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "2@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01T00:00:00Z"},
		},
	})
	// third email, with In-Reply-To and References, but they don't exist
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "3"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "non-exist@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01T00:00:00Z"},
		},
	})
}
