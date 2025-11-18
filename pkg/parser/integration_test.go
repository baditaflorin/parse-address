package parser

import (
	"strings"
	"testing"
)

// TestRealWorldAddresses tests the parser with real-world examples
// Note: This is a simplified parser focused on security.
// It handles common cases but may not match the sophistication of the original JS version.
func TestRealWorldAddresses(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{
			name:     "Standard address",
			input:    "1005 N Gravenstein Highway Sebastopol CA 95472",
			wantType: "address",
		},
		{
			name:     "Simple address",
			input:    "123 Main Street",
			wantType: "address",
		},
		{
			name:     "Intersection",
			input:    "Mission St and Valencia St",
			wantType: "intersection",
		},
		{
			name:     "PO Box",
			input:    "PO Box 1234",
			wantType: "po_box",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ParseLocation(tt.input)
			if err != nil {
				t.Fatalf("ParseLocation failed: %v", err)
			}

			if result.Type != tt.wantType {
				t.Errorf("Type: got %q, want %q", result.Type, tt.wantType)
			}

			// Verify we got some parse result
			if result.Type == "none" {
				t.Error("Failed to parse address")
			}
		})
	}
}

// TestParserDoesNotPanic ensures parser handles all inputs gracefully
func TestParserDoesNotPanic(t *testing.T) {
	p := NewParser()

	inputs := []string{
		"",
		"normal address 123 Main St",
		"'; DROP TABLE--",
		"<script>alert('xss')</script>",
		strings.Repeat("A", 5000),
	}

	for _, input := range inputs {
		t.Run("Input: "+input[:min(len(input), 50)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parser panicked: %v", r)
				}
			}()

			// Should not panic
			_, _ = p.ParseLocation(input)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
