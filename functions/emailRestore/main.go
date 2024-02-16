/*
 * WARNING: This code is used once as a temporary solution and is not production ready.
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jhillyerd/enmime"

	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/format"
)

func main() {
	lambda.Start(handler)
}

type client struct {
	s3Client       *s3.Client
	dynamoDBClient *dynamodb.Client
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(env.Region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	cli := &client{
		s3Client:       s3.NewFromConfig(cfg),
		dynamoDBClient: dynamodb.NewFromConfig(cfg),
	}

	failures := make([]events.SQSBatchItemFailure, 0)
	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)
		err := restoreEmail(ctx, cli, message.Body)
		if err != nil {
			failures = append(failures, events.SQSBatchItemFailure{
				ItemIdentifier: message.MessageId,
			})
		}
	}

	return events.SQSEventResponse{
		BatchItemFailures: failures,
	}, nil
}

func restoreEmail(ctx context.Context, cli *client, messageID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	item := make(map[string]dynamodbTypes.AttributeValue)

	getResp, err := cli.dynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(env.TableName),
		Key: map[string]dynamodbTypes.AttributeValue{
			"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: messageID},
		},
	})
	fmt.Println("got item from dynamodb, err:", err)
	if err == nil {
		if len(getResp.Item) > 0 {
			fmt.Println("item already exist in dynamodb: " + messageID)
			return nil
		}
	}

	object, err := cli.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &env.S3Bucket,
		Key:    &messageID,
	})
	fmt.Println("got object from s3, err:", err)
	if err != nil {
		if apiErr := new(s3Types.NotFound); errors.As(err, &apiErr) {
			fmt.Println("s3 object not found: " + messageID)
			return nil
		}
		return err
	}

	envelope, err := enmime.ReadEnvelope(object.Body)
	fmt.Println("envelop parsed, err:", err)
	if err != nil {
		return err
	}

	item["DateSent"] = &dynamodbTypes.AttributeValueMemberS{Value: format.Date(envelope.GetHeader("Date"))}

	// YYYY-MM
	typeYearMonth, err := format.FormatTypeYearMonth("inbox", *object.LastModified)
	if err != nil {
		return err
	}
	item["TypeYearMonth"] = &dynamodbTypes.AttributeValueMemberS{Value: typeYearMonth}

	item["DateTime"] = &dynamodbTypes.AttributeValueMemberS{Value: format.DateTime(*object.LastModified)}
	item["MessageID"] = &dynamodbTypes.AttributeValueMemberS{Value: messageID}
	item["Subject"] = &dynamodbTypes.AttributeValueMemberS{Value: envelope.GetHeader("Subject")}
	item["Source"] = &dynamodbTypes.AttributeValueMemberS{Value: cleanAddress(envelope.GetHeader("Return-Path"), true)}

	item["Destination"] = &dynamodbTypes.AttributeValueMemberSS{Value: cleanAddresses(envelope.GetHeaderValues("To"), true)}
	item["From"] = &dynamodbTypes.AttributeValueMemberSS{Value: cleanAddresses(envelope.GetHeaderValues("From"), false)}
	item["To"] = &dynamodbTypes.AttributeValueMemberSS{Value: cleanAddresses(envelope.GetHeaderValues("To"), false)}
	item["ReturnPath"] = &dynamodbTypes.AttributeValueMemberS{Value: cleanAddress(envelope.GetHeader("Return-Path"), true)}

	replyTo := envelope.GetHeaderValues("Reply-To")
	if len(replyTo) > 0 {
		replyTo = cleanAddresses(replyTo, true)
		item["ReplyTo"] = &dynamodbTypes.AttributeValueMemberSS{Value: replyTo}
	}

	item["Text"] = &dynamodbTypes.AttributeValueMemberS{Value: envelope.Text}
	item["HTML"] = &dynamodbTypes.AttributeValueMemberS{Value: envelope.HTML}

	item["Attachments"] = storage.ParseFiles(envelope.Attachments).ToAttributeValue()
	item["Inlines"] = storage.ParseFiles(envelope.Inlines).ToAttributeValue()
	item["OtherParts"] = storage.ParseFiles(envelope.OtherParts).ToAttributeValue()

	resp, err := cli.dynamoDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &env.TableName,
		ConditionExpression: aws.String("attribute_not_exists(MessageID)"),
		Item:                item,
	})
	fmt.Println("item put to dynamodb, err:", err)
	if err != nil {
		var ccf *dynamodbTypes.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			fmt.Printf("item already exists: %s, skipping" + messageID)
			err = nil
		} else if strings.Contains(err.Error(), "ValidationException") {
			fmt.Println("ValidationException, skipping")
			err = nil
		}
		return err
	}

	fmt.Printf("DynamoDB returned metadata: %s", resp.ResultMetadata)
	return nil
}

var (
	emailWithQuotedName = regexp.MustCompile(`^"(.*?)" ?<(.*?)>$`)
	emailWithName       = regexp.MustCompile(`^(.*?) ?<(.*?)>$`)
)

func cleanAddresses(addresses []string, omitName bool) []string {
	if len(addresses) == 0 {
		return nil
	}
	for i := range addresses {
		addresses[i] = cleanAddress(addresses[i], omitName)
	}
	return addresses
}

func cleanAddress(address string, omitName bool) string {
	address = strings.Trim(address, " ")

	if strings.HasPrefix(address, "<") && strings.HasSuffix(address, ">") {
		return address[1 : len(address)-1]
	}

	if emailWithQuotedName.MatchString(address) {
		match := emailWithQuotedName.FindStringSubmatch(address)
		name := match[1]
		if omitName {
			return match[2]
		}
		return name + " <" + match[2] + ">"
	}

	if emailWithName.MatchString(address) {
		match := emailWithName.FindStringSubmatch(address)
		name := match[1]
		if omitName {
			return match[2]
		}
		return name + " <" + match[2] + ">"
	}

	return address
}
