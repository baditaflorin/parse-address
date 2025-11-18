package parser

import (
	"strings"
)

// ParsedAddress represents a fully parsed street address
type ParsedAddress struct {
	Number      string `json:"number,omitempty"`
	Prefix      string `json:"prefix,omitempty"`
	Street      string `json:"street,omitempty"`
	Type        string `json:"type,omitempty"`
	Suffix      string `json:"suffix,omitempty"`
	SecUnitType string `json:"sec_unit_type,omitempty"`
	SecUnitNum  string `json:"sec_unit_num,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	ZIP         string `json:"zip,omitempty"`
	Plus4       string `json:"plus4,omitempty"`
}

// ParsedIntersection represents a street intersection
type ParsedIntersection struct {
	Prefix1 string `json:"prefix1,omitempty"`
	Street1 string `json:"street1,omitempty"`
	Type1   string `json:"type1,omitempty"`
	Suffix1 string `json:"suffix1,omitempty"`
	Prefix2 string `json:"prefix2,omitempty"`
	Street2 string `json:"street2,omitempty"`
	Type2   string `json:"type2,omitempty"`
	Suffix2 string `json:"suffix2,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	ZIP     string `json:"zip,omitempty"`
}

// ParseResult is a union type that can hold different parse results
type ParseResult struct {
	Type         string              `json:"type"` // "address", "intersection", "po_box", "none"
	Address      *ParsedAddress      `json:"address,omitempty"`
	Intersection *ParsedIntersection `json:"intersection,omitempty"`
}

// IsEmpty checks if all fields of ParsedAddress are empty
func (p *ParsedAddress) IsEmpty() bool {
	return p.Number == "" &&
		p.Prefix == "" &&
		p.Street == "" &&
		p.Type == "" &&
		p.Suffix == "" &&
		p.SecUnitType == "" &&
		p.SecUnitNum == "" &&
		p.City == "" &&
		p.State == "" &&
		p.ZIP == "" &&
		p.Plus4 == ""
}

// Normalize applies title casing and trimming to address fields
func (p *ParsedAddress) Normalize() {
	p.Number = strings.TrimSpace(p.Number)
	p.Prefix = strings.TrimSpace(p.Prefix)
	p.Street = titleCase(p.Street)
	p.Type = strings.TrimSpace(p.Type)
	p.Suffix = strings.TrimSpace(p.Suffix)
	p.SecUnitType = strings.TrimSpace(p.SecUnitType)
	p.SecUnitNum = strings.TrimSpace(p.SecUnitNum)
	p.City = titleCase(p.City)
	p.State = strings.ToUpper(strings.TrimSpace(p.State))
	p.ZIP = strings.TrimSpace(p.ZIP)
	p.Plus4 = strings.TrimSpace(p.Plus4)
}

// titleCase converts a string to title case
func titleCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
