package email

import (
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/format"
	"github.com/stretchr/testify/assert"
)

func TestToTimeIndex(t *testing.T) {
	tests := []struct {
		gsi         GSIIndex
		ti          *TimeIndex
		expectedErr error
	}{
		{
			GSIIndex{MessageID: "1", TypeYearMonth: "inbox#2022-03", DateTime: "10-20:20:20"},
			&TimeIndex{MessageID: "1", Type: "inbox", TimeReceived: "2022-03-10T20:20:20Z"},
			nil,
		},
		{
			GSIIndex{MessageID: "1", TypeYearMonth: "sent#2022-03", DateTime: "10-20:20:20"},
			&TimeIndex{MessageID: "1", Type: "sent", TimeSent: "2022-03-10T20:20:20Z"},
			nil,
		},
		{
			GSIIndex{MessageID: "1", TypeYearMonth: "draft#2022-03", DateTime: "10-20:20:20"},
			&TimeIndex{MessageID: "1", Type: "draft", TimeCreated: "2022-03-10T20:20:20Z"},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ti, err := test.gsi.ToTimeIndex()
			assert.Equal(t, test.ti, ti)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestUnmarshalGSI(t *testing.T) {
	tests := []struct {
		items                map[string]types.AttributeValue
		expectedEmailType    string
		expectedTimeReceived string
		expectedTargetErr    error // only checks for error type
	}{
		{
			items: map[string]types.AttributeValue{
				"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox#2022-03"},
				"DateTime":      &types.AttributeValueMemberS{Value: "10-10:10:10"},
			},
			expectedEmailType:    "inbox",
			expectedTimeReceived: "2022-03-10T10:10:10Z",
		},
		{
			items: map[string]types.AttributeValue{
				"TypeYearMonth": &types.AttributeValueMemberSS{Value: []string{"inbox"}},
				"DateTime":      &types.AttributeValueMemberS{Value: "10-10:10:10"},
			},
			expectedTargetErr: &attributevalue.UnmarshalTypeError{},
		},
		{
			items: map[string]types.AttributeValue{
				"TypeYearMonth": &types.AttributeValueMemberS{Value: "inbox"},
				"DateTime":      &types.AttributeValueMemberSS{Value: []string{"10-10:10:10"}},
			},
			expectedTargetErr: &attributevalue.UnmarshalTypeError{},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			emailType, timeReceived, err := unmarshalGSI(test.items)
			assert.Equal(t, test.expectedEmailType, emailType)
			assert.Equal(t, test.expectedTimeReceived, timeReceived)
			if test.expectedTargetErr == nil {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.IsType(t, err, test.expectedTargetErr)
			}
		})
	}
}

func TestParseGSI(t *testing.T) {
	tests := []struct {
		typeYearMonth        string
		datetime             string
		expectedEmailType    string
		expectedTimeReceived string
		expectedErr          error
	}{
		{
			typeYearMonth:        "inbox#2022-03",
			datetime:             "10-10:00:00",
			expectedEmailType:    "inbox",
			expectedTimeReceived: "2022-03-10T10:00:00Z",
		},
		{
			typeYearMonth: "invalid",
			expectedErr:   format.ErrInvalidFormatForTypeYearMonth,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			emailType, timeReceived, err := parseGSI(test.typeYearMonth, test.datetime)
			assert.Equal(t, test.expectedEmailType, emailType)
			assert.Equal(t, test.expectedTimeReceived, timeReceived)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
