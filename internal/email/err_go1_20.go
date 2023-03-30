//go:build !go1.19

package email

import "errors"

var joinErrors = errors.Join
