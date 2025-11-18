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
	ErrInputTooLong     = errors.New("input exceeds maximum allowed length")
	ErrInputEmpty       = errors.New("input is empty")
	ErrInvalidCharacters = errors.New("input contains invalid characters")
	ErrInvalidUTF8      = errors.New("input is not valid UTF-8")
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

	// Limit length for safety
	if len(input) > MaxAddressLength {
		input = input[:MaxAddressLength]
	}

	return input
}

// ValidateAndSanitize combines validation and sanitization
func ValidateAndSanitize(input string) (string, error) {
	if err := ValidateInput(input); err != nil {
		return "", err
	}
	return SanitizeInput(input), nil
}
