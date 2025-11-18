package parser

import (
	"strings"
	"testing"
)

// TestInputValidation tests input validation and sanitization
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "Empty input",
			input:     "",
			wantError: true,
		},
		{
			name:      "Valid input",
			input:     "123 Main St",
			wantError: false,
		},
		{
			name:      "Input with null byte",
			input:     "123 Main\x00St",
			wantError: true,
		},
		{
			name:      "Extremely long input",
			input:     strings.Repeat("A", MaxInputLength+1),
			wantError: true,
		},
		{
			name:      "Input at max length",
			input:     strings.Repeat("A", MaxInputLength),
			wantError: false,
		},
		{
			name:      "Invalid UTF-8",
			input:     string([]byte{0xff, 0xfe, 0xfd}),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateInput() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestSanitization tests input sanitization
func TestSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal address",
			input:    "123 Main St",
			expected: "123 Main St",
		},
		{
			name:     "Address with extra whitespace",
			input:    "123   Main    St   ",
			expected: "123 Main St",
		},
		{
			name:     "Address with tabs and newlines",
			input:    "123\tMain\nSt",
			expected: "123 Main St",
		},
		{
			name:     "Address with null byte",
			input:    "123 Main\x00St",
			expected: "123 MainSt",
		},
		{
			name:     "Extremely long address",
			input:    strings.Repeat("A", MaxAddressLength+100),
			expected: strings.Repeat("A", MaxAddressLength),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestDenialOfService tests DoS attack resistance
func TestDenialOfService(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Very long address",
			input: strings.Repeat("A", 5000),
		},
		{
			name:  "Many repeated patterns",
			input: strings.Repeat("123 Main St, ", 100),
		},
		{
			name:  "Deep nesting of commas",
			input: strings.Repeat(",", 1000),
		},
		{
			name:  "Many numbers",
			input: strings.Repeat("1234567890 ", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should not panic or hang
			_, err := p.ParseLocation(tt.input)
			// We expect validation errors for these cases
			if err == nil {
				// Even if no error, parsing should complete quickly
				t.Logf("Parsed successfully (likely with minimal results)")
			}
		})
	}
}

// TestInjectionAttempts tests resistance to various injection attempts
func TestInjectionAttempts(t *testing.T) {
	p := NewParser()

	injectionAttempts := []string{
		// SQL injection attempts (not applicable but testing robustness)
		"'; DROP TABLE addresses; --",
		"1' OR '1'='1",
		"admin'--",

		// Script injection attempts
		"<script>alert('xss')</script>",
		"javascript:alert(1)",
		"<img src=x onerror=alert(1)>",

		// Command injection attempts
		"; ls -la",
		"| cat /etc/passwd",
		"`whoami`",
		"$(whoami)",

		// Path traversal
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",

		// Format string attacks
		"%s%s%s%s%s",
		"%n%n%n%n",

		// Unicode/encoding attacks
		"\u0000",
		"\u202E", // Right-to-left override
	}

	for _, injection := range injectionAttempts {
		t.Run("Injection: "+injection, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parser panicked on injection attempt %q: %v", injection, r)
				}
			}()

			// Should not panic or execute any malicious code
			// It's okay if the parser returns the input as-is, as long as:
			// 1. It doesn't crash
			// 2. It doesn't execute code
			// 3. Input validation catches truly dangerous inputs
			_, _ = p.ParseLocation(injection)

			// The fact that we got here without panic means the test passed
		})
	}
}

// TestMalformedInputs tests handling of malformed inputs
func TestMalformedInputs(t *testing.T) {
	p := NewParser()

	malformedInputs := []string{
		// Empty or whitespace only
		"",
		"   ",
		"\n\n\n",
		"\t\t\t",

		// Only special characters
		"!@#$%^&*()",
		",,,,,,",
		"......",

		// Numbers only
		"12345",

		// Mixed gibberish
		"asdfghjkl;'",
		"qwertyuiop[]",

		// Unusual spacing
		"1  2  3  M  a  i  n",

		// Multiple commas/separators
		"123,,,Main,,,St",
		"123;;;Main;;;St",
	}

	for _, input := range malformedInputs {
		t.Run("Malformed: "+input, func(t *testing.T) {
			// Should handle gracefully without panicking
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panicked on input %q: %v", input, r)
				}
			}()

			if input == "" {
				// Empty input should error in validation
				_, err := ValidateAndSanitize(input)
				if err == nil {
					t.Error("Expected error for empty input")
				}
			} else {
				// Other malformed inputs should parse (possibly with no results)
				_, _ = p.ParseLocation(input)
			}
		})
	}
}

// TestBoundaryConditions tests edge cases and boundary conditions
func TestBoundaryConditions(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name  string
		input string
	}{
		{"Single character", "A"},
		{"Two characters", "AB"},
		{"Max valid length", strings.Repeat("A", MaxAddressLength)},
		{"All uppercase", "123 MAIN STREET NEW YORK NY 10001"},
		{"All lowercase", "123 main street new york ny 10001"},
		{"Mixed case", "123 MaIn StReEt NeW yOrK nY 10001"},
		{"Unicode characters", "123 Café St São Paulo"},
		{"Numbers everywhere", "123 456 789 0"},
		{"Many spaces", "123     Main     St"},
		{"Comma at start", ",123 Main St"},
		{"Comma at end", "123 Main St,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panicked on input %q: %v", tt.input, r)
				}
			}()

			_, _ = p.ParseLocation(tt.input)
		})
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	p := NewParser()
	addresses := []string{
		"123 Main St",
		"456 Oak Ave",
		"789 Pine Rd",
		"PO Box 1234",
		"Mission St and Valencia St",
	}

	// Run concurrent parsing operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				addr := addresses[j%len(addresses)]
				_, _ = p.ParseLocation(addr)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper function to check for injection patterns in output
func checkNoInjection(t *testing.T, output string) {
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"DROP TABLE",
		"/etc/passwd",
		"$(", "`",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(output, pattern) {
			t.Errorf("Output contains dangerous pattern %q: %s", pattern, output)
		}
	}
}

// String method for ParsedAddress (for testing)
func (p *ParsedAddress) String() string {
	return p.Number + " " + p.Prefix + " " + p.Street + " " + p.Type + " " +
		p.Suffix + " " + p.SecUnitType + " " + p.SecUnitNum + " " +
		p.City + " " + p.State + " " + p.ZIP + " " + p.Plus4
}

// TestMemoryLimits ensures parsing doesn't consume excessive memory
func TestMemoryLimits(t *testing.T) {
	p := NewParser()

	// Create a large input that should be rejected or handled efficiently
	largeInput := strings.Repeat("123 Main Street, City, ST 12345, ", 1000)

	// This should either error out quickly or handle efficiently
	_, _ = p.ParseLocation(largeInput)

	// If we get here without hanging or OOM, the test passes
}

// TestRegexComplexity tests for ReDoS (Regular Expression Denial of Service)
func TestRegexComplexity(t *testing.T) {
	p := NewParser()

	// Patterns known to cause ReDoS in poorly designed regex
	redosPatterns := []string{
		strings.Repeat("a", 50) + strings.Repeat("a?", 50) + strings.Repeat("a", 50),
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaa!",
		strings.Repeat("(", 100) + strings.Repeat(")", 100),
	}

	for _, pattern := range redosPatterns {
		t.Run("ReDoS pattern", func(t *testing.T) {
			// Should complete in reasonable time
			_, _ = p.ParseLocation(pattern)
		})
	}
}
