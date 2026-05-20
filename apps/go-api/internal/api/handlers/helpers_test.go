// Package handlers — helpers_test.go : tests boîte blanche pour writeJSON et writeError.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON_SetsContentTypeAndStatus(t *testing.T) {
	w := httptest.NewRecorder()
	type payload struct {
		Value string `json:"value"`
	}
	writeJSON(w, http.StatusCreated, payload{Value: "test"})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var resp payload
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode body: %v", err)
	}
	if resp.Value != "test" {
		t.Fatalf("expected value 'test', got %q", resp.Value)
	}
}

func TestWriteError_SetsErrorBody(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(context.Background(), w, http.StatusBadRequest, "invalid_input", "Le champ X est requis.")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode body: %v", err)
	}
	if resp["code"] != "invalid_input" {
		t.Fatalf("expected error code 'invalid_input', got %v", resp["code"])
	}
	if resp["message"] != "Le champ X est requis." {
		t.Fatalf("expected message, got %v", resp["message"])
	}
}
