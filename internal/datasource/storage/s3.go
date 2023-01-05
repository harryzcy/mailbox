package storage

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jhillyerd/enmime"
)

var (
	s3Bucket = os.Getenv("S3_BUCKET")
)

type GetEmailResponse struct {
	Text string
	HTML string
}

// S3Storage is an interface that defines required S3 functions
type S3Storage interface {
	GetEmail(ctx context.Context, api S3GetObjectAPI, messageID string) (*GetEmailResponse, error)
	DeleteEmail(ctx context.Context, api S3DeleteObjectAPI, messageID string) error
}

type s3Storage struct{}

// S3 holds functions that handles S3 related operations
var S3 S3Storage = s3Storage{}

// readEmailEnvelope is used in GetEmail will be mocked in unit testing
var readEmailEnvelope = enmime.ReadEnvelope

// S3GetObjectAPI defines set of API required by GetEmail functions
type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// GetEmail retrieved an email from s3 bucket
func (s s3Storage) GetEmail(ctx context.Context, api S3GetObjectAPI, messageID string) (*GetEmailResponse, error) {
	object, err := api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s3Bucket,
		Key:    &messageID,
	})
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()

	env, err := readEmailEnvelope(object.Body)
	if err != nil {
		return nil, err
	}
	return &GetEmailResponse{
		Text: env.Text,
		HTML: env.HTML,
	}, nil
}

// S3DeleteObjectAPI defines set of API required by DeleteEmail functions
type S3DeleteObjectAPI interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// DeleteEmail deletes an email from S3 bucket
func (s s3Storage) DeleteEmail(ctx context.Context, api S3DeleteObjectAPI, messageID string) error {
	_, err := api.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s3Bucket,
		Key:    &messageID,
	})
	if err != nil {
		return err
	}

	return nil
}
