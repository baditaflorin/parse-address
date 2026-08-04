package parser

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTitleCasePreservesMultiByteFirstRune guards against a regression
// where titleCase sliced the first character of each word by *byte*
// offset (word[:1] / word[1:]) instead of by rune. For any word starting
// with a multi-byte UTF-8 character - e.g. Puerto Rico municipios like
// "Añasco"/"Bayamón", or other accented city/street names - this split a
// multi-byte rune in half, and strings.ToUpper/ToLower on the resulting
// invalid UTF-8 silently replaced it with U+FFFD, permanently destroying
// the original character.
func TestTitleCasePreservesMultiByteFirstRune(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"óvalo", "Óvalo"},
		{"ÓVALO", "Óvalo"},
		{"örebro väg", "Örebro Väg"},
		{"café", "Café"},               // multi-byte, but not first rune - sanity check
		{"main street", "Main Street"}, // ASCII path unaffected
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := titleCase(tt.input)
			if got != tt.want {
				t.Errorf("titleCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("titleCase(%q) produced invalid UTF-8: %q", tt.input, got)
			}
			if strings.Contains(got, "�") {
				t.Errorf("titleCase(%q) = %q contains U+FFFD replacement character (corrupted rune)", tt.input, got)
			}
		})
	}
}

// TestParseAddressPreservesAccentedCity is an end-to-end check that a real
// Puerto Rico address with an accented city name round-trips through
// ParseAddress without the city name being corrupted.
func TestParseAddressPreservesAccentedCity(t *testing.T) {
	p := NewParser()
	result := p.ParseAddress("5 Calle 1, Óvalo, PR 00610")

	if result.City != "Óvalo" {
		t.Errorf("City: got %q, want %q", result.City, "Óvalo")
	}
	if !utf8.ValidString(result.City) {
		t.Errorf("City is not valid UTF-8: %q", result.City)
	}
}

// TestSanitizeInputTruncatesOnRuneBoundary guards against a regression
// where SanitizeInput truncated long input at a raw byte offset
// (input[:MaxAddressLength]), which can land in the middle of a
// multi-byte UTF-8 rune and produce an invalid UTF-8 string. Downstream
// consumers (e.g. encoding/json) silently replace the mangled tail with
// U+FFFD, corrupting user data instead of erroring or truncating cleanly.
func TestSanitizeInputTruncatesOnRuneBoundary(t *testing.T) {
	// Build input where a multi-byte rune straddles the MaxAddressLength
	// byte boundary.
	prefix := strings.Repeat("1", MaxAddressLength-2)
	input := prefix + "日本語のテスト住所です"

	got := SanitizeInput(input)

	if !utf8.ValidString(got) {
		t.Fatalf("SanitizeInput produced invalid UTF-8 (len=%d): %q", len(got), got)
	}
	if len(got) > MaxAddressLength {
		t.Errorf("SanitizeInput result exceeds MaxAddressLength: %d > %d", len(got), MaxAddressLength)
	}
}

// TestValidateAndSanitizeAlwaysReturnsValidUTF8 fuzzes the rune-boundary
// truncation across every possible cut point near the limit, to make sure
// no off-by-one leaves a dangling partial rune.
func TestValidateAndSanitizeAlwaysReturnsValidUTF8(t *testing.T) {
	base := strings.Repeat("あ", MaxAddressLength) // 3 bytes/rune, all multi-byte
	for extra := 0; extra < 6; extra++ {
		input := base[:len(base)-extra]
		sanitized, err := ValidateAndSanitize(input)
		if err != nil {
			continue
		}
		if !utf8.ValidString(sanitized) {
			t.Errorf("extra=%d: ValidateAndSanitize produced invalid UTF-8: %q", extra, sanitized)
		}
	}
}
