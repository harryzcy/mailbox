package storage

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jhillyerd/enmime"
)

var (
	s3Bucket = os.Getenv("S3_BUCKET")
)

type s3Storage struct{}

var S3 s3Storage

// GetEmail retrieved an email from s3 bucket
func (s s3Storage) GetEmail(ctx context.Context, cfg aws.Config, messageID string) (text, html string, err error) {
	svc := s3.NewFromConfig(cfg)

	resp, err := svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s3Bucket,
		Key:    &messageID,
	})
	if err != nil {
		return
	}

	env, err := enmime.ReadEnvelope(resp.Body)
	if err != nil {
		return
	}
	return env.Text, env.HTML, nil
}
