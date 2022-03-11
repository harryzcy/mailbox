package util

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Errors
var (
	ErrInvalidEmailType = errors.New("invalid email type")
)

// ExtractTypeYearMonth parses type-year-month string and returns EmailType and year-month
func ExtractTypeYearMonth(s string) (emailType string, yearMonth string, err error) {
	parts := strings.Split(s, "#")
	if len(parts) != 2 {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		return "", "", ErrInvalidEmailType
	}

	emailType = parts[0]
	if emailType != "inbox" && emailType != "sent" {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		return "", "", ErrInvalidEmailType
	}

	yearMonthParts := strings.Split(parts[1], "-")
	if len(yearMonthParts) != 2 {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		return "", "", ErrInvalidEmailType
	}

	if year, err := strconv.Atoi(yearMonthParts[0]); err != nil || !(year >= 1000) {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		return "", "", ErrInvalidEmailType
	}
	if month, err := strconv.Atoi(yearMonthParts[1]); err != nil || !(month >= 1 && month <= 12) {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		return "", "", ErrInvalidEmailType
	}

	yearMonth = parts[1]
	return emailType, yearMonth, nil
}
