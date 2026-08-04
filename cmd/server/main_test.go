package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/parse-address/pkg/parser"
)

// TestParseHandlerValidatesNonAutoTypes guards against a regression where
// only the default/"auto" parse type routed through
// parser.ValidateAndSanitize (via ParseLocation); the "standard",
// "informal", "intersection", and "po_box" types called the parser
// methods directly on the raw request body, silently bypassing the
// UTF-8 validation, null-byte rejection, and MaxAddressLength (DoS)
// protections the README documents as blanket security properties of
// this service for 4 of its 5 API request types.
func TestParseHandlerValidatesNonAutoTypes(t *testing.T) {
	p := parser.NewParser()
	handler := parseHandler(p)

	oversized := strings.Repeat("A", parser.MaxInputLength+1)

	tests := []struct {
		name    string
		reqType string
		address string
	}{
		{"standard rejects null byte", "standard", "123 Main\x00St"},
		{"standard rejects oversized input", "standard", oversized},
		{"informal rejects null byte", "informal", "123 Main\x00St"},
		{"intersection rejects null byte", "intersection", "Main St\x00 and Elm Ave"},
		{"po_box rejects null byte", "po_box", "PO Box\x00 123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{
				"address": tt.address,
				"type":    tt.reqType,
			})
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/parse", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}

			var resp parseResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if resp.Success {
				t.Errorf("expected success=false for invalid input, got success=true")
			}
		})
	}
}

// TestParseHandlerAcceptsValidNonAutoTypes is a sanity check that the
// validation added above doesn't reject legitimate input on the
// non-"auto" code paths.
func TestParseHandlerAcceptsValidNonAutoTypes(t *testing.T) {
	p := parser.NewParser()
	handler := parseHandler(p)

	body, _ := json.Marshal(map[string]string{
		"address": "123 Main St Apt 4B San Francisco CA 94105",
		"type":    "standard",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp parseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got error: %s", resp.Error)
	}
	if resp.Result == nil || resp.Result.Address == nil || resp.Result.Address.City != "San Francisco" {
		t.Errorf("unexpected parse result: %+v", resp.Result)
	}
}
