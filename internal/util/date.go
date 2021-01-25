package util

import (
	"net/mail"
	"time"
)

// FormatDate formats Date from SMTP headers to RFC3399, as it's used by DynamoDB
func FormatDate(date string) string {
	t, err := mail.ParseDate(date)
	if err != nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// FormatTimestamp converts time.Time to RFC3399
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ExtractYearMonth formats time.Time to YYYY-MM
func ExtractYearMonth(t time.Time) string {
	return t.UTC().Format("2006-01")
}
