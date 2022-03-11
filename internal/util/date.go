package util

import (
	"net/mail"
	"strings"
	"time"
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

// FormatDateTime converts time.Time to dd-hh:mm:ss
func FormatDateTime(t time.Time) string {
	return t.UTC().Format("02-15:04:05")
}

// FormatInboxYearMonth formats time.Time to type#YYYY-MM
func FormatInboxYearMonth(t time.Time) string {
	return "inbox#" + t.UTC().Format("2006-01")
}

// RejoinDate converts year-month and date-time to RFC3399
func RejoinDate(ym string, dt string) string {
	dt = strings.Replace(dt, "-", "T", 1)
	return ym + "-" + dt + "Z"
}
