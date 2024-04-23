package types

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func (f File) ToAttributeValue() types.AttributeValue {
	params := make(map[string]types.AttributeValue)
	for k, v := range f.ContentTypeParams {
		params[k] = &types.AttributeValueMemberS{
			Value: v,
		}
	}

	return &types.AttributeValueMemberM{
		Value: map[string]types.AttributeValue{
			"contentID": &types.AttributeValueMemberS{
				Value: f.ContentID,
			},
			"contentType": &types.AttributeValueMemberS{
				Value: f.ContentType,
			},
			"contentTypeParams": &types.AttributeValueMemberM{
				Value: params,
			},
			"filename": &types.AttributeValueMemberS{
				Value: f.Filename,
			},
		},
	}
}

type Files []File

func (fs Files) ToAttributeValue() types.AttributeValue {
	value := make([]types.AttributeValue, len(fs))
	for i, f := range fs {
		value[i] = f.ToAttributeValue()
	}

	return &types.AttributeValueMemberL{
		Value: value,
	}
}
