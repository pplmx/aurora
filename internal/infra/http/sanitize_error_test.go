package http

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The reference promises that a failed fetch reports the HTTP status or the
// underlying cause and never echoes the source URL: userinfo, path segments and
// query parameters can all carry credentials, and `last_error` is served to any
// API-key holder and copied into the scheduler log (docs/api.md, "Oracle").
// sanitizeFetchError is what keeps that promise, and nothing tested it.

// secretURL is a source URL whose every credential-bearing part is
// recognisable, so a leak names itself in the failure message.
const (
	secretUser   = "svc-account"
	secretPass   = "s3cr3t-password"
	secretPath   = "internal-billing-export"
	secretQuery  = "api_key=AKIAIOSFODNN7EXAMPLE"
	secretSource = "https://" + secretUser + ":" + secretPass + "@api.example.com/" + secretPath + "?" + secretQuery
)

// requireNoSecrets asserts that an error carries none of the URL's
// credential-bearing parts. The host stays acceptable: it is the one part a
// caller already configured in clear text.
func requireNoSecrets(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	msg := err.Error()
	for _, part := range []string{secretUser, secretPass, secretPath, "topsecret", "AKIAIOSFODNN7EXAMPLE"} {
		assert.NotContainsf(t, msg, part, "the error must not echo %q from the source URL", part)
	}
	assert.NotContains(t, msg, secretSource, "the complete source URL must never reach the caller")
}

// TestSanitizeFetchErrorDropsTheURL covers the shape net/http actually
// produces: a *url.Error whose Err is the transport failure.
func TestSanitizeFetchErrorDropsTheURL(t *testing.T) {
	cause := errors.New("connection refused")
	urlErr := &url.Error{Op: "Get", URL: secretSource, Err: cause}

	err := sanitizeFetchError(urlErr)

	requireNoSecrets(t, err)
	assert.ErrorIs(t, err, cause, "the cause must survive so sentinel checks still work")
	assert.Equal(t, "request failed: connection refused", err.Error())
}

// TestSanitizeFetchErrorKeepsNestedSentinels is the reason the function
// recurses: the SSRF guard's sentinel is what the API classifies, and a wrapped
// url.Error that dropped it would report a blocked destination as a generic
// 500.
func TestSanitizeFetchErrorKeepsNestedSentinels(t *testing.T) {
	inner := fmt.Errorf("%w: dial target %q", ErrBlockedDestination, "10.0.0.5:80")

	err := sanitizeFetchError(&url.Error{Op: "Get", URL: secretSource, Err: inner})

	requireNoSecrets(t, err)
	assert.ErrorIs(t, err, ErrBlockedDestination, "a blocked destination must stay classifiable")
}

// TestSanitizeFetchErrorHandlesNestedURLErrors covers a redirect chain: the
// transport can wrap one *url.Error in another, and stopping at the outer one
// would leave the inner URL in place.
func TestSanitizeFetchErrorHandlesNestedURLErrors(t *testing.T) {
	inner := &url.Error{Op: "Get", URL: secretSource + "&page=2", Err: errors.New("timeout")}

	err := sanitizeFetchError(&url.Error{Op: "Get", URL: secretSource, Err: inner})

	requireNoSecrets(t, err)
	assert.Contains(t, err.Error(), "timeout", "the underlying cause is the useful part")
	assert.Equal(t, "request failed: request failed: timeout", err.Error())
}

// TestSanitizeFetchErrorLeavesOtherErrorsAlone keeps the function from
// rewriting errors it did not produce: a validation failure already names the
// field, not the URL, and its message is what the caller needs.
func TestSanitizeFetchErrorLeavesOtherErrorsAlone(t *testing.T) {
	plain := errors.New("invalid source: method must be an HTTP token")
	assert.Equal(t, plain, sanitizeFetchError(plain))

	sentinel := fmt.Errorf("%w: empty body", ErrHTTPError)
	assert.ErrorIs(t, sanitizeFetchError(sentinel), ErrHTTPError)
	assert.Contains(t, sanitizeFetchError(sentinel).Error(), "empty body")
}

// TestSanitizeFetchErrorFillsAnEmptyCause covers a *url.Error with no wrapped
// error, where formatting the nil cause would print %!w(<nil>).
func TestSanitizeFetchErrorFillsAnEmptyCause(t *testing.T) {
	err := sanitizeFetchError(&url.Error{Op: "Get", URL: secretSource})

	requireNoSecrets(t, err)
	assert.Equal(t, "request failed: request failed", err.Error())
}

// TestDialFailureDoesNotLeakTheURL exercises the same promise through the error
// the transport really returns for a refused connection. It uses the guard's
// own path rather than a live network call, so the test stays offline.
func TestDialFailureDoesNotLeakTheURL(t *testing.T) {
	urlErr := &url.Error{
		Op:  "Get",
		URL: secretSource,
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connect: connection refused"),
		},
	}

	err := sanitizeFetchError(urlErr)

	requireNoSecrets(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}
