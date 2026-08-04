package parser

import (
	"testing"
)

func TestParseAddress(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		input    string
		expected ParsedAddress
	}{
		{
			name:  "Basic address with ZIP",
			input: "1005 Gravenstein Hwy 95472",
			expected: ParsedAddress{
				Number: "1005",
				Street: "Gravenstein",
				Type:   "hwy",
				ZIP:    "95472",
			},
		},
		{
			name:  "Address with comma before ZIP",
			input: "1005 Gravenstein Hwy, 95472",
			expected: ParsedAddress{
				Number: "1005",
				Street: "Gravenstein",
				Type:   "hwy",
				ZIP:    "95472",
			},
		},
		{
			name:  "Address with directional suffix",
			input: "1005 Gravenstein Hwy N, 95472",
			expected: ParsedAddress{
				Number: "1005",
				Street: "Gravenstein",
				Type:   "hwy",
				Suffix: "N",
				ZIP:    "95472",
			},
		},
		{
			name:  "Address with directional prefix",
			input: "1005 N Gravenstein Highway, Sebastopol, CA",
			expected: ParsedAddress{
				Number: "1005",
				Prefix: "N",
				Street: "Gravenstein",
				Type:   "hwy",
				City:   "Sebastopol",
				State:  "CA",
			},
		},
		{
			name:  "Address with suite",
			input: "1005 N Gravenstein Highway, Suite 500, Sebastopol, CA",
			expected: ParsedAddress{
				Number:      "1005",
				Prefix:      "N",
				Street:      "Gravenstein",
				Type:        "hwy",
				SecUnitType: "Suite",
				SecUnitNum:  "500",
				City:        "Sebastopol",
				State:       "CA",
			},
		},
		{
			name:  "Full address with all components",
			input: "1005 N Gravenstein Highway Sebastopol CA 95472",
			expected: ParsedAddress{
				Number: "1005",
				Prefix: "N",
				Street: "Gravenstein",
				Type:   "hwy",
				City:   "Sebastopol",
				State:  "CA",
				ZIP:    "95472",
			},
		},
		{
			name:  "Address with apartment number",
			input: "123 Main St Apt 4B San Francisco CA 94105",
			expected: ParsedAddress{
				Number:      "123",
				Street:      "Main",
				Type:        "st",
				SecUnitType: "Apt",
				SecUnitNum:  "4B",
				City:        "San",
				State:       "CA",
				ZIP:         "94105",
			},
		},
		{
			name:  "Address with ZIP+4",
			input: "789 Oak Avenue, Portland, OR 97201-1234",
			expected: ParsedAddress{
				Number: "789",
				Street: "Oak",
				Type:   "ave",
				City:   "Portland",
				State:  "OR",
				ZIP:    "97201",
				Plus4:  "1234",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ParseAddress(tt.input)

			if result.Number != tt.expected.Number {
				t.Errorf("Number: got %q, want %q", result.Number, tt.expected.Number)
			}
			if result.Prefix != tt.expected.Prefix {
				t.Errorf("Prefix: got %q, want %q", result.Prefix, tt.expected.Prefix)
			}
			if result.Street != tt.expected.Street {
				t.Errorf("Street: got %q, want %q", result.Street, tt.expected.Street)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Type: got %q, want %q", result.Type, tt.expected.Type)
			}
			if result.Suffix != tt.expected.Suffix {
				t.Errorf("Suffix: got %q, want %q", result.Suffix, tt.expected.Suffix)
			}
			if tt.expected.SecUnitType != "" && result.SecUnitType != tt.expected.SecUnitType {
				t.Errorf("SecUnitType: got %q, want %q", result.SecUnitType, tt.expected.SecUnitType)
			}
			if tt.expected.SecUnitNum != "" && result.SecUnitNum != tt.expected.SecUnitNum {
				t.Errorf("SecUnitNum: got %q, want %q", result.SecUnitNum, tt.expected.SecUnitNum)
			}
			if tt.expected.ZIP != "" && result.ZIP != tt.expected.ZIP {
				t.Errorf("ZIP: got %q, want %q", result.ZIP, tt.expected.ZIP)
			}
			if tt.expected.State != "" && result.State != tt.expected.State {
				t.Errorf("State: got %q, want %q", result.State, tt.expected.State)
			}
		})
	}
}

