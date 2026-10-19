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

func TestParseHandlerBrazilianCEPRequiresExplicitCountry(t *testing.T) {
	p := parser.NewParser()
	handler := parseHandler(p)

	parse := func(t *testing.T, address, country string) (*parseResponse, []byte) {
		t.Helper()
		request := map[string]string{
			"address": address,
			"type":    "auto",
		}
		if country != "" {
			request["country"] = country
		}
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}
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
		if !resp.Success || resp.Result == nil || resp.Result.Address == nil {
			t.Fatalf("unexpected parse response: %+v", resp)
		}
		return &resp, rec.Body.Bytes()
	}

	t.Run("Brazil CEP is returned as postal_code", func(t *testing.T) {
		resp, body := parse(t, "Av. Paulista, 1000, São Paulo, SP 01310-100", "BR")
		if got := resp.Result.Address.PostalCode; got != "01310-100" {
			t.Errorf("postal_code = %q, want %q", got, "01310-100")
		}
		if !bytes.Contains(body, []byte(`"postal_code":"01310-100"`)) {
			t.Errorf("response does not expose the postal_code field: %s", body)
		}
		if resp.Result.Address.ZIP != "" || resp.Result.Address.Plus4 != "" {
			t.Errorf("Brazilian CEP must not be exposed as a US ZIP: %+v", resp.Result.Address)
		}
	})

	for _, country := range []string{"", "US", "MX"} {
		t.Run("Brazil-shaped value with country="+country+" keeps legacy ZIP behavior", func(t *testing.T) {
			resp, body := parse(t, "Av. Paulista, 1000, São Paulo, SP 01310-100", country)
			if resp.Result.Address.PostalCode != "" {
				t.Errorf("postal_code = %q, want empty when country is %q", resp.Result.Address.PostalCode, country)
			}
			if got := resp.Result.Address.ZIP; got != "01310" {
				t.Errorf("ZIP = %q, want legacy five-digit match", got)
			}
			if bytes.Contains(body, []byte(`"postal_code"`)) {
				t.Errorf("legacy response unexpectedly contains postal_code: %s", body)
			}
		})
	}

	t.Run("malformed Brazilian CEP is not emitted", func(t *testing.T) {
		resp, _ := parse(t, "Av. Paulista, 1000, São Paulo, SP 01310-10", "BR")
		if resp.Result.Address.PostalCode != "" {
			t.Errorf("postal_code = %q, want empty for malformed CEP", resp.Result.Address.PostalCode)
		}
	})

	for _, address := range []struct {
		name  string
		value string
	}{
		{"reference number", "Reference: ticket 12345-678"},
		{"date and reference", "Date 2026-10-19, ref 12345-678"},
		{"comma-separated phone-like text", "Reference, date, phone 12345-678"},
		{"address followed by a second phone-like code", "Av. Paulista, 1000, São Paulo, SP 01310-100; phone 12345-678"},
	} {
		t.Run("unrelated numeric text is not a CEP: "+address.name, func(t *testing.T) {
			resp, _ := parse(t, address.value, "BR")
			if resp.Result.Address.PostalCode != "" {
				t.Errorf("postal_code = %q, want empty for unrelated or ambiguous numeric text", resp.Result.Address.PostalCode)
			}
		})
	}

	t.Run("US ZIP plus four remains unchanged", func(t *testing.T) {
		resp, body := parse(t, "1005 N Gravenstein Highway, Sebastopol, CA 95472-1234", "US")
		if got := resp.Result.Address.ZIP; got != "95472" {
			t.Errorf("ZIP = %q, want %q", got, "95472")
		}
		if got := resp.Result.Address.Plus4; got != "1234" {
			t.Errorf("plus4 = %q, want %q", got, "1234")
		}
		if resp.Result.Address.PostalCode != "" {
			t.Errorf("postal_code = %q, want empty for US ZIP+4", resp.Result.Address.PostalCode)
		}
		if bytes.Contains(body, []byte(`"postal_code"`)) {
			t.Errorf("US response unexpectedly contains postal_code: %s", body)
		}
	})
}
