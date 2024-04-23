package avutil

import (
	"bytes"
	"strconv"
	"testing"

	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAttributeValue(t *testing.T) {
	tests := []struct {
		in       dynamodbTypes.AttributeValue
		expected []byte
	}{
		{
			in:       &dynamodbTypes.AttributeValueMemberB{Value: []byte("this text is base64-encoded")},
			expected: []byte("{\"B\":\"dGhpcyB0ZXh0IGlzIGJhc2U2NC1lbmNvZGVk\"}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberBOOL{Value: true},
			expected: []byte("{\"BOOL\":true}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberBS{Value: [][]byte{[]byte("Sunny"), []byte("Rainy"), []byte("Snowy")}},
			expected: []byte("{\"BS\":[\"U3Vubnk=\",\"UmFpbnk=\",\"U25vd3k=\"]}"),
		},
		{
			in: &dynamodbTypes.AttributeValueMemberL{Value: []dynamodbTypes.AttributeValue{
				&dynamodbTypes.AttributeValueMemberS{Value: "foo"},
			}},
			expected: []byte("{\"L\":[{\"S\":\"foo\"}]}"),
		},
		{
			in: &dynamodbTypes.AttributeValueMemberM{
				Value: map[string]dynamodbTypes.AttributeValue{
					"foo": &dynamodbTypes.AttributeValueMemberS{Value: "bar"},
				},
			},
			expected: []byte("{\"M\":{\"foo\":{\"S\":\"bar\"}}}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberN{Value: "123.45"},
			expected: []byte("{\"N\":\"123.45\"}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberNS{Value: []string{"123.45", "123.45"}},
			expected: []byte("{\"NS\":[\"123.45\",\"123.45\"]}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberNULL{Value: true},
			expected: []byte("{\"NULL\":true}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberNULL{Value: false},
			expected: []byte("{\"NULL\":false}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberS{Value: "foo"},
			expected: []byte("{\"S\":\"foo\"}"),
		},
		{
			in:       &dynamodbTypes.AttributeValueMemberSS{Value: []string{"foo", "bar"}},
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

func TestAttributeValueMemberM(t *testing.T) {
	// test that the order of the map is not important
	m := &dynamodbTypes.AttributeValueMemberM{
		Value: map[string]dynamodbTypes.AttributeValue{
			"foo":  &dynamodbTypes.AttributeValueMemberS{Value: "bar"},
			"foo2": &dynamodbTypes.AttributeValueMemberS{Value: "bar2"},
		},
	}
	expected := []byte("{\"M\":{\"foo\":{\"S\":\"bar\"},\"foo2\":{\"S\":\"bar2\"}}}")
	expected2 := []byte("{\"M\":{\"foo2\":{\"S\":\"bar2\"},\"foo\":{\"S\":\"bar\"}}}")
	actual := EncodeAttributeValue(m)
	assert.True(t, bytes.Equal(expected, actual) || bytes.Equal(expected2, actual))
}
