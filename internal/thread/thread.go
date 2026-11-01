package thread

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/email"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/model"
	"github.com/harryzcy/mailbox/internal/platform"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/harryzcy/mailbox/internal/util/idutil"
	"github.com/harryzcy/mailbox/internal/util/timeutil"
)

type Thread struct {
	MessageID   string   `json:"messageID"`
	Type        string   `json:"type"`    // always "thread"
	Subject     string   `json:"subject"` // The subject of the first email in the thread
	EmailIDs    []string `json:"emailIDs"`
	DraftID     string   `json:"draftID,omitempty"`
	TimeUpdated string   `json:"timeUpdated"`           // The time the last email is received or sent
	TrashedTime *string  `json:"trashedTime,omitempty"` // Time in RFC3339 format

	Emails []email.GetResult `json:"emails,omitempty"`
	Draft  *email.GetResult  `json:"draft,omitempty"`
}

func GetThread(ctx context.Context, client platform.GetItemAPI, messageID string) (*Thread, error) {
	return getThread(ctx, client, messageID, false)
}

// getThread gets a thread, using a strongly consistent read if consistentRead is true.
func getThread(ctx context.Context, client platform.GetItemAPI, messageID string, consistentRead bool) (*Thread, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	}
	if consistentRead {
		input.ConsistentRead = aws.Bool(true)
	}
	resp, err := client.GetItem(ctx, input)
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, platform.ErrTooManyRequests
		}
		return nil, err
	}
	if len(resp.Item) == 0 {
		return nil, platform.ErrNotFound
	}

	emailType, _, err := email.UnmarshalGSI(resp.Item)
	if err != nil {
		return nil, err
	}
	if emailType != "thread" {
		return nil, platform.ErrNotFound
	}

	result := &Thread{
		Type: "thread",
	}
	err = attributevalue.UnmarshalMap(resp.Item, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetThreadWithEmails(ctx context.Context, client platform.GetThreadWithEmailsAPI, messageID string) (*Thread, error) {
	thread, err := GetThread(ctx, client, messageID)
	if err != nil {
		return nil, err
	}
	thread.EmailIDs = uniqueEmailIDs(messageID, thread.EmailIDs)

	keys := []map[string]dynamodbTypes.AttributeValue{}
	for _, emailID := range thread.EmailIDs {
		keys = append(keys, map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: emailID},
		})
	}
	if thread.DraftID != "" {
		keys = append(keys, map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: thread.DraftID},
		})
	}

	resp, err := client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]dynamodbTypes.KeysAndAttributes{
			env.TableName: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return nil, platform.ErrTooManyRequests
		}
		return nil, err
	}

	orderMap := map[string]int{}
	for i, emailID := range thread.EmailIDs {
		orderMap[emailID] = i
	}

	thread.Emails = make([]email.GetResult, len(thread.EmailIDs))

	for _, item := range resp.Responses[env.TableName] {
		email, err := email.ParseGetResult(item)
		if err != nil {
			return nil, err
		}

		if email.MessageID == thread.DraftID {
			thread.Draft = email
		} else {
			thread.Emails[orderMap[email.MessageID]] = *email
		}
	}

	return thread, nil
}

// uniqueEmailIDs drops repeated email IDs, logging a warning if any are found.
func uniqueEmailIDs(threadID string, emailIDs []string) []string {
	seen := make(map[string]bool, len(emailIDs))
	unique := make([]string, 0, len(emailIDs))
	for _, emailID := range emailIDs {
		if !seen[emailID] {
			seen[emailID] = true
			unique = append(unique, emailID)
		}
	}
	if len(unique) < len(emailIDs) {
		log.Printf("thread %s has duplicate email IDs, %d ignored (expected only from receives before v2.3.0)\n", threadID, len(emailIDs)-len(unique))
	}
	return unique
}

type DetermineThreadInput struct {
	InReplyTo  string
	References string
}

