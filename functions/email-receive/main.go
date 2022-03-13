package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/harryzcy/mailbox/internal/datasource/storage"
	"github.com/harryzcy/mailbox/internal/util/format"
)

// AWS Region
var region = os.Getenv("REGION")

func receiveEmail(ctx context.Context, ses events.SimpleEmailService) {
	log.Printf("received an email from %s", ses.Mail.Source)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	item := make(map[string]types.AttributeValue)
	item["DateSent"] = &types.AttributeValueMemberS{Value: format.FormatDate(ses.Mail.CommonHeaders.Date)}

	// YYYY-MM
	typeYearMonth, _ := format.FormatTypeYearMonth("inbox", ses.Mail.Timestamp)
	item["TypeYearMonth"] = &types.AttributeValueMemberS{Value: typeYearMonth}

	item["DateTime"] = &types.AttributeValueMemberS{Value: format.FormatDateTime(ses.Mail.Timestamp)}
	item["MessageID"] = &types.AttributeValueMemberS{Value: ses.Mail.MessageID}
	item["Subject"] = &types.AttributeValueMemberS{Value: ses.Mail.CommonHeaders.Subject}
	item["Source"] = &types.AttributeValueMemberS{Value: ses.Mail.Source}
	item["Destination"] = &types.AttributeValueMemberSS{Value: ses.Mail.Destination}
	item["From"] = &types.AttributeValueMemberSS{Value: ses.Mail.CommonHeaders.From}
	item["To"] = &types.AttributeValueMemberSS{Value: ses.Mail.CommonHeaders.To}
	item["ReturnPath"] = &types.AttributeValueMemberS{Value: ses.Mail.CommonHeaders.ReturnPath}

	text, html, err := storage.S3.GetEmail(ctx, s3.NewFromConfig(cfg), ses.Mail.MessageID)
	if err != nil {
		log.Fatalf("failed to get object, %v", err)
	}
	item["Text"] = &types.AttributeValueMemberS{Value: text}
	item["HTML"] = &types.AttributeValueMemberS{Value: html}

	log.Printf("subject: %v", ses.Mail.CommonHeaders.Subject)

	err = storage.DynamoDB.Store(ctx, dynamodb.NewFromConfig(cfg), item)
	if err != nil {
		log.Fatalf("failed to store item, %v", err)
	}
}

func handler(ctx context.Context, sesEvent events.SimpleEmailEvent) error {
	for _, record := range sesEvent.Records {
		ses := record.SES
		fmt.Printf("[%s - %s] Mail = %+v, Receipt = %+v \n", record.EventVersion, record.EventSource, ses.Mail, ses.Receipt)
		receiveEmail(ctx, record.SES)
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
