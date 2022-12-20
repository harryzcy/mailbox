package avutil

import (
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAttributeValue(t *testing.T) {
	tests := []struct {
		in       types.AttributeValue
		expected []byte
	}{
		{
			in:       &types.AttributeValueMemberB{Value: []byte("this text is base64-encoded")},
			expected: []byte("{\"B\":\"dGhpcyB0ZXh0IGlzIGJhc2U2NC1lbmNvZGVk\"}"),
		},
		{
			in:       &types.AttributeValueMemberBOOL{Value: true},
			expected: []byte("{\"BOOL\":true}"),
		},
		{
			in:       &types.AttributeValueMemberBS{Value: [][]byte{[]byte("Sunny"), []byte("Rainy"), []byte("Snowy")}},
			expected: []byte("{\"BS\":[\"U3Vubnk=\",\"UmFpbnk=\",\"U25vd3k=\"]}"),
		},
		{
			in: &types.AttributeValueMemberL{Value: []types.AttributeValue{
				&types.AttributeValueMemberS{Value: "foo"},
			}},
			expected: []byte("{\"L\":[{\"S\":\"foo\"}]}"),
		},
		{
			in: &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"foo":  &types.AttributeValueMemberS{Value: "bar"},
					"foo2": &types.AttributeValueMemberS{Value: "bar2"},
				},
			},
			expected: []byte("{\"M\":{\"foo\":{\"S\":\"bar\"},\"foo2\":{\"S\":\"bar2\"}}}"),
		},
		{
			in:       &types.AttributeValueMemberN{Value: "123.45"},
			expected: []byte("{\"N\":\"123.45\"}"),
		},
		{
			in:       &types.AttributeValueMemberNS{Value: []string{"123.45", "123.45"}},
			expected: []byte("{\"NS\":[\"123.45\",\"123.45\"]}"),
		},
		{
			in:       &types.AttributeValueMemberNULL{Value: true},
			expected: []byte("{\"NULL\":true}"),
		},
		{
			in:       &types.AttributeValueMemberS{Value: "foo"},
			expected: []byte("{\"S\":\"foo\"}"),
		},
		{
			in:       &types.AttributeValueMemberSS{Value: []string{"foo", "bar"}},
			expected: []byte("{\"SS\":[\"foo\",\"bar\"]}"),
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := EncodeAttributeValue(test.in)
			assert.Equal(t, test.expected, actual)
		})
	}
}
