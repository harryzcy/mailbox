package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTypeYearMonth(t *testing.T) {
	tests := []struct {
		in        string
		emailType string
		yearMonth string
		err       error
	}{
		{"inbox-2021-01", "inbox", "2021-01", nil},
		{"sent-2021-01", "sent", "2021-01", nil},
		{"invalid-2021-01", "", "", ErrInvalidEmailType},
	}

	for _, test := range tests {
		emailType, yearMonth, err := ExtractTypeYearMonth(test.in)
		assert.Equal(t, test.emailType, emailType)
		assert.Equal(t, test.yearMonth, yearMonth)
		assert.Equal(t, test.err, err)
	}
}
