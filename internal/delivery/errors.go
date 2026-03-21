package delivery

import "errors"

// RetryableError indicates the delivery can be retried.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// PermanentError indicates the delivery should not be retried.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// IsRetryable returns true if the error is retryable.
func IsRetryable(err error) bool {
	var permanent *PermanentError
	if errors.As(err, &permanent) {
		return false
	}
	return true
}
