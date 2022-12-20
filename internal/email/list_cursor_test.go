package email

import (
	"encoding/json"
	"fmt"
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
			fmt.Println(string(encoded))

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

func TestCursor_Decode(t *testing.T) {
	tests := []struct {
		input            []byte
		lastEvaluatedKey LastEvaluatedKey
		expectedErr      error
	}{
		{
			input:            []byte("invalid"),
			lastEvaluatedKey: nil,
			expectedErr:      avutil.ErrDecodeError,
		},
		{
			input:            []byte("{\"S\":\"foo\"}"),
			lastEvaluatedKey: nil,
			expectedErr:      ErrInvalidInputToDecode,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var key LastEvaluatedKey
			err := key.Decode(test.input)
			assert.Equal(t, test.lastEvaluatedKey, key)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
