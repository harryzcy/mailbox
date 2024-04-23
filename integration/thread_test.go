package integration

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/thread"
	"github.com/stretchr/testify/assert"
)

func checkEmptyTable(t *testing.T) int {
	resp, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(env.TableName),
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Items))
	return len(resp.Items)
}

func TestStoreEmails_NoThread(t *testing.T) {
	defer deleteAllItems()

	if num := checkEmptyTable(t); num != 0 {
		return
	}

	// first email
	thread.StoreEmail(context.TODO(), client, &thread.StoreEmailInput{
		Item: map[string]dynamodbTypes.AttributeValue{
			"MessageID":         &dynamodbTypes.AttributeValueMemberS{Value: "1"},
			"OriginalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: "1@example.com"},
			"TypeYearMonth":     &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &dynamodbTypes.AttributeValueMemberS{Value: "01-00:00:00"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	// second email, no In-Reply-To or References
	thread.StoreEmail(context.TODO(), client, &thread.StoreEmailInput{
		Item: map[string]dynamodbTypes.AttributeValue{
			"MessageID":         &dynamodbTypes.AttributeValueMemberS{Value: "2"},
			"OriginalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: "2@example.com"},
			"TypeYearMonth":     &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &dynamodbTypes.AttributeValueMemberS{Value: "01-00:00:00"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	// third email, with In-Reply-To and References, but they don't exist
	thread.StoreEmail(context.TODO(), client, &thread.StoreEmailInput{
		Item: map[string]dynamodbTypes.AttributeValue{
			"MessageID":         &dynamodbTypes.AttributeValueMemberS{Value: "3"},
			"OriginalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: "3@example.com"},
			"TypeYearMonth":     &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &dynamodbTypes.AttributeValueMemberS{Value: "01-00:00:00"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})

	testItemExists(t, "1")
	testItemExists(t, "2")
	testItemExists(t, "3")
}

func TestStoreEmails_BasicThread(t *testing.T) {
	defer deleteAllItems()

	if num := checkEmptyTable(t); num != 0 {
		return
	}

	thread.StoreEmail(context.TODO(), client, &thread.StoreEmailInput{
		Item: map[string]dynamodbTypes.AttributeValue{
			"MessageID":         &dynamodbTypes.AttributeValueMemberS{Value: "1"},
			"OriginalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: "1@example.com"},
			"TypeYearMonth":     &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &dynamodbTypes.AttributeValueMemberS{Value: "01-00:00:00"},
			"Subject":           &dynamodbTypes.AttributeValueMemberS{Value: "Subject 1"},
		},
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	testItemExists(t, "1")
	testItemNoAttribute(t, "1", "IsThreadLatest") // no thread yet

	// should create a new thread
	thread.StoreEmail(context.TODO(), client, &thread.StoreEmailInput{
		Item: map[string]dynamodbTypes.AttributeValue{
			"MessageID":         &dynamodbTypes.AttributeValueMemberS{Value: "2"},
			"OriginalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: "2@example.com"},
			"TypeYearMonth":     &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &dynamodbTypes.AttributeValueMemberS{Value: "01-00:00:00"},
			"Subject":           &dynamodbTypes.AttributeValueMemberS{Value: "Subject 2"},
		},
		InReplyTo:    "1@example.com",
		References:   "1@example.com",
		TimeReceived: "2023-02-01T00:00:00Z",
	})
	testItemExists(t, "2")
	testItemNoAttribute(t, "1", "IsThreadLatest")
	testItemHasAttribute(t, "2", "IsThreadLatest", &dynamodbTypes.AttributeValueMemberBOOL{Value: true})

	// should add to the same thread
	thread.StoreEmail(context.TODO(), client, &thread.StoreEmailInput{
		Item: map[string]dynamodbTypes.AttributeValue{
			"MessageID":         &dynamodbTypes.AttributeValueMemberS{Value: "3"},
			"OriginalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: "3@example.com"},
			"TypeYearMonth":     &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2023-01"},
			"DateTime":          &dynamodbTypes.AttributeValueMemberS{Value: "01-00:00:00"},
			"Subject":           &dynamodbTypes.AttributeValueMemberS{Value: "Subject 3"},
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
	testItemHasAttribute(t, "3", "IsThreadLatest", &dynamodbTypes.AttributeValueMemberBOOL{Value: true})

	resp, err := client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(env.TableName),
	})
	assert.NoError(t, err)

	assert.Equal(t, 4, len(resp.Items))

	threadID := ""
	for _, item := range resp.Items {
		messageID := item["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value
		if len(messageID) == 32 {
			// is thread
			threadID = messageID

			ids := item["EmailIDs"].(*dynamodbTypes.AttributeValueMemberL).Value
			assert.Equal(t, 3, len(ids))
			assert.Equal(t, "1", ids[0].(*dynamodbTypes.AttributeValueMemberS).Value)
			assert.Equal(t, "2", ids[1].(*dynamodbTypes.AttributeValueMemberS).Value)
			assert.Equal(t, "3", ids[2].(*dynamodbTypes.AttributeValueMemberS).Value)

			assert.Equal(t, "Subject 1", item["Subject"].(*dynamodbTypes.AttributeValueMemberS).Value)
			break
		}
	}

	for _, item := range resp.Items {
		messageID := item["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value
		if len(messageID) == 32 {
			// is thread
			continue
		}
		assert.Equal(t, threadID, item["ThreadID"].(*dynamodbTypes.AttributeValueMemberS).Value)
	}
}

func testItemExists(t *testing.T, messageID string) {
	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Item)
}

func testItemNoAttribute(t *testing.T, messageID, attribute string) {
	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Item)
	assert.Nil(t, resp.Item[attribute])
}

func testItemHasAttribute(t *testing.T, messageID, attribute string, value dynamodbTypes.AttributeValue) {
	resp, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Item)
	assert.Equal(t, value, resp.Item[attribute])
}
