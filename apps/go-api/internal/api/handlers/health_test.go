// Package handlers_test — health_test.go : tests HealthHandler.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
)

// mockBootstrapRepo implémente port.BootstrapRepository pour les tests.
type mockBootstrapRepo struct {
	matchCount  int
	matchErr    error
	dbVersion   string
	playerCount int
	lastSync    *time.Time
}

func (m *mockBootstrapRepo) GetMatchCount(_ context.Context) (int, error) {
	return m.matchCount, m.matchErr
}
func (m *mockBootstrapRepo) GetDBVersion(_ context.Context) (string, error) {
	return m.dbVersion, nil
}
func (m *mockBootstrapRepo) GetPlayerCount(_ context.Context) (int, error) {
	return m.playerCount, nil
}
func (m *mockBootstrapRepo) GetLastSyncAt(_ context.Context) (*time.Time, error) {
	return m.lastSync, nil
}

func TestHealthHandler_OK(t *testing.T) {
	now := time.Now()
	repo := &mockBootstrapRepo{
		matchCount:  1500,
		dbVersion:   "v1.4.4",
		playerCount: 3,
		lastSync:    &now,
	}
	h := handlers.NewHealthHandlerWithVersion(repo, "1.0.0-test").
		WithMediaTooling(domain.MediaToolingStatus{FFmpeg: true, FFprobe: true, FFmpegVersion: "ffmpeg version 5.1.6"})
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
	if int(resp["match_count"].(float64)) != 1500 {
		t.Errorf("expected match_count=1500, got %v", resp["match_count"])
	}
	if resp["app_version"] != "1.0.0-test" {
		t.Errorf("expected app_version=1.0.0-test, got %v", resp["app_version"])
	}
	// Preuve positive de l'outillage média exposée dans /health.
	mt, ok := resp["media_tooling"].(map[string]interface{})
	if !ok {
		t.Fatalf("champ media_tooling manquant ou mauvais type: %v", resp["media_tooling"])
	}
	if mt["ffmpeg"] != true {
		t.Errorf("attendu media_tooling.ffmpeg=true, obtenu %v", mt["ffmpeg"])
	}
	if mt["ffprobe"] != true {
		t.Errorf("attendu media_tooling.ffprobe=true, obtenu %v", mt["ffprobe"])
	}
	if mt["ffmpeg_version"] != "ffmpeg version 5.1.6" {
		t.Errorf("attendu media_tooling.ffmpeg_version renseigné, obtenu %v", mt["ffmpeg_version"])
	}
}

func TestHealthHandler_MediaTooling_DefaultsFalse(t *testing.T) {
	// Sans WithMediaTooling, le champ est présent avec ffmpeg/ffprobe=false
	// (zéro-valeur) — jamais absent, pour rester une sonde fiable.
	repo := &mockBootstrapRepo{matchCount: 1, dbVersion: "v1"}
	h := handlers.NewHealthHandler(repo)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	mt, ok := resp["media_tooling"].(map[string]interface{})
	if !ok {
		t.Fatalf("champ media_tooling doit toujours être présent, obtenu %v", resp["media_tooling"])
	}
	if mt["ffmpeg"] != false || mt["ffprobe"] != false {
		t.Errorf("défaut attendu ffmpeg=false ffprobe=false, obtenu %v", mt)
	}
}

func TestHealthHandler_DBUnavailable(t *testing.T) {
	repo := &mockBootstrapRepo{matchErr: errors.New("connection refused")}
	h := handlers.NewHealthHandler(repo)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHealthHandler_NoVersion(t *testing.T) {
	repo := &mockBootstrapRepo{matchCount: 100, dbVersion: "v1.4.4"}
	h := handlers.NewHealthHandler(repo)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	// app_version doit être vide (pas de version fournie)
	if v, ok := resp["app_version"]; ok && v != "" {
		t.Errorf("expected empty app_version, got %v", v)
	}
}

// ─── P8.11 (revue 2026-04-29) — /healthz et /readyz séparés ────────────────

func TestLiveness_AlwaysOK_NoDBQuery(t *testing.T) {
	// Liveness doit retourner 200 même si la DB est en panne.
	repo := &mockBootstrapRepo{matchErr: errors.New("connection refused")}
	h := handlers.NewHealthHandler(repo)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "alive" {
		t.Errorf("expected status=alive, got %v", resp["status"])
	}
}

func TestReadiness_OK_WhenDBHealthy(t *testing.T) {
	repo := &mockBootstrapRepo{matchCount: 1500, dbVersion: "v1.4.4"}
	h := handlers.NewHealthHandler(repo)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ready" {
		t.Errorf("expected status=ready, got %v", resp["status"])
	}
	checks, ok := resp["checks"].(map[string]any)
	if !ok {
		t.Fatal("checks field manquant ou mauvais type")
	}
	if checks["duckdb"] != "ok" {
		t.Errorf("expected checks.duckdb=ok, got %v", checks["duckdb"])
	}
}

func TestReadiness_503_WhenDBDown(t *testing.T) {
	repo := &mockBootstrapRepo{matchErr: errors.New("connection refused")}
	h := handlers.NewHealthHandler(repo)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "not_ready" {
		t.Errorf("expected status=not_ready, got %v", resp["status"])
	}
	checks, _ := resp["checks"].(map[string]any)
	dbCheck, _ := checks["duckdb"].(string)
	if dbCheck == "" || dbCheck == "ok" {
		t.Errorf("expected checks.duckdb=err message, got %q", dbCheck)
	}
}
