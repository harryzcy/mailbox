package idutil

import (
	"strings"

	"github.com/google/uuid"
)

func GenerateThreadID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
