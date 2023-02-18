package format

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

// Errors
var (
	ErrInvalidFormatForTypeYearMonth = errors.New("invalid format: expecting type#year-month")
	ErrInvalidEmailType              = errors.New("invalid email type: expecting inbox or sent")
	ErrInvalidEmailYear              = errors.New("invalid email year: expecting 4 digit integer string")
	ErrInvalidEmailMonth             = errors.New("invalid email type: expecting 2 digit integer string")
)

// FormatDate formats Date from SMTP headers to RFC3399, as it's used by DynamoDB.
//
// TODO: date from Gmail produce an error
func FormatDate(date string) string {
	t, err := mail.ParseDate(date)
	if err != nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// FormatTypeYearMonth formats time.Time to type#YYYY-MM
func FormatTypeYearMonth(emailType string, t time.Time) (string, error) {
	if emailType != "inbox" && emailType != "sent" && emailType != "draft" && emailType != "thread" {
		return "", ErrInvalidEmailType
	}

	return emailType + "#" + t.UTC().Format("2006-01"), nil
}

// FormatDateTime converts time.Time to dd-hh:mm:ss
func FormatDateTime(t time.Time) string {
	return t.UTC().Format("02-15:04:05")
}

// RejoinDate converts year-month and date-time to RFC3399
func RejoinDate(ym string, dt string) string {
	dt = strings.Replace(dt, "-", "T", 1)
	return ym + "-" + dt + "Z"
}
