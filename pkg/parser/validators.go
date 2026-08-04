package parser

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxInputLength prevents DoS via extremely long inputs
	MaxInputLength = 10000

	// MaxAddressLength is a reasonable max for a single address
	MaxAddressLength = 500
)

var (
	ErrInputTooLong      = errors.New("input exceeds maximum allowed length")
	ErrInputEmpty        = errors.New("input is empty")
	ErrInvalidCharacters = errors.New("input contains invalid characters")
	ErrInvalidUTF8       = errors.New("input is not valid UTF-8")
)

// ValidateInput performs security and sanity checks on input strings
func ValidateInput(input string) error {
	if input == "" {
		return ErrInputEmpty
	}

	// Check UTF-8 validity
	if !utf8.ValidString(input) {
		return ErrInvalidUTF8
	}

	// Check length to prevent DoS
	if len(input) > MaxInputLength {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrInputTooLong, len(input), MaxInputLength)
	}

	// Check for null bytes and other control characters that could cause issues
	if strings.ContainsAny(input, "\x00") {
		return ErrInvalidCharacters
	}

	return nil
}

// SanitizeInput removes dangerous characters and normalizes whitespace
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Normalize whitespace (tabs, newlines, etc. to single space)
	input = strings.Join(strings.Fields(input), " ")

	// Trim leading/trailing whitespace
	input = strings.TrimSpace(input)

	// Limit length for safety. Slicing at a raw byte offset can land in
	// the middle of a multi-byte UTF-8 rune (e.g. non-Latin-script city/
	// street names), producing an invalid UTF-8 tail that later gets
	// silently mangled (e.g. replaced with U+FFFD) by anything that
	// re-validates or re-encodes the string. Back off to the nearest
	// rune boundary instead.
	if len(input) > MaxAddressLength {
		input = truncateAtRuneBoundary(input, MaxAddressLength)
	}

	return input
}

// truncateAtRuneBoundary truncates s to at most maxBytes bytes without
// splitting a multi-byte UTF-8 rune. If the raw byte cut lands inside a
// multi-byte rune, that trailing partial rune is dropped entirely rather
// than left as invalid UTF-8.
func truncateAtRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r != utf8.RuneError || size != 1 {
			break // last rune decodes cleanly (valid and complete)
		}
		s = s[:len(s)-1]
	}
	return s
}

// ValidateAndSanitize combines validation and sanitization
func ValidateAndSanitize(input string) (string, error) {
	if err := ValidateInput(input); err != nil {
		return "", err
	}
	return SanitizeInput(input), nil
}