type DetermineThreadOutput struct {
	ThreadID          string
	Exists            bool   // If true, the email belongs to an existing thread
	PreviousMessageID string // If Exists is true, the messageID of the last email in the thread
	EmailCount        int    // If Exists is true, the number of emails in the thread

	ShouldCreate    bool   // If true, a new thread should be created
	CreatingEmailID string // If ShouldCreate is true, the messageID of the first email in the thread
	CreatingSubject string // If ShouldCreate is true, the subject of the first email in the thread
	CreatingTime    string // If ShouldCreate is true, the time the first email is received
}

// DetermineThread determines which thread an incoming email belongs to.
// If a thread already exists, the ThreadID is returned and Exists is true.
// If a thread does not exist and a new thread should be created, the ThreadID is randomly generated and ShouldCreate is true.
//
// TODO: refactor this function
//
//gocyclo:ignore
func DetermineThread(ctx context.Context, client platform.QueryAndGetItemAPI, input *DetermineThreadInput) (*DetermineThreadOutput, error) {
	fmt.Println("Determining thread...")
	originalMessageID := ""
	if len(input.InReplyTo) > 0 {
		originalMessageID = input.InReplyTo
	} else if len(input.References) > 0 {
		references := strings.Split(input.References, " ")
		originalMessageID = references[len(references)-1] // The last messageID in the References header
	}

	if originalMessageID == "" {
		return &DetermineThreadOutput{}, nil
	}

	sesDomain := env.Region + ".amazonses.com"
	var possibleSentID string
	if strings.HasSuffix(originalMessageID, "@"+sesDomain+">") {
		fmt.Println("incoming email is replying to a SES email")
		// If the messageID ends with @<SES domain>, it maybe a messageID of a sent email.
		// In this case, we need check if there's a corresponding sent email.
		possibleSentID = strings.TrimSuffix(originalMessageID, "@"+sesDomain+">")
		possibleSentID = strings.TrimPrefix(possibleSentID, "<")
	}

	var previousEmail *email.GetResult
	var err error
	if possibleSentID != "" {
		// Check if the messageID is a sent email first
		fmt.Println("checking possible sent email")
		previousEmail, err = email.Get(ctx, client, possibleSentID)
		if err != nil && !errors.Is(err, platform.ErrNotFound) {
			return nil, err
		}
	}

	if previousEmail == nil {
		// If the messageID does not corresponded to a sent email, check if it's a received email
		fmt.Println("checking original messageID")
		var resp *dynamodb.QueryOutput
		resp, err = client.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(env.TableName),
			IndexName:              aws.String(env.GsiOriginalIndexName),
			KeyConditionExpression: aws.String("OriginalMessageID = :originalMessageID"),
			ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
				":originalMessageID": &dynamodbTypes.AttributeValueMemberS{Value: originalMessageID},
			},
		})
		if err != nil {
			if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
				return nil, platform.ErrTooManyRequests
			}
			return nil, err
		}
		// TODO: handle the case where len(resp.Items) > 1
		if len(resp.Items) == 0 {
			return &DetermineThreadOutput{}, nil
		}

		searchMessageID := resp.Items[0]["MessageID"].(*dynamodbTypes.AttributeValueMemberS).Value
		previousEmail, err = email.Get(ctx, client, searchMessageID)
		if err != nil {
			if errors.Is(err, platform.ErrNotFound) {
				return &DetermineThreadOutput{}, nil
			}
			return nil, err
		}
	}

	if previousEmail.ThreadID == "" {
		// There's no thread for previousEmail, so we need to create a new thread
		fmt.Println("determining thread finished: new thread should be created")
		threadID := idutil.GenerateThreadID()
		output := &DetermineThreadOutput{
			ThreadID:        threadID,
			ShouldCreate:    true,
			CreatingEmailID: previousEmail.MessageID,
			CreatingSubject: previousEmail.Subject,
			CreatingTime:    previousEmail.TimeReceived,
		}
		if previousEmail.Type == model.EmailTypeSent {
			output.CreatingTime = previousEmail.TimeSent
		}
		return output, nil
	}

	// IsThreadLatest isn't moved by a sent reply, so the thread itself is read
	fmt.Println("determining thread finished: getting existing thread")
	consistentRead := true
	thread, err := getThread(ctx, client, previousEmail.ThreadID, consistentRead)
	if err != nil {
		return nil, err
	}
	fmt.Println("get existing thread finished")
	return &DetermineThreadOutput{
		ThreadID:          previousEmail.ThreadID,
		Exists:            true,
		PreviousMessageID: thread.EmailIDs[len(thread.EmailIDs)-1],
		EmailCount:        len(thread.EmailIDs),
	}, nil
}

