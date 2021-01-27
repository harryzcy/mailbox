package util

import (
	"errors"
	"fmt"
	"strings"
)

// Errors
var (
	ErrInvalidEmailType = errors.New("invalid email type")
)

// ExtractTypeYearMonth parses type-year-month string and returns EmailType and year-month
func ExtractTypeYearMonth(s string) (emailType string, yearMonth string, err error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		err = ErrInvalidEmailType
		return
	}
	if parts[0] != "inbox" && parts[0] != "sent" {
		fmt.Printf("ExtractTypeYearMonth(%s) is invalid\n", s)
		err = ErrInvalidEmailType
		return
	}
	emailType = parts[0]
	yearMonth = parts[1]
	return
}
