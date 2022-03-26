package email

import (
	"encoding/base64"
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
		{Cursor{LastEvaluatedKey: map[string]types.AttributeValue{}}},
		{Cursor{LastEvaluatedKey: map[string]types.AttributeValue{
			"foo": &types.AttributeValueMemberS{Value: "bar"},
		}}},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			encoded, err := json.Marshal(test.cursor)
			assert.Nil(t, err)

			var decoded Cursor
			err = json.Unmarshal(encoded, &decoded)
			assert.Nil(t, err)
		})
	}
}

func TestCursor_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		input       []byte
		cursor      Cursor
		expectedErr error
	}{
		{[]byte{}, Cursor{}, ErrInvalidInputToUnmarshal},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var cursor Cursor
			err := cursor.UnmarshalJSON(test.input)
			assert.Equal(t, test.cursor, cursor)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestCursor_BindString(t *testing.T) {
	tests := []struct {
		input       string
		cursor      Cursor
		expectedErr error
	}{
		{"", Cursor{}, nil},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var cursor Cursor
			err := cursor.BindString(test.input)
			assert.Equal(t, test.cursor, cursor)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestCursor_Bind(t *testing.T) {
	tests := []struct {
		input       []byte
		cursor      Cursor
		expectedErr error
	}{
		{
			input:       []byte("aW52YWxpZA=="), // base64 for invalid
			cursor:      Cursor{},
			expectedErr: avutil.ErrDecodeError,
		},
		{
			input:       []byte("XYZ"),
			cursor:      Cursor{},
			expectedErr: base64.CorruptInputError(0),
		},
		{
			input:       []byte("eyJTIjoiZm9vIn0="), // base64 for {"S":"foo"}
			cursor:      Cursor{},
			expectedErr: ErrInvalidInputToDecode,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var cursor Cursor
			err := cursor.Bind(test.input)
			assert.Equal(t, test.cursor, cursor)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
