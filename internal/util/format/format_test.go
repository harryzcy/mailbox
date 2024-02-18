package format

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Fri, 30 Nov 2012 06:02:48 -0700", "2012-11-30T06:02:48-07:00"},
		{"Invalid Date", ""},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := Date(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestRFC3399(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected string
	}{
		{time.Date(2021, 9, 10, 21, 57, 52, 0, time.UTC), "2021-09-10T21:57:52Z"},
		{time.Date(2021, 9, 10, 21, 57, 52, 0, time.FixedZone("UTC-7", -7*60*60)), "2021-09-10T21:57:52-07:00"},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := RFC3399(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTypeYearMonth(t *testing.T) {
	tests := []struct {
		emailType   string
		emailTime   time.Time
		expected    string
		expectedErr error
	}{
		{
			"inbox", time.Date(2022, 3, 10, 21, 57, 52, 0, time.UTC),
			"inbox#2022-03", nil,
		},
		{
			"sent", time.Date(2021, 9, 10, 21, 57, 52, 0, time.UTC),
			"sent#2021-09", nil,
		},
		{
			"draft", time.Date(2021, 9, 10, 21, 57, 52, 0, time.UTC),
			"draft#2021-09", nil,
		},
		{
			"invalid", time.Date(2021, 9, 10, 21, 57, 52, 0, time.UTC),
			"", ErrInvalidEmailType,
		},
	}

	for _, test := range tests {
		actual, err := TypeYearMonth(test.emailType, test.emailTime)
		assert.Equal(t, test.expected, actual)
		assert.Equal(t, test.expectedErr, err)
	}
}

func TestDateTime(t *testing.T) {
	tests := []struct {
		emailTime time.Time
		expected  string
	}{
		{
			time.Date(2022, 3, 10, 21, 57, 52, 0, time.UTC),
			"10-21:57:52",
		},
		{
			time.Date(2021, 9, 10, 21, 00, 00, 0, time.UTC),
			"10-21:00:00",
		},
	}

	for _, test := range tests {
		actual := DateTime(test.emailTime)
		assert.Equal(t, test.expected, actual)
	}
}

func TestRejoinDate(t *testing.T) {
	tests := []struct {
		ym       string
		dt       string
		expected string
	}{
		{"2022-03", "10-21:00:00", "2022-03-10T21:00:00Z"},
		{"2021-09", "10-21:57:52", "2021-09-10T21:57:52Z"},
	}

	for _, test := range tests {
		actual := RejoinDate(test.ym, test.dt)
		assert.Equal(t, test.expected, actual)
	}
}