type StoreEmailWithExistingThreadInput struct {
	ThreadID          string
	Email             map[string]dynamodbTypes.AttributeValue
	TimeReceived      string
	PreviousMessageID string
	EmailCount        int // the number of emails in the thread when PreviousMessageID was read
}

// StoreEmailWithExistingThread stores the email and updates the thread.
// It returns platform.ErrThreadConflict if the thread has gained an email since EmailCount was read.
func StoreEmailWithExistingThread(ctx context.Context, client platform.TransactWriteItemsAPI, input *StoreEmailWithExistingThreadInput) error {
	input.Email["IsThreadLatest"] = &dynamodbTypes.AttributeValueMemberBOOL{Value: true}
	emailPut := newEmailPut(input.Email)
	items := []dynamodbTypes.TransactWriteItem{
		{
			// Store new email
			Put: emailPut,
		},
		{
			// Update the thread
			Update: &dynamodbTypes.Update{
				TableName: aws.String(env.TableName),
				Key: map[string]dynamodbTypes.AttributeValue{
					"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.ThreadID},
				},
				UpdateExpression:    aws.String("SET #emails = list_append(#emails, :emails), #timeUpdated = :timeUpdated"),
				ConditionExpression: aws.String("size(#emails) = :emailCount"),
				ExpressionAttributeNames: map[string]string{
					"#emails":      "EmailIDs",
					"#timeUpdated": "TimeUpdated",
				},
				ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
					":emails":      &dynamodbTypes.AttributeValueMemberL{Value: []dynamodbTypes.AttributeValue{input.Email["MessageID"]}},
					":timeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: input.TimeReceived},
					":emailCount":  &dynamodbTypes.AttributeValueMemberN{Value: strconv.Itoa(input.EmailCount)},
				},
			},
		},
		{
			// Remove IsThreadLatest from the previous email
			Update: &dynamodbTypes.Update{
				TableName: aws.String(env.TableName),
				Key: map[string]dynamodbTypes.AttributeValue{
					"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.PreviousMessageID},
				},
				UpdateExpression: aws.String("REMOVE IsThreadLatest"),
			},
		},
	}
	_, err := client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	if isPutConditionFailed(err, items, emailPut) {
		return platform.ErrEmailExists
	}
	if isConditionFailed(err) {
		return platform.ErrThreadConflict
	}
	if err != nil {
		return err
	}

	return nil
}

type StoreEmailWithNewThreadInput struct {
	ThreadID        string
	Email           map[string]dynamodbTypes.AttributeValue
	TimeReceived    string
	CreatingEmailID string
	CreatingSubject string
	CreatingTime    string
}