func TestParseIntersection(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		input    string
		expected ParsedIntersection
	}{
		{
			name:  "Simple intersection with 'and'",
			input: "Mission St and Valencia St",
			expected: ParsedIntersection{
				Street1: "Mission",
				Type1:   "st",
				Street2: "Valencia",
				Type2:   "st",
			},
		},
		{
			name:  "Intersection with city and state",
			input: "Mission St and Valencia St, San Francisco CA",
			expected: ParsedIntersection{
				Street1: "Mission",
				Type1:   "st",
				Street2: "Valencia",
				Type2:   "st",
				City:    "San Francisco",
				State:   "CA",
			},
		},
		{
			name:  "Intersection with & symbol",
			input: "5th Ave & Main St",
			expected: ParsedIntersection{
				Street1: "5th",
				Type1:   "ave",
				Street2: "Main",
				Type2:   "st",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ParseIntersection(tt.input)

			if result == nil {
				t.Fatal("ParseIntersection returned nil")
			}

			if result.Street1 != tt.expected.Street1 {
				t.Errorf("Street1: got %q, want %q", result.Street1, tt.expected.Street1)
			}
			if result.Type1 != tt.expected.Type1 {
				t.Errorf("Type1: got %q, want %q", result.Type1, tt.expected.Type1)
			}
			if result.Street2 != tt.expected.Street2 {
				t.Errorf("Street2: got %q, want %q", result.Street2, tt.expected.Street2)
			}
			if result.Type2 != tt.expected.Type2 {
				t.Errorf("Type2: got %q, want %q", result.Type2, tt.expected.Type2)
			}
		})
	}
}

func TestParsePoAddress(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		input    string
		expected ParsedAddress
	}{
		{
			name:  "Basic PO Box",
			input: "PO Box 1234",
			expected: ParsedAddress{
				SecUnitType: "PO Box",
				SecUnitNum:  "1234",
			},
		},
		{
			name:  "PO Box with city state ZIP",
			input: "PO Box 5678 New York NY 10001",
			expected: ParsedAddress{
				SecUnitType: "PO Box",
				SecUnitNum:  "5678",
				City:        "New York",
				State:       "NY",
				ZIP:         "10001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ParsePoAddress(tt.input)

			if result.SecUnitType != tt.expected.SecUnitType {
				t.Errorf("SecUnitType: got %q, want %q", result.SecUnitType, tt.expected.SecUnitType)
			}
			if result.SecUnitNum != tt.expected.SecUnitNum {
				t.Errorf("SecUnitNum: got %q, want %q", result.SecUnitNum, tt.expected.SecUnitNum)
			}
			if tt.expected.ZIP != "" && result.ZIP != tt.expected.ZIP {
				t.Errorf("ZIP: got %q, want %q", result.ZIP, tt.expected.ZIP)
			}
		})
	}
}

func TestParseLocation(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{
			name:         "Standard address",
			input:        "1005 N Gravenstein Hwy Sebastopol CA 95472",
			expectedType: "address",
		},
		{
			name:         "Intersection",
			input:        "Mission St and Valencia St",
			expectedType: "intersection",
		},
		{
			name:         "PO Box",
			input:        "PO Box 1234",
			expectedType: "po_box",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ParseLocation(tt.input)
			if err != nil {
				t.Fatalf("ParseLocation failed: %v", err)
			}

			if result.Type != tt.expectedType {
				t.Errorf("Type: got %q, want %q", result.Type, tt.expectedType)
			}
		})
	}
}

func TestNormalizers(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) string
		input    string
		expected string
	}{
		{"Directional north", NormalizeDirectional, "north", "N"},
		{"Directional N", NormalizeDirectional, "N", "N"},
		{"Directional northeast", NormalizeDirectional, "northeast", "NE"},
		{"Street type avenue", NormalizeStreetType, "avenue", "ave"},
		{"Street type blvd", NormalizeStreetType, "boulevard", "blvd"},
		{"State California", NormalizeState, "california", "CA"},
		{"State CA", NormalizeState, "CA", "CA"},
		{"State Texas", NormalizeState, "texas", "TX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func BenchmarkParseAddress(b *testing.B) {
	p := NewParser()
	addr := "1005 N Gravenstein Highway, Suite 500, Sebastopol, CA 95472"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.ParseAddress(addr)
	}
}

func BenchmarkParseLocation(b *testing.B) {
	p := NewParser()
	addr := "1005 N Gravenstein Highway, Suite 500, Sebastopol, CA 95472"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.ParseLocation(addr)
	}
}

// TestStreetNamesWithoutRecognizedSuffix guards against a regression where
// NormalizeStreetType returned the lower-cased input for *any* unrecognized
// word (instead of ""), which made the caller's case-sensitive comparison
// spuriously treat every capitalized, non-suffix street-name word as a
// street type - stripping it out of the street name entirely. Streets like
// "Broadway" or "Peachtree" that don't end in a suffix (St/Ave/Blvd/...)
// are common in real US addresses.
func TestStreetNamesWithoutRecognizedSuffix(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		input    string
		wantType string // parse type ("address"/"po_box"/...)
		expected ParsedAddress
	}{
		{
			name:  "Broadway has no street type suffix",
			input: "1 Broadway, New York, NY 10004",
			expected: ParsedAddress{
				Number: "1",
				Street: "Broadway",
				City:   "New York",
				State:  "NY",
				ZIP:    "10004",
			},
		},
		{
			name:  "Peachtree has no street type suffix",
			input: "123 Peachtree, Atlanta, GA 30303",
			expected: ParsedAddress{
				Number: "123",
				Street: "Peachtree",
				City:   "Atlanta",
				State:  "GA",
				ZIP:    "30303",
			},
		},
		{
			name:  "Non-English street name word (Calle Sol)",
			input: "123 Calle Sol, San Juan, PR 00901",
			expected: ParsedAddress{
				Number: "123",
				Street: "Calle Sol",
				City:   "San Juan",
				State:  "PR",
				ZIP:    "00901",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ParseAddress(tt.input)
			if result.Number != tt.expected.Number {
				t.Errorf("Number: got %q, want %q", result.Number, tt.expected.Number)
			}
			if result.Street != tt.expected.Street {
				t.Errorf("Street: got %q, want %q", result.Street, tt.expected.Street)
			}
			if result.Type != "" {
				t.Errorf("Type: got %q, want empty (word isn't a real street type)", result.Type)
			}
			if result.City != tt.expected.City {
				t.Errorf("City: got %q, want %q", result.City, tt.expected.City)
			}
			if result.State != tt.expected.State {
				t.Errorf("State: got %q, want %q", result.State, tt.expected.State)
			}
			if result.ZIP != tt.expected.ZIP {
				t.Errorf("ZIP: got %q, want %q", result.ZIP, tt.expected.ZIP)
			}
		})
	}
}

