package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/harryzcy/mailbox/util"
)

func receiveEmail(ses events.SimpleEmailService) {
	log.Printf("received an email from %s", ses.Mail.Source)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	item := make(map[string]types.AttributeValue)
	item["dateSent"] = &types.AttributeValueMemberS{Value: util.FormatDate(ses.Mail.CommonHeaders.Date)}
	item["timeReceived"] = &types.AttributeValueMemberS{Value: util.FormatTimestamp(ses.Mail.Timestamp)}
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

	err = storeInDynamoDB(cfg, item)
	if err != nil {
		log.Fatalf("failed to store item, %v", err)
	}
	return
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
