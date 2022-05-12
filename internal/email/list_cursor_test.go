package email

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/avutil"
	"github.com/stretchr/testify/assert"
)

func TestCursor(t *testing.T) {
	tests := []struct {
		cursor Cursor
	}{
		{
			Cursor{
				QueryInfo: QueryInfo{
					Type:  "inbox",
					Year:  "2022",
					Month: "04",
					Order: "asc",
				},
			},
		},
		{
			Cursor{
				LastEvaluatedKey: map[string]types.AttributeValue{
					"foo": &types.AttributeValueMemberS{Value: "bar"},
				},
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			encoded, err := json.Marshal(test.cursor)
			assert.Nil(t, err)

			// encoded is base64 encoded
			assert.NotContains(t, string(encoded), ",")
			assert.NotContains(t, string(encoded), ":")
			assert.NotContains(t, string(encoded), "{")

			var decoded Cursor
			err = json.Unmarshal(encoded, &decoded)
			assert.Nil(t, err)
			assert.Equal(t, test.cursor, decoded)

			trimmedStr := string(encoded)
			trimmedStr = trimmedStr[1 : len(trimmedStr)-1] // remove quotes
			err = decoded.BindString(trimmedStr)
			assert.Nil(t, err)
			assert.Equal(t, test.cursor, decoded)

			trimmed := []byte(trimmedStr)
			err = decoded.Bind(trimmed)
			assert.Nil(t, err)
			assert.Equal(t, test.cursor, decoded)
		})
	}
}

func TestCursor_Empty(t *testing.T) {
	var cursor Cursor
	err := cursor.BindString("")
	assert.Nil(t, err)
	assert.Empty(t, cursor.QueryInfo)
	assert.Empty(t, cursor.LastEvaluatedKey)
}

func TestLastEvaluatedKey_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input            []byte
		lastEvaluatedKey LastEvaluatedKey
		expectedErr      error
	}{
		{[]byte{'"'}, nil, &json.SyntaxError{}},
		{[]byte{'"', '"'}, nil, nil},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var key LastEvaluatedKey
			err := json.Unmarshal(test.input, &key)
			assert.Equal(t, test.lastEvaluatedKey, key)
			assert.True(t, test.expectedErr == nil && err == nil || test.expectedErr != nil && err != nil)
			if test.expectedErr != nil {
				assert.IsType(t, test.expectedErr, err)
			}
		})
	}
}

func TestLastEvaluatedKey_UnmarshalJSON_Error(t *testing.T) {
	var key LastEvaluatedKey
	err := key.UnmarshalJSON([]byte{})
	assert.Equal(t, ErrInvalidInputToUnmarshal, err)
}

func TestLastEvaluatedKey_BindString(t *testing.T) {
	tests := []struct {
		input            string
		lastEvaluatedKey LastEvaluatedKey
		expectedErr      error
	}{
		{"", nil, nil},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var key LastEvaluatedKey
			err := key.BindString(test.input)
			assert.Equal(t, test.lastEvaluatedKey, key)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestCursor_Bind(t *testing.T) {
	tests := []struct {
		input            []byte
		lastEvaluatedKey LastEvaluatedKey
		expectedErr      error
	}{
		{
			input:            []byte("aW52YWxpZA=="), // base64 for invalid
			lastEvaluatedKey: nil,
			expectedErr:      avutil.ErrDecodeError,
		},
		{
			input:            []byte("XYZ"),
			lastEvaluatedKey: nil,
			expectedErr:      avutil.ErrDecodeError,
		},
		{
			input:            []byte("eyJTIjoiZm9vIn0="), // base64 for {"S":"foo"}
			lastEvaluatedKey: nil,
			expectedErr:      ErrInvalidInputToDecode,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var key LastEvaluatedKey
			err := key.Bind(test.input)
			assert.Equal(t, test.lastEvaluatedKey, key)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
