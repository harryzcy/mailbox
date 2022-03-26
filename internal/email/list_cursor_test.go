package email

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
