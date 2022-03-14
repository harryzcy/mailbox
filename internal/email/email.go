package email

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/util/format"
)

var (
	// tableName represents DynamoDB table name
	tableName = os.Getenv("DYNAMODB_TABLE")
	// gsiIndexName represents DynamoDB's GSI name
	gsiIndexName = os.Getenv("DYNAMODB_TIME_INDEX")
)

// TimeIndex represents time attributes of an email
type TimeIndex struct {
	MessageID    string `json:"messageID"`
	Type         string `json:"type"`
	TimeReceived string `json:"timeReceived"`
}

// GSIIndex represents Global Secondary Index of an email
type GSIIndex struct {
	MessageID     string `dynamodbav:"MessageID"`
	TypeYearMonth string `dynamodbav:"TypeYearMonth"`
	DateTime      string `dynamodbav:"DateTime"`
}

// ToTimeIndex returns TimeIndex
func (gsi GSIIndex) ToTimeIndex() (*TimeIndex, error) {
	timeIndex := new(TimeIndex)
	var err error

	timeIndex.MessageID = gsi.MessageID
	timeIndex.Type, timeIndex.TimeReceived, err = parseGSI(gsi.TypeYearMonth, gsi.DateTime)
	return timeIndex, err
}

func unmarshalGSI(item map[string]types.AttributeValue) (emailType, timeReceived string, err error) {
	var typeYearMonth string
	var dt string // date-time
	err = attributevalue.Unmarshal(item["TypeYearMonth"], &typeYearMonth)
	if err != nil {
		fmt.Printf("unmarshal TypeYearMonth failed: %v", err)
		return
	}
	err = attributevalue.Unmarshal(item["DateTime"], &dt)
	if err != nil {
		fmt.Printf("unmarshal DateTime failed: %v", err)
		return
	}
	return parseGSI(typeYearMonth, dt)
}

func parseGSI(typeYearMonth, dt string) (emailType, timeReceived string, err error) {
	var ym string // YYYY-MM
	emailType, ym, err = format.ExtractTypeYearMonth(typeYearMonth)
	if err != nil {
		fmt.Printf("extract TypeYearMonth failed: %v\n", err)
		return
	}
	timeReceived = format.RejoinDate(ym, dt)
	return
}
