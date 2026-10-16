package main

import (
	"errors"
	"fmt"
	"testing"

	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
)

func TestIsRawEmailDeleted(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "no error", err: nil, expected: false},
		{name: "missing object", err: fmt.Errorf("operation error S3: GetObject, %w", &s3Types.NoSuchKey{}), expected: true},
		{name: "access denied", err: errors.New("operation error S3: GetObject, AccessDenied"), expected: false},
		{name: "missing bucket", err: &s3Types.NoSuchBucket{}, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, isRawEmailDeleted(test.err))
		})
	}
}
