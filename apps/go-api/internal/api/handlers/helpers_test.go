// Package handlers — helpers_test.go : tests boîte blanche pour writeJSON et writeError.
package handlers

import (
	"context"
	"encoding/json"
	"math"
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

// --- sanitizeFloatsForJSON : NaN/Inf neutralization ---
// Bug : un float64 NaN dans une struct passée par valeur à writeJSON faisait
// échouer json.Marshal et retournait 500 (vu en prod le 2026-05-26 sur le
// détail match Madina97294/615b3ebc...). La sanitize existait déjà mais
// n'agissait pas sur les champs de struct par valeur (CanSet()=false via
// reflect.ValueOf d'un interface non-pointeur).

type sanitizePayload struct {
	Score    float64            `json:"score"`
	Accuracy float64            `json:"accuracy"`
	KDA      *float64           `json:"kda,omitempty"`
	Axes     []float64          `json:"axes"`
	ByPlayer map[string]float64 `json:"by_player"`
	Nested   sanitizeNested     `json:"nested"`
}

type sanitizeNested struct {
	Ratio float64 `json:"ratio"`
}

func TestWriteJSON_NeutralizesNaNInStructByValue(t *testing.T) {
	w := httptest.NewRecorder()
	nan := math.NaN()
	payload := sanitizePayload{
		Score:    nan,
		Accuracy: math.Inf(1),
		KDA:      &nan,
		Axes:     []float64{1.0, nan, math.Inf(-1)},
		ByPlayer: map[string]float64{"alice": nan, "bob": 0.5},
		Nested:   sanitizeNested{Ratio: nan},
	}
	writeJSON(w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after sanitize, got %d (body=%s)", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("body must be valid JSON, got err=%v body=%s", err, w.Body.String())
	}
	if got["score"].(float64) != 0 {
		t.Errorf("Score NaN should become 0, got %v", got["score"])
	}
	if got["accuracy"].(float64) != 0 {
		t.Errorf("Accuracy +Inf should become 0, got %v", got["accuracy"])
	}
	if _, present := got["kda"]; present {
		t.Errorf("*float64 NaN with omitempty should disappear, got %v", got["kda"])
	}
	axes := got["axes"].([]interface{})
	if axes[1].(float64) != 0 || axes[2].(float64) != 0 {
		t.Errorf("Slice NaN/Inf elements should become 0, got %v", axes)
	}
	byPlayer := got["by_player"].(map[string]interface{})
	if byPlayer["alice"].(float64) != 0 {
		t.Errorf("Map NaN value should become 0, got %v", byPlayer["alice"])
	}
	if byPlayer["bob"].(float64) != 0.5 {
		t.Errorf("Map non-NaN value should be preserved, got %v", byPlayer["bob"])
	}
	nested := got["nested"].(map[string]interface{})
	if nested["ratio"].(float64) != 0 {
		t.Errorf("Nested struct NaN should become 0, got %v", nested["ratio"])
	}
}

func TestWriteJSON_PreservesValidFloats(t *testing.T) {
	w := httptest.NewRecorder()
	payload := sanitizePayload{
		Score:    1.23,
		Accuracy: 0.456,
		Axes:     []float64{1, 2, 3},
		ByPlayer: map[string]float64{"a": 0.9},
		Nested:   sanitizeNested{Ratio: 0.5},
	}
	writeJSON(w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got sanitizePayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Score != 1.23 || got.Accuracy != 0.456 || got.Nested.Ratio != 0.5 {
		t.Errorf("valid floats altered: %+v", got)
	}
}

func TestSanitizeFloatsForJSON_ReportsPaths(t *testing.T) {
	nan := math.NaN()
	payload := sanitizePayload{
		Score:    nan,
		Accuracy: math.Inf(1),
		KDA:      &nan,
		Axes:     []float64{1.0, nan},
		ByPlayer: map[string]float64{"alice": nan},
		Nested:   sanitizeNested{Ratio: nan},
	}
	_, paths := sanitizeFloatsForJSON(payload)
	if len(paths) == 0 {
		t.Fatalf("expected paths to be reported, got none")
	}
	// Each NaN/Inf source should appear (order not guaranteed, no duplicate check).
	expectedSubstrings := []string{".Score", ".Accuracy", ".KDA", ".Axes[1]", ".ByPlayer[alice]", ".Nested.Ratio"}
	joined := ""
	for _, p := range paths {
		joined += p + "|"
	}
	for _, want := range expectedSubstrings {
		if !containsSubstr(joined, want) {
			t.Errorf("expected path containing %q, got: %v", want, paths)
		}
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSanitizeFloatsForJSON_NilSafe(t *testing.T) {
	out, paths := sanitizeFloatsForJSON(nil)
	if out != nil || paths != nil {
		t.Fatalf("nil input must return (nil, nil), got (%v, %v)", out, paths)
	}
}

func TestWriteJSON_PointerArgumentStillWorks(t *testing.T) {
	w := httptest.NewRecorder()
	nan := math.NaN()
	payload := &sanitizePayload{Score: nan, Nested: sanitizeNested{Ratio: nan}}
	writeJSON(w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Pour un pointeur, l'original est modifié en place.
	if !math.IsNaN(payload.Score) {
		// après sanitize, NaN doit être 0 (modif en place via pointeur).
		if payload.Score != 0 {
			t.Errorf("pointer payload Score should be 0 after in-place sanitize, got %v", payload.Score)
		}
	}
}
