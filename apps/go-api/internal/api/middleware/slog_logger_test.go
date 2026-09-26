// Package middleware_test — slog_logger_test.go : tests du middleware SlogLogger.
package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/observability/timing"
)

func TestSlogLogger_PassesThrough(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	handler := middleware.SlogLogger(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("inner handler not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSlogLogger_CapturesStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := middleware.SlogLogger(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ─── Plan perf 2026-09-23, lot L1 : requêtes lentes et sections chronométrées ──────────

const envSlowRequestMS = "LEVELUP_SLOW_REQUEST_MS"

// captureLogs redirige slog.Default vers un tampon JSON au niveau donné (restauré en fin
// de test). Le middleware journalise de façon synchrone : le tampon est complet au retour
// de ServeHTTP.
func captureLogs(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// logLines rend les lignes brutes du tampon dont le message vaut msg, avec leur décodage.
func logLines(t *testing.T, buf *bytes.Buffer, msg string) ([]string, []map[string]any) {
	t.Helper()
	var raws []string
	var recs []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("ligne de log illisible %q : %v", line, err)
		}
		if rec["msg"] == msg {
			raws = append(raws, line)
			recs = append(recs, rec)
		}
	}
	return raws, recs
}

// logRecords décode les lignes du tampon dont le message vaut msg.
func logRecords(t *testing.T, buf *bytes.Buffer, msg string) []map[string]any {
	t.Helper()
	_, recs := logLines(t, buf, msg)
	return recs
}

// sleepingHandler répond `status` après `delay`.
func sleepingHandler(delay time.Duration, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(status)
	}
}

// sectionHandler chronomètre deux fois « load » (delay chacune) et une fois « build ».
func sectionHandler(t *testing.T, delay time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tm := timing.FromContext(r.Context())
		if tm == nil {
			t.Error("aucun *timing.Timings sur le contexte de la requête")
		}
		for i := 0; i < 2; i++ {
			func() {
				defer tm.Section("load")()
				time.Sleep(delay)
			}()
		}
		func() { defer tm.Section("build")() }()
		w.WriteHeader(http.StatusOK)
	}
}

func serveOnce(h http.Handler, path string) {
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, path, nil))
}

func TestSlogLogger_SlowSuccessLoggedAtInfo(t *testing.T) {
	t.Setenv(envSlowRequestMS, "1")
	buf := captureLogs(t, slog.LevelInfo)
	serveOnce(middleware.SlogLogger(sleepingHandler(5*time.Millisecond, http.StatusOK)), "/api/v1/lente")

	recs := logRecords(t, buf, "http")
	if len(recs) != 1 {
		t.Fatalf("%d ligne(s) http, attendu 1 : %s", len(recs), buf.String())
	}
	if recs[0]["level"] != "INFO" || recs[0]["slow"] != true || recs[0]["status"] != float64(http.StatusOK) {
		t.Fatalf("ligne http = %v, attendu INFO, slow=true, status 200", recs[0])
	}
}

func TestSlogLogger_FastSuccessStaysDebug(t *testing.T) {
	t.Setenv(envSlowRequestMS, "60000")
	buf := captureLogs(t, slog.LevelDebug)
	serveOnce(middleware.SlogLogger(sleepingHandler(0, http.StatusOK)), "/api/v1/rapide")

	recs := logRecords(t, buf, "http")
	if len(recs) != 1 {
		t.Fatalf("%d ligne(s) http, attendu 1 : %s", len(recs), buf.String())
	}
	if recs[0]["level"] != "DEBUG" {
		t.Errorf("niveau = %v, attendu DEBUG", recs[0]["level"])
	}
	if _, ok := recs[0]["slow"]; ok {
		t.Errorf("attribut slow présent sur une requête rapide : %v", recs[0])
	}
}

// Seuil non atteint, niveau INFO : aucune ligne — ni `http`, ni `http_timings` (le handler
// chronomètre pourtant des sections).
func TestSlogLogger_FastRequestSilentAtInfo(t *testing.T) {
	t.Setenv(envSlowRequestMS, "60000")
	buf := captureLogs(t, slog.LevelInfo)
	serveOnce(middleware.SlogLogger(sectionHandler(t, 0)), "/api/v1/rapide")

	if buf.Len() != 0 {
		t.Fatalf("requête rapide journalisée en INFO : %s", buf.String())
	}
}

// Les deux côtés du MÊME seuil.
func TestSlogLogger_ThresholdRespected(t *testing.T) {
	t.Setenv(envSlowRequestMS, "200")
	buf := captureLogs(t, slog.LevelDebug)
	var delay time.Duration
	h := middleware.SlogLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
	serveOnce(h, "/api/v1/sous-le-seuil")
	delay = 250 * time.Millisecond
	serveOnce(h, "/api/v1/au-dela-du-seuil")

	recs := logRecords(t, buf, "http")
	if len(recs) != 2 {
		t.Fatalf("%d ligne(s) http, attendu 2 : %s", len(recs), buf.String())
	}
	if recs[0]["level"] != "DEBUG" || recs[0]["slow"] != nil {
		t.Errorf("sous le seuil : %v, attendu DEBUG sans slow", recs[0])
	}
	if recs[1]["level"] != "INFO" || recs[1]["slow"] != true {
		t.Errorf("au-delà du seuil : %v, attendu INFO slow=true", recs[1])
	}
	if ms, _ := recs[1]["duration_ms"].(float64); ms < 250 {
		t.Errorf("duration_ms = %v, attendu >= 250", recs[1]["duration_ms"])
	}
}

