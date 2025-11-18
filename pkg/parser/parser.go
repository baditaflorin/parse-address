package parser

import (
	"regexp"
	"strings"
)

// Parser handles address parsing operations
type Parser struct {
	initialized bool
	patterns    *regexPatterns
}

type regexPatterns struct {
	number       *regexp.Regexp
	street       *regexp.Regexp
	city         *regexp.Regexp
	state        *regexp.Regexp
	zip          *regexp.Regexp
	secUnit      *regexp.Regexp
	corner       *regexp.Regexp
	poBox        *regexp.Regexp
	directional  *regexp.Regexp
}

// NewParser creates a new address parser
func NewParser() *Parser {
	p := &Parser{}
	p.init()
	return p
}

// init initializes the parser's regex patterns
func (p *Parser) init() {
	if p.initialized {
		return
	}
	p.initialized = true

	// Build regex patterns
	p.patterns = &regexPatterns{
		// Street number: digits with optional hyphen, or grid coordinates
		number: regexp.MustCompile(`(?i)^[^\w#]*(\d+[\-]?\d*|[NSEW]\d{1,3}[NSEW]\d{1,6})\b`),

		// ZIP code: 5 digits with optional +4
		zip: regexp.MustCompile(`(?i)\b(\d{5})(?:[-\s]?(\d{4}))?\b`),

		// State: 2-letter abbreviation
		state: regexp.MustCompile(`(?i)\b([A-Z]{2})\b`),

		// Secondary unit: Apt, Suite, Unit, #, etc.
		secUnit: regexp.MustCompile(`(?i)(?:\b(apt|apartment|suite|ste|unit|#|room|rm|floor|fl|building|bldg)\W*([a-z0-9\-]+)|(\bbasement\b|\bfront\b|\brear\b))`),

		// Intersection indicators
		corner: regexp.MustCompile(`(?i)\b(and|at|&|@)\b`),

		// PO Box
		poBox: regexp.MustCompile(`(?i)^[^\w]*p\W*(?:o|ost\s*office)?\W*box\W*(\d+)`),

		// Directional prefixes/suffixes
		directional: regexp.MustCompile(`(?i)\b(north|south|east|west|northeast|northwest|southeast|southwest|n|s|e|w|ne|nw|se|sw)\.?\b`),

		// City (simple pattern - alphanumeric with spaces, commas)
		city: regexp.MustCompile(`(?i)([a-z][a-z\s]+)`),

		// Street will be handled in parsing logic
	}
}

// ParseLocation is the main entry point - intelligently routes to appropriate parser
func (p *Parser) ParseLocation(address string) (*ParseResult, error) {
	// Validate and sanitize input
	sanitized, err := ValidateAndSanitize(address)
	if err != nil {
		return nil, err
	}

	// Check for intersection
	if p.patterns.corner.MatchString(sanitized) {
		intersection := p.ParseIntersection(sanitized)
		if intersection != nil && intersection.Street1 != "" {
			return &ParseResult{
				Type:         "intersection",
				Intersection: intersection,
			}, nil
		}
	}

	// Check for PO Box
	if p.patterns.poBox.MatchString(sanitized) {
		addr := p.ParsePoAddress(sanitized)
		if addr != nil && !addr.IsEmpty() {
			return &ParseResult{
				Type:    "po_box",
				Address: addr,
			}, nil
		}
	}

	// Try standard address parsing
	addr := p.ParseAddress(sanitized)
	if addr != nil && !addr.IsEmpty() {
		return &ParseResult{
			Type:    "address",
			Address: addr,
		}, nil
	}

	// Fall back to informal address parsing
	addr = p.ParseInformalAddress(sanitized)
	if addr != nil && !addr.IsEmpty() {
		return &ParseResult{
			Type:    "address",
			Address: addr,
		}, nil
	}

	return &ParseResult{
		Type: "none",
	}, nil
}

