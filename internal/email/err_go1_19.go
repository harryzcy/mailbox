//go:build go1.19

package email

type joinedError struct {
	errs []error
}

func (e *joinedError) Error() string {
	var b string
	for i, err := range e.errs {
		if i > 0 {
			b += ";"
		}
		b += err.Error()
	}
	return b
}

var joinErrors = func(errs ...error) error {
	return &joinedError{errs}
}
