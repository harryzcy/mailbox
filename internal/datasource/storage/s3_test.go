package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/harryzcy/mailbox/internal/env"
	"github.com/jhillyerd/enmime"
	"github.com/stretchr/testify/assert"
)

type mockGetObjectAPI func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)

func (m mockGetObjectAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestS3_GetEmail(t *testing.T) {
	env.S3Bucket = "test_bucket"

	cases := []struct {
		client            func(t *testing.T) S3GetObjectAPI
		readEmailEnvelope func(r io.Reader) (*enmime.Envelope, error)
		messageID         string
		expectedText      string
		expectedHTML      string
		expectedErr       error
	}{
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(_ context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					assert.NotNil(t, params.Bucket, "expect bucket to not be nil")
					assert.Equal(t, env.S3Bucket, *params.Bucket)
					assert.NotNil(t, params.Key, "expect key to not be nil")
					assert.Equal(t, "exampleMessageID", *params.Key)

					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader([]byte(""))),
					}, nil
				})
			},
			readEmailEnvelope: func(_ io.Reader) (*enmime.Envelope, error) {
				return &enmime.Envelope{Text: "example-text", HTML: "<p>example-html</p>"}, nil
			},
			messageID:    "exampleMessageID",
			expectedText: "example-text",
			expectedHTML: "<p>example-html</p>",
		},
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(_ context.Context, _ *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					return &s3.GetObjectOutput{}, errors.New("some-error")
				})
			},
			expectedErr: errors.New("some-error"),
		},
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(_ context.Context, _ *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader([]byte(""))),
					}, nil
				})
			},
			readEmailEnvelope: func(_ io.Reader) (*enmime.Envelope, error) {
				return &enmime.Envelope{Text: "example-text", HTML: "<p>example-html</p>"}, errors.New("some-error-in-email")
			},
			expectedErr: errors.New("some-error-in-email"),
		},
	}

	for i, test := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			readEmailEnvelope = test.readEmailEnvelope

			response, err := S3.GetEmail(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
			if response != nil {
				assert.Equal(t, test.expectedText, response.Text)
				assert.Equal(t, test.expectedHTML, response.HTML)
			}
		})
	}
}

func TestS3_GetEmailRaw(t *testing.T) {
	env.S3Bucket = "test_bucket"

	cases := []struct {
		client      func(t *testing.T) S3GetObjectAPI
		messageID   string
		expectedRaw []byte
		expectedErr error
	}{
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					assert.NotNil(t, params.Bucket, "expect bucket to not be nil")
					assert.Equal(t, env.S3Bucket, *params.Bucket)
					assert.NotNil(t, params.Key, "expect key to not be nil")
					assert.Equal(t, "exampleMessageID", *params.Key)

					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader([]byte("MIME content"))),
					}, nil
				})
			},
			messageID:   "exampleMessageID",
			expectedRaw: []byte("MIME content"),
		},
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					return &s3.GetObjectOutput{}, errors.New("some-error")
				})
			},
			expectedErr: errors.New("some-error"),
		},
	}

	for i, test := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			response, err := S3.GetEmailRaw(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
			if response != nil {
				assert.Equal(t, test.expectedRaw, response)
			}
		})
	}
}

type mockDeleteObjectAPI func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)

func (m mockDeleteObjectAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestS3_DeleteEmail(t *testing.T) {
	env.S3Bucket = "test_bucket"
	tests := []struct {
		client      func(t *testing.T) S3DeleteObjectAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) S3DeleteObjectAPI {
				return mockDeleteObjectAPI(func(_ context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					t.Helper()
					assert.NotNil(t, params.Bucket, "expect bucket to not be nil")
					assert.Equal(t, env.S3Bucket, *params.Bucket)
					assert.NotNil(t, params.Key, "expect key to not be nil")
					assert.Equal(t, "exampleMessageID", *params.Key)

					return &s3.DeleteObjectOutput{}, nil
				})
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) S3DeleteObjectAPI {
				return mockDeleteObjectAPI(func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					t.Helper()

					return &s3.DeleteObjectOutput{}, errors.New("some-error")
				})
			},
			expectedErr: errors.New("some-error"),
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()

			err := S3.DeleteEmail(ctx, test.client(t), test.messageID)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