// ParseAddress parses a standard street address
func (p *Parser) ParseAddress(address string) *ParsedAddress {
	result := &ParsedAddress{}

	// Extract ZIP code
	if matches := p.patterns.zip.FindStringSubmatch(address); len(matches) > 0 {
		result.ZIP = matches[1]
		if len(matches) > 2 && matches[2] != "" {
			result.Plus4 = matches[2]
		}
		// Remove ZIP from address for further parsing
		address = p.patterns.zip.ReplaceAllString(address, "")
	}

	// Extract state
	parts := strings.Split(address, ",")
	if len(parts) >= 2 {
		// State is usually in the last part after city
		lastPart := strings.TrimSpace(parts[len(parts)-1])
		if matches := p.patterns.state.FindStringSubmatch(lastPart); len(matches) > 0 {
			result.State = NormalizeState(matches[1])
			address = strings.TrimSuffix(address, lastPart)
			address = strings.TrimSuffix(address, ",")
		}

		// City is usually the second-to-last part
		if len(parts) >= 2 {
			result.City = strings.TrimSpace(parts[len(parts)-2])
			address = strings.Join(parts[:len(parts)-2], ",")
		}
	} else if len(parts) == 1 {
		// Try to extract state from a single line
		words := strings.Fields(address)
		if len(words) >= 2 {
			// Check last few words for state
			for i := len(words) - 1; i >= 0 && i >= len(words)-3; i-- {
				if matches := p.patterns.state.FindStringSubmatch(words[i]); len(matches) > 0 {
					result.State = NormalizeState(matches[1])
					// City might be the words before state
					if i > 0 {
						cityEnd := i
						cityStart := cityEnd - 1
						// Find where city starts (after street type or number)
						for cityStart > 0 {
							word := strings.ToLower(words[cityStart-1])
							if _, isType := StreetType[word]; isType {
								break
							}
							if p.patterns.number.MatchString(words[cityStart-1]) {
								break
							}
							cityStart--
						}
						result.City = strings.Join(words[cityStart:cityEnd], " ")
						// Rebuild address without city/state
						address = strings.Join(words[:cityStart], " ")
					}
					break
				}
			}
		}
	}

	// Extract secondary unit (apartment, suite, etc.)
	if matches := p.patterns.secUnit.FindStringSubmatch(address); len(matches) > 0 {
		if matches[1] != "" {
			result.SecUnitType = strings.TrimSpace(matches[1])
			if len(matches) > 2 && matches[2] != "" {
				result.SecUnitNum = strings.TrimSpace(matches[2])
			}
		} else if matches[3] != "" {
			result.SecUnitType = strings.TrimSpace(matches[3])
		}
		address = p.patterns.secUnit.ReplaceAllString(address, " ")
	}

	// Extract street number
	if matches := p.patterns.number.FindStringSubmatch(address); len(matches) > 0 {
		result.Number = strings.TrimSpace(matches[1])
		// Replace only the first match
		address = strings.Replace(address, matches[0], "", 1)
	}

	// Parse remaining street components
	address = strings.TrimSpace(address)
	words := strings.Fields(address)

	if len(words) == 0 {
		result.Normalize()
		return result
	}

	// Check for directional prefix
	if len(words) > 0 {
		if dir := NormalizeDirectional(words[0]); dir != "" {
			result.Prefix = dir
			words = words[1:]
		}
	}

	// Check for directional suffix (from end)
	if len(words) > 0 {
		if dir := NormalizeDirectional(words[len(words)-1]); dir != "" {
			result.Suffix = dir
			words = words[:len(words)-1]
		}
	}

	// Check for street type (from end)
	if len(words) > 0 {
		streetType := NormalizeStreetType(words[len(words)-1])
		if streetType != "" && streetType != words[len(words)-1] {
			result.Type = streetType
			words = words[:len(words)-1]
		}
	}

	// Remaining words are the street name
	if len(words) > 0 {
		result.Street = strings.Join(words, " ")
	}

	result.Normalize()
	return result
}

// ParseInformalAddress parses informal address formats
func (p *Parser) ParseInformalAddress(address string) *ParsedAddress {
	// For informal addresses, we're more lenient
	// Try the standard parser first
	result := p.ParseAddress(address)

	// If we got minimal results, try to extract what we can
	if result.Number == "" && result.Street == "" {
		// Try to find any street-like component
		words := strings.Fields(address)
		if len(words) > 0 {
			result.Street = strings.Join(words, " ")
		}
	}

	return result
}

