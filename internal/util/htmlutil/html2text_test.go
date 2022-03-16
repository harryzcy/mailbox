package htmlutil

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateText(t *testing.T) {
	tests := []struct {
		html        string
		text        string
		expectedErr error
	}{
		{
			html: `<p>Title</p>`,
			text: "Title",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			text, err := GenerateText(test.html)
			assert.Equal(t, test.text, text)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
