package utils

import (
	"errors"
	"fmt"
)

// WrapError wraps an error with a message.
func WrapError(err error, msg string, args ...any) error {
	if err == nil {
		return nil
	}
	return errors.Join(err, fmt.Errorf(msg, args...))
}
