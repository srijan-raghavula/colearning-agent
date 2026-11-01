package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

func TestDecodeJSONEnforcesRequestContract(t *testing.T) {
	type requestBody struct {
		Message string `json:"message"`
	}

	tests := []struct {
		name        string
		contentType string
		body        []byte
		wantStatus  int
		wantDecoded bool
	}{
		{name: "valid JSON with media type parameters", contentType: "application/json; charset=utf-8", body: []byte(`{"message":"hello"}`), wantStatus: http.StatusOK, wantDecoded: true},
		{name: "missing content type", body: []byte(`{"message":"hello"}`), wantStatus: http.StatusUnsupportedMediaType},
		{name: "wrong content type", contentType: "text/plain", body: []byte(`{"message":"hello"}`), wantStatus: http.StatusUnsupportedMediaType},
		{name: "malformed JSON", contentType: "application/json", body: []byte(`{`), wantStatus: http.StatusBadRequest},
		{name: "unknown field", contentType: "application/json", body: []byte(`{"message":"hello","extra":true}`), wantStatus: http.StatusBadRequest},
		{name: "multiple JSON values", contentType: "application/json", body: []byte(`{"message":"hello"} {}`), wantStatus: http.StatusBadRequest},
		{name: "empty body", contentType: "application/json", body: []byte(" \n "), wantStatus: http.StatusBadRequest},
		{name: "null body", contentType: "application/json", body: []byte(`null`), wantStatus: http.StatusBadRequest},
		{name: "body exceeds limit", contentType: "application/json", body: bytes.Repeat([]byte(" "), int(maxJSONBodyBytes+1)), wantStatus: http.StatusRequestEntityTooLarge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()
			var decoded requestBody
			gotDecoded := decodeJSON(response, request, &decoded)

			if gotDecoded != test.wantDecoded {
				t.Fatalf("decodeJSON() = %v, want %v", gotDecoded, test.wantDecoded)
			}
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantDecoded {
				if decoded.Message != "hello" {
					t.Fatalf("decoded message = %q, want hello", decoded.Message)
				}
				return
			}

			if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("error content type = %q", got)
			}
			var envelope map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope["error"] == "" {
				t.Fatalf("error envelope has no message: %v", envelope)
			}
		})
	}
}

func TestAPITurnSerializesEmptySourcesAsArray(t *testing.T) {
	encoded, err := json.Marshal(toAPITurn(domain.LearningTurn{}))
	if err != nil {
		t.Fatalf("marshal API turn: %v", err)
	}
	var turn map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &turn); err != nil {
		t.Fatalf("decode API turn: %v", err)
	}
	sources, found := turn["sources"]
	if !found || string(sources) != "[]" {
		t.Fatalf("sources = %s, want []", sources)
	}
}
