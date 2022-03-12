package email

import (
	"os"
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
	MessageID     string `dynamodbav:"messageID"`
	TypeYearMonth string `dynamodbav:"type-year-month"`
	DateTime      string `dynamodbav:"date-time"`
}

// ToTimeIndex returns TimeIndex
func (gsi GSIIndex) ToTimeIndex() (*TimeIndex, error) {
	timeIndex := new(TimeIndex)
	var err error

	timeIndex.MessageID = gsi.MessageID
	timeIndex.Type, timeIndex.TimeReceived, err = parseGSI(gsi.TypeYearMonth, gsi.DateTime)
	return timeIndex, err
}
