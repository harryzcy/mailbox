package avutil

import (
	"encoding/base64"
	"strconv"
	"testing"

	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

func TestDecodeAttributeValue(t *testing.T) {
	tests := []struct {
		in          []byte
		expected    dynamodbTypes.AttributeValue
		expectedErr error
	}{
		{
			in:       []byte("{\"B\":\"dGhpcyB0ZXh0IGlzIGJhc2U2NC1lbmNvZGVk\"}"),
			expected: &dynamodbTypes.AttributeValueMemberB{Value: []byte("this text is base64-encoded")},
		},
		{
			in:       []byte("{\"BOOL\":true}"),
			expected: &dynamodbTypes.AttributeValueMemberBOOL{Value: true},
		},
		{
			in:       []byte("{\"BOOL\":false}"),
			expected: &dynamodbTypes.AttributeValueMemberBOOL{Value: false},
		},
		{
			in:       []byte("{\"BS\":[\"U3Vubnk=\",\"UmFpbnk=\",\"U25vd3k=\"]}"),
			expected: &dynamodbTypes.AttributeValueMemberBS{Value: [][]byte{[]byte("Sunny"), []byte("Rainy"), []byte("Snowy")}},
		},
		{
			in: []byte("{\"L\":[{\"S\":\"foo\"}]}"),
			expected: &dynamodbTypes.AttributeValueMemberL{Value: []dynamodbTypes.AttributeValue{
				&dynamodbTypes.AttributeValueMemberS{Value: "foo"},
			}},
		},
		{
			in: []byte("{\"M\":{\"foo\":{\"S\":\"bar\"}}}"),
			expected: &dynamodbTypes.AttributeValueMemberM{
				Value: map[string]dynamodbTypes.AttributeValue{
					"foo": &dynamodbTypes.AttributeValueMemberS{Value: "bar"},
				},
			},
		},
		{
			in:       []byte("{\"N\":\"123.45\"}"),
			expected: &dynamodbTypes.AttributeValueMemberN{Value: "123.45"},
		},
		{
			in:       []byte("{\"NS\":[\"123.45\",\"123.45\"]}"),
			expected: &dynamodbTypes.AttributeValueMemberNS{Value: []string{"123.45", "123.45"}},
		},
		{
			in:       []byte("{\"NULL\":true}"),
			expected: &dynamodbTypes.AttributeValueMemberNULL{Value: true},
		},
		{
			in:       []byte("{\"S\":\"foo\"}"),
			expected: &dynamodbTypes.AttributeValueMemberS{Value: "foo"},
		},
		{
			in:       []byte("{\"SS\":[\"foo\",\"bar\"]}"),
			expected: &dynamodbTypes.AttributeValueMemberSS{Value: []string{"foo", "bar"}},
		},

		/* errors */

		// DecodeAttributeValue
		{
			in:          []byte{},
			expectedErr: ErrDecodeError,
		},
		{
			in:          []byte("{\"S\"[]}"),
			expectedErr: ErrDecodeError,
		},
		{
			in:          []byte("{\"INVALID\":{}}"),
			expectedErr: ErrDecodeError,
		},

		// DecodeAttributeValueB
		{
			in:          []byte("{\"B\":\"XYZ}"),
			expectedErr: ErrDecodeError,
		},
		{
			in:          []byte("{\"B\":\"XYZ\"}"),
			expectedErr: base64.CorruptInputError(0),
		},

		// DecodeAttributeValueBOOL
		{
			in:          []byte("{\"BOOL\":invalid}"),
			expectedErr: ErrDecodeError,
		},

		// DecodeAttributeValueBS
		{
			in:          []byte("{\"BS\":[\"U3Vubnk=]}"),
			expectedErr: ErrDecodeError,
		},
		{
			in:          []byte("{\"BS\":[\"XYZ\"]}"),
			expectedErr: base64.CorruptInputError(0),
		},

		// DecodeAttributeValueL
		{
			in:          []byte("{\"L\":[{}]}"),
			expectedErr: ErrDecodeError,
		},

		// DecodeAttributeValueM
		{
			in:          []byte("{\"M\":{\"foo\"}}}"),
			expectedErr: ErrDecodeError,
		},
		{
			in:          []byte("{\"M\":{\"foo:{}}}}"),
			expectedErr: ErrDecodeError,
		},
		{
			in:          []byte("{\"M\":{\"foo\":{}}}"),
			expectedErr: ErrDecodeError,
		},

		// DecodeAttributeValueNS
		{
			in:          []byte("{\"NS\":[\"123.45]}"),
			expectedErr: ErrDecodeError,
		},

		// DecodeAttributeValueNULL
		{
			in:          []byte("{\"NULL\":false}"),
			expectedErr: ErrDecodeError,
		},

		// DecodeAttributeValueSS
		{
			in:          []byte("{\"SS\":[\"foo]}"),
			expectedErr: ErrDecodeError,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := DecodeAttributeValue(test.in)
			assert.Equal(t, test.expected, actual)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestDecodeAttributeValue_Error(t *testing.T) {
	in := []byte("{}")
	expectedErr := ErrDecodeError

	actual, err := DecodeAttributeValueB(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueBOOL(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueBS(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueL(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueM(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueN(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueNS(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueNULL(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueS(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)

	actual, err = DecodeAttributeValueSS(in)
	assert.Nil(t, actual)
	assert.Equal(t, expectedErr, err)
}
