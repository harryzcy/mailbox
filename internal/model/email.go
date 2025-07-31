package model

import (
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// The constants representing email types
const (
	// EmailTypeInbox represents an inbox email
	EmailTypeInbox = "inbox"
	// EmailTypeInbox represents a sent email
	EmailTypeSent = "sent"
	// EmailTypeInbox represents a draft email
	EmailTypeDraft = "draft"

	// TODO: refactor
	// EmailTypeThread represents a thread, which is a group of emails
	EmailTypeThread = "thread"
)

type File struct {
	ContentID         string            `json:"contentID"`
	ContentType       string            `json:"contentType"`
	ContentTypeParams map[string]string `json:"contentTypeParams"`
	Filename          string            `json:"filename"`
}

func (f File) ToAttributeValue() dynamodbTypes.AttributeValue {
	params := make(map[string]dynamodbTypes.AttributeValue)
	for k, v := range f.ContentTypeParams {
		params[k] = &dynamodbTypes.AttributeValueMemberS{
			Value: v,
		}
	}

	return &dynamodbTypes.AttributeValueMemberM{
		Value: map[string]dynamodbTypes.AttributeValue{
			"contentID": &dynamodbTypes.AttributeValueMemberS{
				Value: f.ContentID,
			},
			"contentType": &dynamodbTypes.AttributeValueMemberS{
				Value: f.ContentType,
			},
			"contentTypeParams": &dynamodbTypes.AttributeValueMemberM{
				Value: params,
			},
			"filename": &dynamodbTypes.AttributeValueMemberS{
				Value: f.Filename,
			},
		},
	}
}

type Files []File

func (fs Files) ToAttributeValue() dynamodbTypes.AttributeValue {
	value := make([]dynamodbTypes.AttributeValue, len(fs))
	for i, f := range fs {
		value[i] = f.ToAttributeValue()
	}

	return &dynamodbTypes.AttributeValueMemberL{
		Value: value,
	}
}
