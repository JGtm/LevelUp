package xblauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPollXboxDeviceCode_CorpsTronqueRendLErreurDeLecture : un corps de polling coupé avant la
// longueur annoncée (Content-Length menteur) rend l'erreur de LECTURE, pas l'erreur JSON du
// fragment reçu qui la masquerait.
func TestPollXboxDeviceCode_CorpsTronqueRendLErreurDeLecture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte(`{"error":`))
	}))
	defer srv.Close()

	_, _, err := FluxAppareil{Doer: srv.Client(), URLToken: srv.URL}.Poll(context.Background(), "client-id", "device-code", 0)
	if err == nil {
		t.Fatal("attendu une erreur sur un corps tronqué")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("erreur %q : attendu l'erreur de lecture du corps (io.ErrUnexpectedEOF)", err)
	}
	if strings.Contains(err.Error(), "JSON") {
		t.Errorf("erreur %q : l'erreur JSON masque l'erreur de lecture", err)
	}
}
