package format

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractTypeYearMonth parses type-year-month string and returns EmailType and year-month
func ExtractTypeYearMonth(s string) (emailType string, yearMonth string, err error) {
	parts := strings.Split(s, "#")
	if len(parts) != 2 {
		fmt.Printf("ExtractTypeYearMonth(%s) failed: expecting type#year-month format\n", s)
		return "", "", ErrInvalidFormatForTypeYearMonth
	}

	yearMonthParts := strings.Split(parts[1], "-")
	if len(yearMonthParts) != 2 {
		fmt.Printf("ExtractTypeYearMonth(%s) failed expecting type#year-month format\n", s)
		return "", "", ErrInvalidFormatForTypeYearMonth
	}

	emailType = parts[0]
	if emailType != "inbox" && emailType != "sent" && emailType != "draft" {
		fmt.Printf("ExtractTypeYearMonth(%s) failed: type can only be 'inbox' or 'sent'\n", s)
		return "", "", ErrInvalidEmailType
	}

	if year, err := strconv.Atoi(yearMonthParts[0]); err != nil || !(year >= 1000) {
		fmt.Printf("ExtractTypeYearMonth(%s) fail: year must be 4 digit integer\n", s)
		return "", "", ErrInvalidEmailYear
	}
	if month, err := strconv.Atoi(yearMonthParts[1]); err != nil || !(month >= 1 && month <= 12) {
		fmt.Printf("ExtractTypeYearMonth(%s) failed: month must be between 1 and 12\n", s)
		return "", "", ErrInvalidEmailMonth
	}

	yearMonth = parts[1]
	return emailType, yearMonth, nil
}
