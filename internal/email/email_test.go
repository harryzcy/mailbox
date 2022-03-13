package email

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTimeIndex(t *testing.T) {
	tests := []struct {
		gsi         GSIIndex
		ti          *TimeIndex
		expectedErr error
	}{
		{
			GSIIndex{MessageID: "1", TypeYearMonth: "inbox#2022-03", DateTime: "10-20:20:20"},
			&TimeIndex{MessageID: "1", Type: "inbox", TimeReceived: "2022-03-10T20:20:20Z"},
			nil,
		},
		{
			GSIIndex{MessageID: "1", TypeYearMonth: "sent#2022-03", DateTime: "10-20:20:20"},
			&TimeIndex{MessageID: "1", Type: "sent", TimeReceived: "2022-03-10T20:20:20Z"},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ti, err := test.gsi.ToTimeIndex()
			assert.Equal(t, test.ti, ti)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
