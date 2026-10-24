// Package utils provides shared utility functions including common validation patterns.
package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var nameRegex = regexp.MustCompile(`^[\p{L}\p{N}\s\-_]+$`)

// ValidateName rejects empty strings and names that contain invalid characters.
// It trims whitespace before checking length and pattern, supporting Unicode letters,
// numbers, spaces, hyphens, and underscores (TASK-071, ISS-062).
func ValidateName(name string, maxLen int) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxLen {
		return fmt.Errorf("name exceeds maximum length of %d", maxLen)
	}
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("name contains invalid characters")
	}
	return nil
}

// ValidateLength checks if a string falls within the specified length bounds.
// minLen must be >= 0; if zero no minimum check applies. maxLen must be > 0 or -1 for unlimited.
func ValidateLength(s string, minLen, maxLen int) error {
	if minLen < 0 {
		return fmt.Errorf("minLen must be non-negative")
	}
	if maxLen != -1 && maxLen < 0 {
		return fmt.Errorf("maxLen must be positive or -1")
	}

	s = strings.TrimSpace(s)
	if len(s) < minLen {
		return fmt.Errorf("value too short (minimum %d characters)", minLen)
	}
	if maxLen != -1 && len(s) > maxLen {
		return fmt.Errorf("value too long (maximum %d characters)", maxLen)
	}
	return nil
}

// ValidateCount verifies that a count value is positive and does not exceed a maximum.
func ValidateCount(count, maxCount int) error {
	if count <= 0 {
		return fmt.Errorf("count must be positive")
	}
	if maxCount > 0 && count > maxCount {
		return fmt.Errorf("count exceeds maximum of %d", maxCount)
	}
	return nil
}

// SanitizeString removes control characters and normalizes whitespace.
// This prevents injection of null bytes, newlines into single-line fields, etc.
func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	// Remove ASCII control characters (0x00-0x1F and 0x7F)
	s = regexp.MustCompile(`[\x00-\x1F\x7F]+`).ReplaceAllString(s, "")
	return s
}

// IsValidPublicKey checks if a byte slice has Ed25519 public key size (32 bytes).
func IsValidPublicKey(pk []byte) bool {
	return len(pk) == 32
}

// IsValidSeed checks if a seed string meets basic requirements (non-empty, reasonable length).
func IsValidSeed(seed string) bool {
	seed = strings.TrimSpace(seed)
	return len(seed) >= 3 && len(seed) <= 256
}
