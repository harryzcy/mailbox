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

func TestStoreEmails(t *testing.T) {
	testStoreEmails_NoThread(t)
	testStoreEmails_BasicThread(t)
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
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "1@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01-00:00:00"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	// second email, no In-Reply-To or References
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "2"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "2@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01-00:00:00"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	// third email, with In-Reply-To and References, but they don't exist
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "3"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "3@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01-00:00:00"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})

	testItemExists(t, "1")
	testItemExists(t, "2")
	testItemExists(t, "3")
}

func testStoreEmails_BasicThread(t *testing.T) {
	defer deleteAllItems()

	if num := testEmptyTable(t); num != 0 {
		return
	}

	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "1"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "1@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01-00:00:00"},
			"Subject":           &types.AttributeValueMemberS{Value: "Subject 1"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	testItemExists(t, "1")
	testItemNoAttribute(t, "1", "IsThreadLatest") // no thread yet

	// should create a new thread
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "2"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "2@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01-00:00:00"},
			"Subject":           &types.AttributeValueMemberS{Value: "Subject 2"},
		},
		InReplyTo:    "1@example.com",
		References:   "1@example.com",
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	testItemExists(t, "2")
	testItemNoAttribute(t, "1", "IsThreadLatest")
	testItemHasAttribute(t, "2", "IsThreadLatest", &types.AttributeValueMemberBOOL{Value: true})

	// should add to the same thread
	email.StoreEmail(context.TODO(), client, &email.StoreEmailInput{
		Item: map[string]types.AttributeValue{
			"MessageID":         &types.AttributeValueMemberS{Value: "3"},
			"OriginalMessageID": &types.AttributeValueMemberS{Value: "3@example.com"},
			"TypeYearMonth":     &types.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &types.AttributeValueMemberS{Value: "01-00:00:00"},
			"Subject":           &types.AttributeValueMemberS{Value: "Subject 3"},
		},
		InReplyTo:    "2@example.com",
		References:   "2@example.com",
		TimeReceived: "2023-02-01T00:00:00Z",
	})

	testItemExists(t, "1")
	testItemNoAttribute(t, "1", "IsThreadLatest")
	testItemExists(t, "2")
	testItemNoAttribute(t, "2", "IsThreadLatest")
	testItemExists(t, "3")
	testItemHasAttribute(t, "3", "IsThreadLatest", &types.AttributeValueMemberBOOL{Value: true})

	resp, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	assert.NoError(t, err)

	assert.Equal(t, 4, len(resp.Items))

	threadID := ""
	for _, item := range resp.Items {
		messageID := item["MessageID"].(*types.AttributeValueMemberS).Value
		if len(messageID) == 32 {
			// is thread
			threadID = messageID

			ids := item["EmailIDs"].(*types.AttributeValueMemberL).Value
			assert.Equal(t, 3, len(ids))
			assert.Equal(t, "1", ids[0].(*types.AttributeValueMemberS).Value)
			assert.Equal(t, "2", ids[1].(*types.AttributeValueMemberS).Value)
			assert.Equal(t, "3", ids[2].(*types.AttributeValueMemberS).Value)

			assert.Equal(t, "Subject 1", item["Subject"].(*types.AttributeValueMemberS).Value)
			break
		}
	}

	for _, item := range resp.Items {
		messageID := item["MessageID"].(*types.AttributeValueMemberS).Value
		if len(messageID) == 32 {
			// is thread
			continue
		}
		assert.Equal(t, threadID, item["ThreadID"].(*types.AttributeValueMemberS).Value)
	}
}

func testItemExists(t *testing.T, messageID string) {
	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Item)
}

func testItemNoAttribute(t *testing.T, messageID, attribute string) {
	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Item)
	assert.Nil(t, resp.Item[attribute])
}

func testItemHasAttribute(t *testing.T, messageID, attribute string, value types.AttributeValue) {
	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"MessageID": &types.AttributeValueMemberS{Value: messageID},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Item)
	assert.Equal(t, value, resp.Item[attribute])
}
