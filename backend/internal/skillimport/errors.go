package skillimport

import (
	"errors"
	"fmt"
	"time"
)

type ErrorKind string

const (
	ErrorInvalidConfig ErrorKind = "invalid_config"
	ErrorInvalidSource ErrorKind = "invalid_source"
	ErrorUnauthorized  ErrorKind = "unauthorized"
	ErrorForbidden     ErrorKind = "forbidden"
	ErrorNotFound      ErrorKind = "not_found"
	ErrorRateLimited   ErrorKind = "rate_limited"
	ErrorTemporary     ErrorKind = "temporary"
	ErrorIntegrity     ErrorKind = "integrity"
	ErrorUnsafe        ErrorKind = "unsafe"
	ErrorBlocked       ErrorKind = "blocked"
)

// AdapterError lets durable workers make retry decisions without parsing text.
type AdapterError struct {
	Adapter    string
	Operation  string
	Kind       ErrorKind
	StatusCode int
	RetryAfter time.Duration
	Err        error
}

func (e *AdapterError) Error() string {
	if e == nil {
		return "<nil>"
	}
	prefix := string(e.Kind)
	if e.Adapter != "" {
		prefix = e.Adapter + ": " + prefix
	}
	if e.Operation != "" {
		prefix += " during " + e.Operation
	}
	if e.Err == nil {
		return prefix
	}
	return fmt.Sprintf("%s: %v", prefix, e.Err)
}

func (e *AdapterError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewAdapterError(adapter, operation string, kind ErrorKind, err error) error {
	if err == nil {
		err = errors.New(string(kind))
	}
	return &AdapterError{Adapter: adapter, Operation: operation, Kind: kind, Err: err}
}

func ErrorKindOf(err error) ErrorKind {
	var typed *AdapterError
	if errors.As(err, &typed) {
		return typed.Kind
	}
	return ""
}

func RetryAfterOf(err error) (time.Duration, bool) {
	var typed *AdapterError
	if errors.As(err, &typed) && typed.RetryAfter > 0 {
		return typed.RetryAfter, true
	}
	return 0, false
}
