// Package handlers_test — media_serve_test.go : tests pour ServeMediaFile
// couvrant la résolution multi-extension (fallback par stem) et le remux WebM
// à la volée pour les containers non-web-natifs (.mkv).
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// writeSettingsJSON crée un fichier app_settings.json minimal pour pointer
// MediaCapturesBaseDir vers le tempdir du test.
func writeSettingsJSON(t *testing.T, capturesBase string) *settings.Store {
	t.Helper()
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "app_settings.json")
	cfg := map[string]any{
		"media_captures_base_dir": capturesBase,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return settings.NewStore(settingsPath)
}

// newServeMediaRouter crée un router chi avec le handler ServeMediaFile branché.
// Les autres factories ne sont pas requises pour cet endpoint.
func newServeMediaRouter(t *testing.T, store *settings.Store) *chi.Mux {
	t.Helper()
	factory := func(_ context.Context, _ string) (port.MediaService, error) {
		return nil, nil
	}
	h := handlers.NewMediaHandler(factory, nil, "").WithSettingsStore(store)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/media/files/*", h.ServeMediaFile)
	})
	return r
}

// ─────────────────────────────────────────────────────────────────────────────

func TestServeMediaFile_ExactMatchMP4(t *testing.T) {
	capturesBase := t.TempDir()
	playerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "clip.mp4"), []byte("fakebytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/clip.mp4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Errorf("Content-Type = %q, want video/mp4", ct)
	}
}

// TestServeMediaFile_FallbackStemMP4OnDisk : URL demande clip.mkv (absent sur
// disque) ; on a un clip.mp4 sur disque. La fallback resolution doit le trouver
// et le servir directement (web-native, pas de remux). Pas besoin de ffmpeg.
func TestServeMediaFile_FallbackStemMP4OnDisk(t *testing.T) {
	capturesBase := t.TempDir()
	playerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "clip.mp4"), []byte("fakebytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	// URL pointe vers .mkv mais seul .mp4 existe.
	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/clip.mkv", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// L'extension résolue est .mp4 → Content-Type doit refléter cela.
	if ct := w.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Errorf("Content-Type = %q, want video/mp4 (fichier réel résolu)", ct)
	}
	if w.Body.String() != "fakebytes" {
		t.Errorf("body = %q, want fakebytes", w.Body.String())
	}
}

func TestServeMediaFile_NotFound(t *testing.T) {
	capturesBase := t.TempDir()
	playerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/missing.mp4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServeMediaFile_PathTraversalIsSafe(t *testing.T) {
	// path.Clean nettoie les ".." avant le check. Selon que la séquence
	// survit (cas exotique) ou pas (cas standard), on récupère 400 ou 404 —
	// les deux refusent de servir le fichier. Le test garantit juste que le
	// fichier sensible n'est PAS servi (pas un 200).
	capturesBase := t.TempDir()
	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/../../../etc/passwd", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("path traversal acceptée (200) — leak potentiel")
	}
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("expected 400 ou 404, got %d", w.Code)
	}
}

// TestServeMediaFile_RemuxMKVtoWebM : URL demande clip.mp4 (absent) ; clip.mkv
// AV1+Opus est sur disque. Le serveur doit le résoudre, détecter le besoin de
// remux, et servir un flux WebM. Skipped si ffmpeg absent.
func TestServeMediaFile_RemuxMKVtoWebM(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — test remux ignoré")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — test remux ignoré")
	}

	capturesBase := t.TempDir()
	playerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mkvPath := filepath.Join(playerDir, "clip.mkv")
	// Génère un MKV AV1+Opus minimal (1s).
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=10:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libaom-av1", "-cpu-used", "8", "-b:v", "100k",
		"-c:a", "libopus", "-b:a", "32k",
		mkvPath,
	}
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Skipf("encodage MKV indisponible (libaom-av1 manquant ?): %v\n%s", err, out)
	}

	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	// URL demande .mp4, fichier réel est .mkv → fallback + remux.
	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/clip.mp4", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "video/webm" {
		t.Errorf("Content-Type = %q, want video/webm (remux)", ct)
	}
	body := w.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("body vide après remux")
	}
	// Signature EBML (0x1A45DFA3) — WebM et MKV partagent ce magic.
	if !(body[0] == 0x1A && body[1] == 0x45 && body[2] == 0xDF && body[3] == 0xA3) {
		t.Errorf("body ne commence pas par EBML magic: % x", body[:min(8, len(body))])
	}
}

// TestServeMediaFile_FallbackDataMediaOnInvalidBase couvre l'incident prod
// 2026-06-13 : un media_captures_base_dir invalide (chemin Windows recopié sur le
// VPS Linux, inexistant) rendait TOUS les médias en 404. Le durcissement ajoute
// {repoRoot}/data/media aux bases candidates de résolution : un fichier présent au
// layout réel (data/media/{owner}/...) doit donc être servi en 200 malgré la base
// configurée invalide.
func TestServeMediaFile_FallbackDataMediaOnInvalidBase(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "data", "media", "JGtm", "thumbs", "clip.webp")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("WEBPDATA"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Base configurée INVALIDE (n'existe pas) — simule le chemin Windows déployé.
	store := writeSettingsJSON(t, filepath.Join(root, "Z-inexistant", "Captures"))
	factory := func(_ context.Context, _ string) (port.MediaService, error) { return nil, nil }
	h := handlers.NewMediaHandler(factory, nil, root).WithSettingsStore(store) // repoRoot non vide
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/media/files/*", h.ServeMediaFile)
	})

	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/thumbs/clip.webp", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 via fallback data/media (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/webp" {
		t.Errorf("Content-Type = %q, want image/webp", ct)
	}
	if w.Body.String() != "WEBPDATA" {
		t.Errorf("body = %q, want WEBPDATA", w.Body.String())
	}
}

// TestServeMediaFile_RemuxIncompatibleCodec : un .mkv à vidéo H.264 (conteneur
// exigeant un remux, mais codec hors av1/vp8/vp9) doit répondre 415 grâce au
// pré-flight, AU LIEU du 200 corps vide de l'ancien flux. Skipped si ffmpeg absent.
func TestServeMediaFile_RemuxIncompatibleCodec(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — test remux ignoré")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — test remux ignoré")
	}
	capturesBase := t.TempDir()
	playerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mkvPath := filepath.Join(playerDir, "clip.mkv")
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=10:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", mkvPath,
	}
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Skipf("encodage MKV H.264 indisponible: %v\n%s", err, out)
	}
	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/clip.mkv", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415 (codec incompatible WebM)", w.Code)
	}
	if w.Header().Get("Content-Type") == "video/webm" {
		t.Errorf("Content-Type video/webm écrit malgré le rejet 415")
	}
}

// TestServeMediaFile_RemuxProbeFailure : un .mkv illisible par ffprobe (octets
// non-média) doit répondre 502 (pré-flight échoué), pas un 200 corps vide.
func TestServeMediaFile_RemuxProbeFailure(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — test remux ignoré")
	}
	capturesBase := t.TempDir()
	playerDir := filepath.Join(capturesBase, "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "clip.mkv"), []byte("not a real matroska file"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := writeSettingsJSON(t, capturesBase)
	r := newServeMediaRouter(t, store)

	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/media/files/JGtm/clip.mkv", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (pré-flight ffprobe échoué)", w.Code)
	}
}
