// Package humacore — tests du socle de migration Huma (Phase 3b) :
// sanitisation NaN/Inf, modèle d'erreur (contrat writeError) et garanties du
// format JSON byte-identique à writeJSON.
package humacore

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"testing"
)

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
	_, paths := SanitizeFloatsForJSON(payload)
	if len(paths) == 0 {
		t.Fatalf("expected paths to be reported, got none")
	}
	joined := strings.Join(paths, "|")
	for _, want := range []string{".Score", ".Accuracy", ".KDA", ".Axes[1]", ".ByPlayer[alice]", ".Nested.Ratio"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected path containing %q, got: %v", want, paths)
		}
	}
}

func TestSanitizeFloatsForJSON_NilSafe(t *testing.T) {
	out, paths := SanitizeFloatsForJSON(nil)
	if out != nil || paths != nil {
		t.Fatalf("nil input must return (nil, nil), got (%v, %v)", out, paths)
	}
}

// TestJSONFormat_NaNSafeAndTrailingNewline : le format Huma produit un corps
// byte-identique à writeJSON — NaN neutralisé (json.Marshal n'échoue pas) et
// trailing "\n".
func TestJSONFormat_NaNSafeAndTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	if err := JSONFormat.Marshal(&buf, sanitizePayload{Score: math.NaN(), Axes: []float64{1.5}}); err != nil {
		t.Fatalf("Marshal NaN ne doit pas échouer (sanitize en amont): %v", err)
	}
	body := buf.Bytes()
	if len(body) == 0 || body[len(body)-1] != '\n' {
		t.Errorf("corps doit finir par un trailing newline, got %q", string(body))
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("corps JSON invalide: %v", err)
	}
	if got["score"].(float64) != 0 {
		t.Errorf("NaN doit devenir 0, got %v", got["score"])
	}
}

// TestNewError_GenericOn5xx : message générique « internal error » + retryable
// true sur 5xx (pas de fuite d'info interne) ; message exact + retryable false
// sur 4xx.
func TestNewError_GenericOn5xx(t *testing.T) {
	e5 := NewError(http.StatusInternalServerError, "boom_code", "détail interne sensible").(interface {
		Error() string
		GetStatus() int
	})
	if e5.GetStatus() != 500 {
		t.Errorf("status = %d, want 500", e5.GetStatus())
	}
	if e5.Error() != "internal error" {
		t.Errorf("5xx message = %q, want 'internal error'", e5.Error())
	}
	b5, _ := json.Marshal(NewError(http.StatusInternalServerError, "boom_code", "x"))
	var m5 map[string]any
	_ = json.Unmarshal(b5, &m5)
	if m5["code"] != "boom_code" || m5["retryable"] != true || m5["message"] != "internal error" {
		t.Errorf("5xx body = %v", m5)
	}

	b4, _ := json.Marshal(NewError(http.StatusNotFound, "not_here", "introuvable"))
	var m4 map[string]any
	_ = json.Unmarshal(b4, &m4)
	if m4["code"] != "not_here" || m4["retryable"] != false || m4["message"] != "introuvable" {
		t.Errorf("4xx body = %v", m4)
	}
}

func TestErrorCodeForStatus(t *testing.T) {
	cases := map[int]string{
		http.StatusBadRequest:          "bad_request",
		http.StatusUnauthorized:        "unauthorized",
		http.StatusForbidden:           "forbidden",
		http.StatusNotFound:            "not_found",
		http.StatusUnprocessableEntity: "validation_error",
		http.StatusBadGateway:          "internal_error",
		http.StatusTeapot:              "error",
	}
	for status, want := range cases {
		if got := ErrorCodeForStatus(status); got != want {
			t.Errorf("ErrorCodeForStatus(%d) = %q, want %q", status, got, want)
		}
	}
}
