// Package lottery provides VRF-based transparent lottery functionality.
// It implements verifiable random function (VRF) to ensure fair and
// transparent winner selection that can be verified on-chain.
package lottery

import "errors"

// ErrNotFound is returned when a lottery record is not found in the
// repository. Domain errors like this allow API handlers to map
// specific error conditions to appropriate HTTP status codes.
var ErrNotFound = errors.New("lottery record not found")
