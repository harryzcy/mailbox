package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/harryzcy/mailbox/internal/db"
	"github.com/harryzcy/mailbox/internal/util/format"
)

// AWS Region
var region = os.Getenv("REGION")

func receiveEmail(ses events.SimpleEmailService) {
	log.Printf("received an email from %s", ses.Mail.Source)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	item := make(map[string]types.AttributeValue)
	item["dateSent"] = &types.AttributeValueMemberS{Value: format.FormatDate(ses.Mail.CommonHeaders.Date)}

	// YYYY-MM
	typeYearMonth, _ := format.FormatTypeYearMonth("inbox", ses.Mail.Timestamp)
	item["type-year-month"] = &types.AttributeValueMemberS{Value: typeYearMonth}

	item["date-time"] = &types.AttributeValueMemberS{Value: format.FormatDateTime(ses.Mail.Timestamp)}
	item["messageID"] = &types.AttributeValueMemberS{Value: ses.Mail.MessageID}
	item["subject"] = &types.AttributeValueMemberS{Value: ses.Mail.CommonHeaders.Subject}
	item["source"] = &types.AttributeValueMemberS{Value: ses.Mail.Source}
	item["destination"] = &types.AttributeValueMemberSS{Value: ses.Mail.Destination}
	item["from"] = &types.AttributeValueMemberSS{Value: ses.Mail.CommonHeaders.From}
	item["to"] = &types.AttributeValueMemberSS{Value: ses.Mail.CommonHeaders.To}
	item["returnPath"] = &types.AttributeValueMemberS{Value: ses.Mail.CommonHeaders.ReturnPath}

	text, html, err := getEmailFromS3(cfg, ses.Mail.MessageID)
	if err != nil {
		log.Fatalf("failed to get object, %v", err)
	}
	item["text"] = &types.AttributeValueMemberS{Value: text}
	item["html"] = &types.AttributeValueMemberS{Value: html}

	log.Printf("subject: %v", ses.Mail.CommonHeaders.Subject)

	err = db.StoreInDynamoDB(cfg, item)
	if err != nil {
		log.Fatalf("failed to store item, %v", err)
	}
}

func handler(ctx context.Context, sesEvent events.SimpleEmailEvent) error {
	for _, record := range sesEvent.Records {
		ses := record.SES
		fmt.Printf("[%s - %s] Mail = %+v, Receipt = %+v \n", record.EventVersion, record.EventSource, ses.Mail, ses.Receipt)
		receiveEmail(record.SES)
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