// StoreEmailWithNewThread stores the email, creates a new thread, and add ThreadID to previous email.
// It returns platform.ErrThreadConflict if the previous email has been added to a thread meanwhile.
func StoreEmailWithNewThread(ctx context.Context, client platform.TransactWriteItemsAPI, input *StoreEmailWithNewThreadInput) error {
	t, err := time.Parse(time.RFC3339, input.CreatingTime)
	if err != nil {
		return err
	}
	typeYearMonth, err := format.TypeYearMonth("thread", t)
	if err != nil {
		return err
	}

	thread := map[string]dynamodbTypes.AttributeValue{
		"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: input.ThreadID},
		"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: typeYearMonth},
		"Subject":       &dynamodbTypes.AttributeValueMemberS{Value: input.CreatingSubject},
		"EmailIDs": &dynamodbTypes.AttributeValueMemberL{
			Value: []dynamodbTypes.AttributeValue{
				&dynamodbTypes.AttributeValueMemberS{Value: input.CreatingEmailID},
				input.Email["MessageID"],
			},
		},
		"TimeUpdated": &dynamodbTypes.AttributeValueMemberS{Value: input.TimeReceived},
	}

	input.Email["IsThreadLatest"] = &dynamodbTypes.AttributeValueMemberBOOL{Value: true}
	emailPut := newEmailPut(input.Email)
	items := []dynamodbTypes.TransactWriteItem{
		{
			// Set ThreadID to previous email
			Update: &dynamodbTypes.Update{
				TableName: aws.String(env.TableName),
				Key: map[string]dynamodbTypes.AttributeValue{
					"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: input.CreatingEmailID},
				},
				UpdateExpression: aws.String("SET #threadID = :threadID"),
				// the update would otherwise create the email if it's been deleted
				ConditionExpression: aws.String("attribute_exists(MessageID) AND " + email.NotInThreadCondition),
				ExpressionAttributeNames: map[string]string{
					"#threadID": "ThreadID",
				},
				ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
					":threadID": &dynamodbTypes.AttributeValueMemberS{Value: input.ThreadID},
					":zero":     &dynamodbTypes.AttributeValueMemberN{Value: "0"},
				},
			},
		},
		{
			// Store the new email
			Put: emailPut,
		},
		{
			// Create the new thread
			Put: &dynamodbTypes.Put{
				TableName: aws.String(env.TableName),
				Item:      thread,
			},
		},
	}
	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	if isPutConditionFailed(err, items, emailPut) {
		return platform.ErrEmailExists
	}
	if isConditionFailed(err) {
		return platform.ErrThreadConflict
	}
	if err != nil {
		return err
	}
	return nil
}

const emailNotExistsCondition = "attribute_not_exists(MessageID)"

func newEmailPut(email map[string]dynamodbTypes.AttributeValue) *dynamodbTypes.Put {
	return &dynamodbTypes.Put{
		TableName:           aws.String(env.TableName),
		Item:                email,
		ConditionExpression: aws.String(emailNotExistsCondition),
	}
}

func isPutConditionFailed(err error, items []dynamodbTypes.TransactWriteItem, put *dynamodbTypes.Put) bool {
	var canceledErr *dynamodbTypes.TransactionCanceledException
	if !errors.As(err, &canceledErr) {
		return false
	}
	// cancellation reasons are in the same order as the transaction's items
	for i, item := range items {
		if item.Put == put && i < len(canceledErr.CancellationReasons) {
			return aws.ToString(canceledErr.CancellationReasons[i].Code) == "ConditionalCheckFailed"
		}
	}
	return false
}

// isConditionFailed reports whether any item's condition canceled the transaction
func isConditionFailed(err error) bool {
	var canceledErr *dynamodbTypes.TransactionCanceledException
	if !errors.As(err, &canceledErr) {
		return false
	}
	for _, reason := range canceledErr.CancellationReasons {
		if aws.ToString(reason.Code) == "ConditionalCheckFailed" {
			return true
		}
	}
	return false
}

type StoreEmailInput struct {
	InReplyTo    string
	References   string
	Item         map[string]dynamodbTypes.AttributeValue
	TimeReceived string // RFC3339
}

// storeAttempts is how many times an email is stored before a thread conflict fails the invocation
const storeAttempts = 3

// conflictWait returns a random wait, so emails that conflicted don't retry in lockstep
var conflictWait = func() time.Duration {
	return time.Duration(50+rand.IntN(200)) * time.Millisecond //nolint:gosec // jitter, not security
}