// ParsePoAddress parses PO Box addresses
func (p *Parser) ParsePoAddress(address string) *ParsedAddress {
	result := &ParsedAddress{}

	// Extract PO Box
	if matches := p.patterns.poBox.FindStringSubmatch(address); len(matches) > 0 {
		result.SecUnitType = "PO Box"
		if len(matches) > 1 {
			result.SecUnitNum = matches[1]
		}
		address = p.patterns.poBox.ReplaceAllString(address, "")
	}

	// Extract ZIP, state, city from remaining address
	if matches := p.patterns.zip.FindStringSubmatch(address); len(matches) > 0 {
		result.ZIP = matches[1]
		if len(matches) > 2 && matches[2] != "" {
			result.Plus4 = matches[2]
		}
		address = p.patterns.zip.ReplaceAllString(address, "")
	}

	// Extract state
	if matches := p.patterns.state.FindStringSubmatch(address); len(matches) > 0 {
		result.State = NormalizeState(matches[1])
		address = p.patterns.state.ReplaceAllString(address, "")
	}

	// Remaining is likely the city
	city := strings.TrimSpace(address)
	city = strings.Trim(city, ",")
	if city != "" {
		result.City = city
	}

	result.Normalize()
	return result
}

// ParseIntersection parses street intersection addresses
func (p *Parser) ParseIntersection(address string) *ParsedIntersection {
	result := &ParsedIntersection{}

	// Split on intersection markers
	parts := p.patterns.corner.Split(address, 2)
	if len(parts) != 2 {
		return nil
	}

	street1 := strings.TrimSpace(parts[0])
	street2 := strings.TrimSpace(parts[1])

	// Parse first street
	words1 := strings.Fields(street1)
	if len(words1) > 0 {
		if dir := NormalizeDirectional(words1[0]); dir != "" {
			result.Prefix1 = dir
			words1 = words1[1:]
		}
		if len(words1) > 0 && NormalizeDirectional(words1[len(words1)-1]) != "" {
			result.Suffix1 = NormalizeDirectional(words1[len(words1)-1])
			words1 = words1[:len(words1)-1]
		}
		if len(words1) > 0 {
			streetType := NormalizeStreetType(words1[len(words1)-1])
			if streetType != "" && streetType != words1[len(words1)-1] {
				result.Type1 = streetType
				words1 = words1[:len(words1)-1]
			}
		}
		if len(words1) > 0 {
			result.Street1 = strings.Join(words1, " ")
		}
	}

	// Parse second street (may contain city/state/zip)
	// Extract city/state/zip first
	if matches := p.patterns.zip.FindStringSubmatch(street2); len(matches) > 0 {
		result.ZIP = matches[1]
		street2 = p.patterns.zip.ReplaceAllString(street2, "")
	}

	streetParts := strings.Split(street2, ",")
	if len(streetParts) > 1 {
		// Last part might have state
		lastPart := strings.TrimSpace(streetParts[len(streetParts)-1])
		if matches := p.patterns.state.FindStringSubmatch(lastPart); len(matches) > 0 {
			result.State = NormalizeState(matches[1])
			streetParts = streetParts[:len(streetParts)-1]
		}
		if len(streetParts) > 1 {
			result.City = strings.TrimSpace(streetParts[len(streetParts)-1])
			street2 = strings.TrimSpace(streetParts[0])
		}
	}

	words2 := strings.Fields(street2)
	if len(words2) > 0 {
		if dir := NormalizeDirectional(words2[0]); dir != "" {
			result.Prefix2 = dir
			words2 = words2[1:]
		}
		if len(words2) > 0 && NormalizeDirectional(words2[len(words2)-1]) != "" {
			result.Suffix2 = NormalizeDirectional(words2[len(words2)-1])
			words2 = words2[:len(words2)-1]
		}
		if len(words2) > 0 {
			streetType := NormalizeStreetType(words2[len(words2)-1])
			if streetType != "" && streetType != words2[len(words2)-1] {
				result.Type2 = streetType
				words2 = words2[:len(words2)-1]
			}
		}
		if len(words2) > 0 {
			result.Street2 = strings.Join(words2, " ")
		}
	}

	// If both streets have the same type or one is missing, use the common type
	if result.Type1 == "" && result.Type2 != "" {
		result.Type1 = result.Type2
	} else if result.Type2 == "" && result.Type1 != "" {
		result.Type2 = result.Type1
	}

	return result
}
