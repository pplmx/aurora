// Package errdetail formats a domain sentinel with a printable detail.
//
// Aurora's contract is that the sentinel (ErrInvalidPayload,
// ErrAuditPublishFailed, …) is what callers branch on, while the underlying
// failure is context for the operator: it is printed but must stay unwrappable,
// or a caller could start classifying parser or driver internals the domain
// never promised. `fmt.Errorf("%w: %v", …)` expresses exactly that; this package
// keeps the one place that writes it — and the reason for it — instead of
// repeating the pattern in every decoder and mutator.
package errdetail

import "fmt"

// Wrap returns an error that matches errors.Is(err, sentinel) and prints
// "sentinel: detail". The detail is context: it never enters the error chain.
func Wrap(sentinel, detail error) error {
	// The non-wrapping verb is the point: errors.Is must only reach the
	// sentinel, never the JSON parser or the driver.
	return fmt.Errorf("%w: %v", sentinel, detail) //nolint:errorlint // see above
}
