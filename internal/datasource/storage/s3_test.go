package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jhillyerd/enmime"
	"github.com/stretchr/testify/assert"
)

type mockGetObjectAPI func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)

func (m mockGetObjectAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestS3_GetEmail(t *testing.T) {
	s3Bucket = "test_bucket"

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
				return mockGetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					assert.NotNil(t, params.Bucket, "expect bucket to not be nil")
					assert.Equal(t, s3Bucket, *params.Bucket)
					assert.NotNil(t, params.Key, "expect key to not be nil")
					assert.Equal(t, "exampleMessageID", *params.Key)

					return &s3.GetObjectOutput{
						Body: ioutil.NopCloser(bytes.NewReader([]byte(""))),
					}, nil
				})
			},
			readEmailEnvelope: func(r io.Reader) (*enmime.Envelope, error) {
				return &enmime.Envelope{Text: "example-text", HTML: "<p>example-html</p>"}, nil
			},
			messageID:    "exampleMessageID",
			expectedText: "example-text",
			expectedHTML: "<p>example-html</p>",
		},
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					return &s3.GetObjectOutput{}, errors.New("some-error")
				})
			},
			expectedErr: errors.New("some-error"),
		},
		{
			client: func(t *testing.T) S3GetObjectAPI {
				return mockGetObjectAPI(func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					return &s3.GetObjectOutput{
						Body: ioutil.NopCloser(bytes.NewReader([]byte(""))),
					}, nil
				})
			},
			readEmailEnvelope: func(r io.Reader) (*enmime.Envelope, error) {
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

type mockDeleteObjectAPI func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)

func (m mockDeleteObjectAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m(ctx, params, optFns...)
}

func TestS3_DeleteEmail(t *testing.T) {
	s3Bucket = "test_bucket"
	tests := []struct {
		client      func(t *testing.T) S3DeleteObjectAPI
		messageID   string
		expectedErr error
	}{
		{
			client: func(t *testing.T) S3DeleteObjectAPI {
				return mockDeleteObjectAPI(func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					t.Helper()
					assert.NotNil(t, params.Bucket, "expect bucket to not be nil")
					assert.Equal(t, s3Bucket, *params.Bucket)
					assert.NotNil(t, params.Key, "expect key to not be nil")
					assert.Equal(t, "exampleMessageID", *params.Key)

					return &s3.DeleteObjectOutput{}, nil
				})
			},
			messageID: "exampleMessageID",
		},
		{
			client: func(t *testing.T) S3DeleteObjectAPI {
				return mockDeleteObjectAPI(func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
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
