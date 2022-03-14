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

type s3Storage struct{}

var S3 s3Storage

// readEmailEnvelope will be mocked in unit testing
var readEmailEnvelope = enmime.ReadEnvelope

type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// GetEmail retrieved an email from s3 bucket
func (s s3Storage) GetEmail(ctx context.Context, api S3GetObjectAPI, messageID string) (text, html string, err error) {
	object, err := api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s3Bucket,
		Key:    &messageID,
	})
	if err != nil {
		return "", "", err
	}
	defer object.Body.Close()

	env, err := readEmailEnvelope(object.Body)
	if err != nil {
		return
	}
	return env.Text, env.HTML, nil
}

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