// Le seuil est lu au montage : changer la variable ensuite ne change rien.
func TestSlogLogger_ThresholdReadOnceAtMount(t *testing.T) {
	t.Setenv(envSlowRequestMS, "60000")
	buf := captureLogs(t, slog.LevelDebug)
	h := middleware.SlogLogger(sleepingHandler(5*time.Millisecond, http.StatusOK))
	t.Setenv(envSlowRequestMS, "1")
	serveOnce(h, "/api/v1/apres-montage")

	recs := logRecords(t, buf, "http")
	if len(recs) != 1 || recs[0]["level"] != "DEBUG" {
		t.Fatalf("lignes http = %v, attendu une ligne DEBUG (seuil du montage : 60 s)", recs)
	}
}

// Un 4xx/5xx lent garde son niveau (jamais rétrogradé en INFO) et porte slow=true.
func TestSlogLogger_SlowErrorsKeepTheirLevel(t *testing.T) {
	t.Setenv(envSlowRequestMS, "1")
	for status, level := range map[int]string{http.StatusInternalServerError: "ERROR", http.StatusNotFound: "WARN"} {
		buf := captureLogs(t, slog.LevelDebug)
		serveOnce(middleware.SlogLogger(sleepingHandler(5*time.Millisecond, status)), "/api/v1/erreur")
		recs := logRecords(t, buf, "http")
		if len(recs) != 1 || recs[0]["level"] != level || recs[0]["slow"] != true {
			t.Errorf("status %d : lignes http = %v, attendu une ligne %s slow=true", status, recs, level)
		}
	}
}

// Valeur invalide : erreur journalisée au montage, seuil par défaut (1 s) appliqué.
func TestSlogLogger_InvalidThresholdFallsBackToDefault(t *testing.T) {
	for _, raw := range []string{"abc", "0", "-5"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv(envSlowRequestMS, raw)
			buf := captureLogs(t, slog.LevelDebug)
			h := middleware.SlogLogger(sleepingHandler(5*time.Millisecond, http.StatusOK))

			var mountErrors int
			for _, rec := range logRecords(t, buf, "http: seuil de requête lente invalide, défaut appliqué") {
				if rec["level"] == "ERROR" && rec["value"] == raw && rec["err"] != nil {
					mountErrors++
				}
			}
			if mountErrors != 1 {
				t.Fatalf("%d erreur(s) de montage pour %q, attendu 1 : %s", mountErrors, raw, buf.String())
			}
			serveOnce(h, "/api/v1/defaut")
			recs := logRecords(t, buf, "http")
			if len(recs) != 1 || recs[0]["level"] != "DEBUG" {
				t.Fatalf("lignes http = %v, attendu DEBUG (5 ms sous le défaut de 1 s)", recs)
			}
		})
	}
}

func TestSlogLogger_TimingsLoggedForSlowRequest(t *testing.T) {
	t.Setenv(envSlowRequestMS, "1")
	buf := captureLogs(t, slog.LevelInfo)
	serveOnce(middleware.SlogLogger(sectionHandler(t, 3*time.Millisecond)), "/api/v1/players/x/pages/teammates")

	raws, recs := logLines(t, buf, "http_timings")
	if len(recs) != 1 {
		t.Fatalf("%d ligne(s) http_timings, attendu 1 : %s", len(recs), buf.String())
	}
	rec := recs[0]
	if rec["level"] != "INFO" || rec["path"] != "/api/v1/players/x/pages/teammates" {
		t.Errorf("ligne http_timings = %v, attendu INFO sur le chemin de la requête", rec)
	}
	if ms, _ := rec["total_ms"].(float64); ms < 6 {
		t.Errorf("total_ms = %v, attendu >= 6 (deux « load » de 3 ms)", rec["total_ms"])
	}
	if _, ok := rec["duration_ms"].(float64); !ok {
		t.Errorf("duration_ms absent : %v", rec)
	}
	sections, _ := rec["sections"].(string)
	if !strings.HasPrefix(sections, "load=") || !strings.Contains(sections, " build=") {
		t.Errorf("sections = %q, attendu load puis build (plus longue d'abord)", sections)
	}
	if rec["calls"] != "load=2" {
		t.Errorf("calls = %v, attendu \"load=2\"", rec["calls"])
	}
	t.Logf("exemple de ligne http_timings : %s", raws[0])
}

func TestSlogLogger_NoTimingsLineWithoutSection(t *testing.T) {
	t.Setenv(envSlowRequestMS, "1")
	buf := captureLogs(t, slog.LevelDebug)
	serveOnce(middleware.SlogLogger(sleepingHandler(5*time.Millisecond, http.StatusOK)), "/api/v1/sans-section")

	if recs := logRecords(t, buf, "http_timings"); len(recs) != 0 {
		t.Fatalf("http_timings émis sans aucune section : %v", recs)
	}
	if recs := logRecords(t, buf, "http"); len(recs) != 1 || recs[0]["slow"] != true {
		t.Fatalf("lignes http = %v, attendu une ligne slow=true", recs)
	}
}

// Requête rapide : les sections ne sont détaillées que si le niveau DEBUG est actif.
func TestSlogLogger_TimingsForFastRequestOnlyWhenDebug(t *testing.T) {
	t.Setenv(envSlowRequestMS, "60000")
	for level, want := range map[slog.Level]int{slog.LevelInfo: 0, slog.LevelDebug: 1} {
		buf := captureLogs(t, level)
		serveOnce(middleware.SlogLogger(sectionHandler(t, 0)), "/api/v1/rapide")
		recs := logRecords(t, buf, "http_timings")
		if len(recs) != want {
			t.Errorf("niveau %v : %d ligne(s) http_timings, attendu %d", level, len(recs), want)
			continue
		}
		if want == 1 && recs[0]["level"] != "INFO" {
			t.Errorf("http_timings au niveau %v, attendu INFO", recs[0]["level"])
		}
	}
}
