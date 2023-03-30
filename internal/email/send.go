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
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/jhillyerd/enmime"
)

type SendResult struct {
	MessageID string
}

// Send sends a draft email
func Send(ctx context.Context, api GetAndSendEmailAPI, messageID string) (*SendResult, error) {
	if !strings.HasPrefix(messageID, "draft-") {
		return nil, ErrEmailIsNotDraft
	}

	resp, err := Get(ctx, api, messageID)
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
	newMessageID, err := sendEmailViaSES(ctx, api, email)
	if err != nil {
		return nil, err
	}
	email.MessageID = newMessageID

	err = markEmailAsSent(ctx, api, messageID, email)
	if err != nil {
		return nil, err
	}

	fmt.Println("send method finished successfully")
	return &SendResult{
		MessageID: newMessageID,
	}, nil
}

func sendEmailViaSES(ctx context.Context, api SendEmailAPI, email *EmailInput) (string, error) {
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
		// Use simple email when it's not a reply
		// We don't need to customize the headers in this case
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
		// Use raw email when it's a reply
		// We need to customize the In-Reply-To and References headers
		data, err := buildMIMEEmail(email)
		if err != nil {
			return "", err
		}
		input.Content.Raw = &sestypes.RawMessage{
			Data: data,
		}
	}

	resp, err := api.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return *resp.MessageId, nil
}

func markEmailAsSent(ctx context.Context, api SendEmailAPI, oldMessageID string, email *EmailInput) error {
	now := getUpdatedTime()
	typeYearMonth, _ := format.FormatTypeYearMonth(EmailTypeSent, now)
	dateTime := format.FormatDateTime(now)

	item := email.GenerateAttributes(typeYearMonth, dateTime)

	_, err := api.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodbtypes.TransactWriteItem{
			{
				Delete: &dynamodbtypes.Delete{
					TableName: aws.String(tableName),
					Key: map[string]dynamodbtypes.AttributeValue{
						"MessageID": &dynamodbtypes.AttributeValueMemberS{Value: oldMessageID},
					},
				},
			},
			{
				Put: &dynamodbtypes.Put{
					TableName: aws.String(tableName),
					Item:      item,
				},
			},
		},
	})

	if err != nil {
		if apiErr := new(dynamodbtypes.ProvisionedThroughputExceededException); errors.As(err, &apiErr) {
			return ErrTooManyRequests
		}
		return err
	}
	return nil
}

func buildMIMEEmail(email *EmailInput) ([]byte, error) {
	var errs []error
	builder := enmime.Builder()
	builder = builder.Subject(email.Subject)

	if len(email.From) == 0 {
		errs = append(errs, ErrInvalidInput)
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
		errs = append(errs, ErrInvalidInput)
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
