// Package handlers_test — build_worker_client_ip_log_test.go : le refus d'un jeton
// d'ouvrier journalise `ip` = IP cliente de la pile (X-Real-IP derrière le proxy),
// jamais le pair TCP du proxy ni un en-tête forgé.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
)

func TestClientIPLogs_WorkerTokenRejectionCarriesClientIP(t *testing.T) {
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	worker := handlers.NewBuildWorkerHandler("jeton-serveur", nil, nil, nil) // pragma: allowlist secret
	guard := worker.RequireWorkerToken(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h := chimiddleware.ClientIPFromRemoteAddr(chimiddleware.ClientIPFromHeader("X-Real-IP")(guard))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/build-queue/claim", nil)
	req.RemoteAddr = "127.0.0.1:40000"
	req.Header.Set("X-Real-IP", "203.0.113.95")
	req.Header.Set("True-Client-IP", "198.51.100.96")
	req.Header.Set("Authorization", "Bearer mauvais-jeton")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("jeton faux : status %d, want 401", rec.Code)
	}

	found := false
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("ligne de journal illisible %q : %v", line, err)
		}
		if rec["msg"] != "build_queue: jeton d'ouvrier refusé" {
			continue
		}
		found = true
		if rec["ip"] != "203.0.113.95" {
			t.Errorf("ip = %v, want 203.0.113.95 (record %v)", rec["ip"], rec)
		}
	}
	if !found {
		t.Fatalf("journal de refus absent : %s", buf.String())
	}
	if strings.Contains(buf.String(), "198.51.100.96") {
		t.Errorf("l'IP forgée apparaît dans les journaux : %s", buf.String())
	}
}