// TestNoCommaAddressWithSecondaryUnit guards against a regression where a
// comma-less address with an abbreviated street type ("St" rather than
// "Street") caused the city-boundary walkback to not recognize the street
// type as a stopping point, swallowing the street name, secondary unit
// type, and secondary unit number into the "city" field.
func TestNoCommaAddressWithSecondaryUnit(t *testing.T) {
	p := NewParser()

	result := p.ParseAddress("123 Main St Apt 4B San Francisco CA 94105")

	if result.Street != "Main" {
		t.Errorf("Street: got %q, want %q", result.Street, "Main")
	}
	if result.Type != "st" {
		t.Errorf("Type: got %q, want %q", result.Type, "st")
	}
	if result.SecUnitType != "Apt" {
		t.Errorf("SecUnitType: got %q, want %q", result.SecUnitType, "Apt")
	}
	if result.SecUnitNum != "4B" {
		t.Errorf("SecUnitNum: got %q, want %q", result.SecUnitNum, "4B")
	}
	if result.City != "San Francisco" {
		t.Errorf("City: got %q, want %q", result.City, "San Francisco")
	}
}

// TestCommaBeforeBareZIP guards against a regression where an address with
// a comma but no recognizable trailing state (e.g. "Street, ZIP" once the
// ZIP has already been extracted, leaving an empty trailing comma-part)
// caused the entire street portion to be misassigned to "city" and
// discarded from further parsing, leaving Number/Street/Type all empty.
func TestCommaBeforeBareZIP(t *testing.T) {
	p := NewParser()

	result := p.ParseAddress("1005 Gravenstein Hwy, 95472")

	if result.Number != "1005" {
		t.Errorf("Number: got %q, want %q", result.Number, "1005")
	}
	if result.Street != "Gravenstein" {
		t.Errorf("Street: got %q, want %q", result.Street, "Gravenstein")
	}
	if result.Type != "hwy" {
		t.Errorf("Type: got %q, want %q", result.Type, "hwy")
	}
	if result.ZIP != "95472" {
		t.Errorf("ZIP: got %q, want %q", result.ZIP, "95472")
	}
}

// TestIntersectionWithAmpersand guards against a regression where the
// corner-detection regex wrapped "&"/"@" in \b word-boundary assertions.
// \b is unsatisfiable around a non-word character that's itself surrounded
// by non-word characters (e.g. a space on each side), so "5th Ave & Main
// St" - the normal, space-separated way "&" appears - never matched.
func TestIntersectionWithAmpersand(t *testing.T) {
	p := NewParser()

	result := p.ParseIntersection("5th Ave & Main St")
	if result == nil {
		t.Fatal("ParseIntersection returned nil")
	}
	if result.Street1 != "5th" || result.Type1 != "ave" {
		t.Errorf("Street1/Type1: got %q/%q, want %q/%q", result.Street1, result.Type1, "5th", "ave")
	}
	if result.Street2 != "Main" || result.Type2 != "st" {
		t.Errorf("Street2/Type2: got %q/%q, want %q/%q", result.Street2, result.Type2, "Main", "st")
	}
}

// TestIntersectionCityStateWithoutComma guards against a regression where
// a city+state combo not itself separated by a comma (e.g. "San Francisco
// CA" following "Valencia St,") caused the whole segment to be dropped
// instead of just the state token, losing the city and leaving state/city
// junk stuck onto Street2/Type2.
func TestIntersectionCityStateWithoutComma(t *testing.T) {
	p := NewParser()

	result := p.ParseIntersection("Mission St and Valencia St, San Francisco CA")
	if result == nil {
		t.Fatal("ParseIntersection returned nil")
	}
	if result.Street2 != "Valencia" {
		t.Errorf("Street2: got %q, want %q", result.Street2, "Valencia")
	}
	if result.Type2 != "st" {
		t.Errorf("Type2: got %q, want %q", result.Type2, "st")
	}
	if result.City != "San Francisco" {
		t.Errorf("City: got %q, want %q", result.City, "San Francisco")
	}
	if result.State != "CA" {
		t.Errorf("State: got %q, want %q", result.State, "CA")
	}
}
