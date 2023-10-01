package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/jhillyerd/enmime"
)

type SendResult struct {
	MessageID string
}

// Send sends a draft email
func Send(ctx context.Context, client api.GetAndSendEmailAPI, messageID string) (*SendResult, error) {
	if !strings.HasPrefix(messageID, "draft-") {
		return nil, api.ErrEmailIsNotDraft
	}

	resp, err := Get(ctx, client, messageID)
	if err != nil {
		return nil, err
	}

	email := &EmailInput{
		MessageID:  messageID,
		Subject:    resp.Subject,
		From:       resp.From,
		To:         resp.To,
		Cc:         resp.Cc,
		Bcc:        resp.Bcc,
		ReplyTo:    resp.ReplyTo,
		InReplyTo:  resp.InReplyTo,
		References: resp.References,
		Text:       resp.Text,
		HTML:       resp.HTML,
		ThreadID:   resp.ThreadID,
	}
	newMessageID, err := sendEmailViaSES(ctx, client, email)
	if err != nil {
		return nil, err
	}
	email.MessageID = newMessageID

	err = markEmailAsSent(ctx, client, messageID, email)
	if err != nil {
		return nil, err
	}

	fmt.Println("send method finished successfully")
	return &SendResult{
		MessageID: newMessageID,
	}, nil
}

// sendEmailViaSES sends an email via SES.
// If it is a reply, it will build the MIME message and send it as a raw email.
// In this case, it is assumed that both InReplyTo and References are not empty.
// Otherwise, it will use the simple email API.
func sendEmailViaSES(ctx context.Context, client api.SendEmailAPI, email *EmailInput) (string, error) {
	fmt.Println("sending email via SES")
	input := &sesv2.SendEmailInput{
		Content: &sestypes.EmailContent{},
		Destination: &sestypes.Destination{
			ToAddresses:  email.To,
			CcAddresses:  email.Cc,
			BccAddresses: email.Bcc,
		},
		FromEmailAddress: aws.String(email.From[0]),
		ReplyToAddresses: email.ReplyTo,
	}

	if email.InReplyTo == "" {
		// Use simple email when it's not a reply,
		// since we don't need to customize the headers in this case
		fmt.Println("sending simple email")
		input.Content.Simple = &sestypes.Message{
			Body: &sestypes.Body{
				Html: &sestypes.Content{
					Data:    aws.String(email.HTML),
					Charset: aws.String("UTF-8"),
				},
				Text: &sestypes.Content{
					Data:    aws.String(email.Text),
					Charset: aws.String("UTF-8"),
				},
			},
			Subject: &sestypes.Content{
				Data:    aws.String(email.Subject),
				Charset: aws.String("UTF-8"),
			},
		}
	} else {
		// Use raw email when it's a reply.
		// We need to customize the In-Reply-To and References headers
		fmt.Println("sending raw email")
		data, err := buildMIMEEmail(email)
		if err != nil {
			return "", err
		}
		input.Content.Raw = &sestypes.RawMessage{
			Data: data,
		}
	}

	resp, err := client.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	fmt.Println("email sent successfully")
	return *resp.MessageId, nil
}

