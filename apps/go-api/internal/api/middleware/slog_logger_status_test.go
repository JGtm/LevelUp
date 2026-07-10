// Package middleware — slog_logger_status_test.go : compteurs HTTP par classe
// de statut (plan monitoring A7, DC-6).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/observability"
)

func TestSlogLogger_CountsStatusClasses(t *testing.T) {
	// Compteurs relatifs (l'expvar est process-global — d'autres tests du
	// package peuvent l'avoir incrémenté).
	before2xx := observability.LoadCounter("http_status_2xx_total")
	before4xx := observability.LoadCounter("http_status_4xx_total")
	before5xx := observability.LoadCounter("http_status_5xx_total")
	before3xx := observability.LoadCounter("http_status_3xx_total")

	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/moved", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ok", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/notfound", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	h := SlogLogger(mux)

	for _, path := range []string{"/ok", "/ok", "/moved", "/notfound", "/boom"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	}

	if got := observability.LoadCounter("http_status_2xx_total") - before2xx; got != 2 {
		t.Errorf("2xx = %d, attendu 2", got)
	}
	if got := observability.LoadCounter("http_status_3xx_total") - before3xx; got != 1 {
		t.Errorf("3xx = %d, attendu 1", got)
	}
	if got := observability.LoadCounter("http_status_4xx_total") - before4xx; got != 1 {
		t.Errorf("4xx = %d, attendu 1", got)
	}
	if got := observability.LoadCounter("http_status_5xx_total") - before5xx; got != 1 {
		t.Errorf("5xx = %d, attendu 1", got)
	}
}