// StoreEmail stores the email and adds it to its thread.
// An email that's already stored is left unchanged, so receiving the same email again is safe.
func StoreEmail(ctx context.Context, client platform.StoreEmailAPI, input *StoreEmailInput) error {
	id, ok := input.Item["MessageID"].(*dynamodbTypes.AttributeValueMemberS)
	if !ok {
		return fmt.Errorf("email has no MessageID, %w", platform.ErrInvalidInput)
	}
	messageID := id.Value

	var err error
	for attempt := 1; attempt <= storeAttempts; attempt++ {
		if attempt > 1 {
			log.Printf("thread changed while storing email %s, retrying\n", messageID)
			if !timeutil.Sleep(ctx, conflictWait()) {
				break
			}
		}
		// each attempt starts from the received item, since storing adds thread attributes to it
		attemptInput := *input
		attemptInput.Item = maps.Clone(input.Item)
		err = storeEmail(ctx, client, messageID, &attemptInput)
		if !errors.Is(err, platform.ErrThreadConflict) {
			break
		}
	}
	if errors.Is(err, platform.ErrEmailExists) {
		log.Printf("email %s already stored, skipping\n", messageID)
		return nil
	}
	return err
}

// storeEmail returns platform.ErrEmailExists if the email is already stored, also when a concurrent invocation stored it first
func storeEmail(ctx context.Context, client platform.StoreEmailAPI, messageID string, input *StoreEmailInput) error {
	exists, err := emailExists(ctx, client, messageID)
	if err != nil {
		return fmt.Errorf("failed to check if email exists, %w", err)
	}
	if exists {
		return platform.ErrEmailExists
	}

	output, err := DetermineThread(ctx, client, &DetermineThreadInput{
		InReplyTo:  input.InReplyTo,
		References: input.References,
	})
	if err != nil {
		// Some lookup failures never clear, e.g. a missing thread, and the email must not be lost over it
		log.Printf("failed to determine thread, storing without one, %v\n", err)
		output = &DetermineThreadOutput{}
	}
	if output.Exists && output.PreviousMessageID == messageID {
		// Stored by a concurrent invocation after the check above. Storing it would put and
		// update the same item in one transaction, which DynamoDB rejects outright.
		return platform.ErrEmailExists
	}
	if output.ThreadID != "" {
		input.Item["ThreadID"] = &dynamodbTypes.AttributeValueMemberS{Value: output.ThreadID}
	}

	switch {
	case output.Exists:
		err = StoreEmailWithExistingThread(ctx, client, &StoreEmailWithExistingThreadInput{
			ThreadID:          output.ThreadID,
			Email:             input.Item,
			TimeReceived:      input.TimeReceived,
			PreviousMessageID: output.PreviousMessageID,
			EmailCount:        output.EmailCount,
		})
		if err != nil {
			return fmt.Errorf("failed to store email with existing thread, %w", err)
		}
	case output.ShouldCreate:
		err = StoreEmailWithNewThread(ctx, client, &StoreEmailWithNewThreadInput{
			ThreadID:        output.ThreadID,
			Email:           input.Item,
			TimeReceived:    input.TimeReceived,
			CreatingEmailID: output.CreatingEmailID,
			CreatingSubject: output.CreatingSubject,
			CreatingTime:    output.CreatingTime,
		})
		if err != nil {
			return fmt.Errorf("failed to store email with new thread, %w", err)
		}
	default:
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName:           &env.TableName,
			Item:                input.Item,
			ConditionExpression: aws.String(emailNotExistsCondition),
		})
		if apiErr := new(dynamodbTypes.ConditionalCheckFailedException); errors.As(err, &apiErr) {
			return platform.ErrEmailExists
		}
		if err != nil {
			return fmt.Errorf("failed to store item in DynamoDB, %w", err)
		}
	}
	return nil
}

// emailExists reads strongly consistently, so an email stored moments ago by another invocation is seen
func emailExists(ctx context.Context, client platform.GetItemAPI, messageID string) (bool, error) {
	resp, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String("MessageID"),
	})
	if err != nil {
		return false, err
	}
	return len(resp.Item) > 0, nil
}