// markEmailAsSent marks an email as sent in DynamoDB.
// It will delete the old draft email and create a new sent email.
// If the email is a reply, it will also update the thread by removing the DraftID attribute and append the new MessageID to the EmailIDs attribute.
//
// input:
//   - oldMessageID: the MessageID of the draft email
//   - email: the new sent email (with the new MessageID)
func markEmailAsSent(ctx context.Context, client api.SendEmailAPI, oldMessageID string, email *EmailInput) error {
	fmt.Println("marking email as sent")
	now := getUpdatedTime()
	typeYearMonth, _ := format.FormatTypeYearMonth(EmailTypeSent, now)
	dateTime := format.FormatDateTime(now)

	item := email.GenerateAttributes(typeYearMonth, dateTime)

	// Delete the old draft email and create the new sent email
	input := &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbTypes.TransactWriteItem{
			{
				Delete: &dynamodbTypes.Delete{
					TableName: aws.String(env.TableName),
					Key: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: oldMessageID},
					},
				},
			},
			{
				Put: &dynamodbTypes.Put{
					TableName: aws.String(env.TableName),
					Item:      item,
				},
			},
		},
	}
	// If it's a reply, update the thread:
	// 1. removing DraftID
	// 2.  append the new MessageID to the EmailIDs attribute
	if email.InReplyTo != "" {
		fmt.Println("include thread update")
		input.TransactItems = append(input.TransactItems, dynamodbTypes.TransactWriteItem{
			Update: &dynamodbTypes.Update{
				TableName: aws.String(env.TableName),
				Key: map[string]dynamodbTypes.AttributeValue{
					"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: email.ThreadID},
				},
				UpdateExpression: aws.String("REMOVE DraftID SET EmailIDs = list_append(EmailIDs, :newMessageID)"),
				ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
					":newMessageID": &dynamodbTypes.AttributeValueMemberL{
						Value: []dynamodbTypes.AttributeValue{
							&dynamodbTypes.AttributeValueMemberS{Value: email.MessageID},
						},
					},
				},
			},
		})
	}
	_, err := client.TransactWriteItems(ctx, input)

	if err != nil {
		if apiErr := new(dynamodbTypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return api.ErrTooManyRequests
		}
		if apiErr := new(dynamodbTypes.TransactionCanceledException); errors.As(err, &apiErr) {
			fmt.Printf("transaction canceled, %s\n", apiErr.Error())
			logCancellationReasons(apiErr.CancellationReasons)
		}
		return err
	}
	fmt.Println("email marked as sent successfully")
	return nil
}

func buildMIMEEmail(email *EmailInput) ([]byte, error) {
	var errs []error
	builder := enmime.Builder()
	builder = builder.Subject(email.Subject)

	if len(email.From) == 0 {
		errs = append(errs, api.ErrInvalidInput)
	} else {
		if from, err := mail.ParseAddress(email.From[0]); err == nil {
			builder = builder.From(from.Name, from.Address)
		} else {
			errs = append(errs, fmt.Errorf("failed to parse from address: %v", err))
		}
	}

	if to, err := convertToMailAddresses(email.To); err == nil {
		builder = builder.ToAddrs(to)
	} else {
		errs = append(errs, fmt.Errorf("failed to parse to address: %v", err))
	}

	if cc, err := convertToMailAddresses(email.Cc); err == nil {
		builder = builder.CCAddrs(cc)
	} else {
		errs = append(errs, fmt.Errorf("failed to parse cc address: %v", err))
	}

	if bcc, err := convertToMailAddresses(email.Bcc); err == nil {
		builder = builder.BCCAddrs(bcc)
	} else {
		errs = append(errs, fmt.Errorf("failed to parse bcc address: %v", err))
	}

	if len(email.ReplyTo) == 0 {
		errs = append(errs, api.ErrInvalidInput)
	} else {
		if replyTo, err := mail.ParseAddress(email.ReplyTo[0]); err == nil {
			builder = builder.ReplyTo(replyTo.Name, replyTo.Address)
		} else {
			errs = append(errs, fmt.Errorf("failed to parse reply-to address: %v", err))
		}
	}

	if email.InReplyTo != "" {
		builder = builder.Header("In-Reply-To", email.InReplyTo)
	}
	if email.References != "" {
		builder = builder.Header("References", email.References)
	}
	builder = builder.Text([]byte(email.Text))
	builder = builder.HTML([]byte(email.HTML))

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	part, err := builder.Build()
	if err != nil {
		return nil, err
	}
	writer := bytes.NewBuffer(nil)
	err = part.Encode(writer)
	if err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

func convertToMailAddresses(addresses []string) ([]mail.Address, error) {
	var mailAddresses []mail.Address
	for _, stringAddress := range addresses {
		address, err := mail.ParseAddress(stringAddress)
		if err != nil {
			return nil, err
		}
		mailAddresses = append(mailAddresses, *address)
	}
	return mailAddresses, nil
}

func logCancellationReasons(reasons []dynamodbTypes.CancellationReason) {
	for _, reason := range reasons {
		if reason.Code == nil {
			continue
		}

		log := fmt.Sprintf("code: %s, message: ", *reason.Code)
		if reason.Message != nil {
			log += *reason.Message
		} else {
			log += "nil"
		}
		log += ", item: "
		if reason.Item != nil {
			log += fmt.Sprintf("%v", reason.Item)
		} else {
			log += "nil"
		}
		fmt.Println(log)
	}
}
