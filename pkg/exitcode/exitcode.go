// Copyright © 2026 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package exitcode defines the standard exit codes used by tkn and provides
// helper types so that errors can carry their intended exit code through the
// call stack without requiring callers to parse error strings.
//
// Exit code table:
//
//	0  Success
//	1  General error / command failure
//	2  Resource not found
//	3  Invalid input / validation error
//	4  Timeout
//	5  Unauthorized / forbidden
package exitcode

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	// Success is the exit code for a successful command.
	Success = 0
	// GeneralError is the exit code for unclassified errors.
	GeneralError = 1
	// NotFound is the exit code when a requested resource does not exist.
	NotFound = 2
	// InvalidInput is the exit code for invalid flags, parameters, or input.
	InvalidInput = 3
	// Timeout is the exit code when an operation exceeds its deadline.
	Timeout = 4
	// Unauthorized is the exit code when the request is unauthorized or forbidden.
	Unauthorized = 5
)

// Error is an error that carries a specific exit code.
type Error struct {
	Code    int
	Message string
	// Err is the underlying error, if any.  It is preserved so that helpers
	// such as k8serrors.IsNotFound keep working through errors.Is/errors.As.
	Err error
}

func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the underlying error so that errors.Is and errors.As can
// inspect the original error (for example a Kubernetes status error) behind
// the exit code.
func (e *Error) Unwrap() error {
	return e.Err
}

// New creates an Error with an explicit code and formatted message.
func New(code int, format string, a ...interface{}) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, a...)}
}

// FromAPIError converts a Kubernetes API error into an Error with the
// appropriate exit code.  If err cannot be classified it is returned
// unchanged.  The original error is kept as the wrapped error so that
// k8serrors helpers (IsNotFound, IsForbidden, ...) still match it.
func FromAPIError(err error) error {
	if err == nil {
		return nil
	}
	if code, ok := classify(err); ok {
		return &Error{Code: code, Message: err.Error(), Err: err}
	}
	return err
}

// classify maps a Kubernetes API or context error to its exit code.  The
// second return value is false when err does not match any known category.
func classify(err error) (int, bool) {
	switch {
	case k8serrors.IsNotFound(err):
		return NotFound, true
	case k8serrors.IsUnauthorized(err), k8serrors.IsForbidden(err):
		return Unauthorized, true
	case k8serrors.IsInvalid(err), k8serrors.IsBadRequest(err):
		return InvalidInput, true
	case k8serrors.IsTimeout(err), k8serrors.IsServerTimeout(err),
		errors.Is(err, context.DeadlineExceeded):
		return Timeout, true
	default:
		return Success, false
	}
}

// CodeFrom returns the exit code carried by err.  A typed exitcode.Error takes
// precedence; otherwise an unclassified Kubernetes/context error is classified
// here so codes still propagate through callers that preserve the cause.  Any
// remaining error maps to GeneralError.
func CodeFrom(err error) int {
	if err == nil {
		return Success
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	if code, ok := classify(err); ok {
		return code
	}
	return GeneralError
}
