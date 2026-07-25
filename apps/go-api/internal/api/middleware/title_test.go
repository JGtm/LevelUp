// Package middleware_test — title_test.go : tests du middleware TitleExtractor.
package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

func TestTitleExtractor_FromHeader(t *testing.T) {
	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "halo_infinite")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != "halo_infinite" {
		t.Errorf("expected halo_infinite from header, got %q", captured)
	}
}

func TestTitleExtractor_InvalidHeader_Fallback(t *testing.T) {
	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "nonexistent_game")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != titlePkg.DefaultSlug {
		t.Errorf("expected fallback %q for invalid header, got %q", titlePkg.DefaultSlug, captured)
	}
}

func TestTitleExtractor_NoHeader_DefaultFallback(t *testing.T) {
	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != titlePkg.DefaultSlug {
		t.Errorf("expected default slug %q, got %q", titlePkg.DefaultSlug, captured)
	}
}

// MT-22 (PMT-8) : le SEAM résout n'importe quel titre CONNU, y compris
// coming_soon — c'est le gate RequireActiveTitle (et non le seam) qui rejette
// un titre non-actif en 503. Ici on prouve que le titre coming_soon demandé est
// bien injecté dans le contexte (pas masqué par un fallback silencieux).
func TestTitleExtractor_ComingSoonHeader_Resolved(t *testing.T) {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug: "futur_titre", Name: "Futur", Status: titlePkg.StatusComingSoon,
	})
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "futur_titre")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != "futur_titre" {
		t.Errorf("le seam doit résoudre le titre coming_soon connu, attendu %q, got %q",
			"futur_titre", captured)
	}
}

// Anti-fuite cross-titre V72-29 : le serveur ÉCHOIE le titre résolu dans le header
// de réponse X-LevelUp-Title-Resolved. Le client API compare cet écho au titre qu'il
// considère actif et rejette toute réponse divergente → aucune réponse d'un autre
// titre ne peut polluer le cache, quelle que soit la course de bascule. On prouve ici
// que l'écho reflète bien la résolution (header ici) ET qu'il est posé même si le
// handler n'écrit rien explicitement.
func TestTitleExtractor_EchoesResolvedTitleHeader(t *testing.T) {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: titlePkg.StatusActive,
	})
	mw := middleware.TitleExtractor(registry)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"header connu échoié tel quel", "halo_5", "halo_5"},
		{"header inconnu → défaut échoié", "nonexistent_game", titlePkg.DefaultSlug},
		{"aucun header → défaut échoié", "", titlePkg.DefaultSlug},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if c.header != "" {
			req.Header.Set("X-LevelUp-Title", c.header)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if got := w.Header().Get(middleware.ResolvedTitleHeader); got != c.want {
			t.Errorf("%s : écho %s=%q, want %q", c.name, middleware.ResolvedTitleHeader, got, c.want)
		}
	}
}

// La locale UI est extraite du header X-LevelUp-Locale et placée dans le contexte
// (ctxkeys.Locale) — alimente les lectures localisées (noms de commendations H5).
func TestTitleExtractor_LocaleFromHeader(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"en", "en"},
		{"en-US", "en"},
		{"fr", "fr"},
		{"fr-FR", "fr"},
		{"", "fr"},   // absent → défaut fr
		{"de", "fr"}, // inconnue → défaut fr
	}
	mw := middleware.TitleExtractor(titlePkg.NewRegistry())
	for _, c := range cases {
		var captured string
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured = ctxkeys.Locale(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if c.header != "" {
			req.Header.Set("X-LevelUp-Locale", c.header)
		}
		handler.ServeHTTP(httptest.NewRecorder(), req)
		if captured != c.want {
			t.Errorf("X-LevelUp-Locale=%q → locale %q, want %q", c.header, captured, c.want)
		}
	}
}

// Header présent mais titre INCONNU : le fallback ne doit pas être silencieux (anti-fuite,
// CLAUDE.md anti-pattern #10). On capture le logger par défaut et on prouve qu'un WARN
// nommant le titre demandé est émis — la mauvaise résolution devient visible dans logs/.
func TestTitleExtractor_UnknownHeader_LogsWarn(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))

	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "nonexistent_game")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if captured != titlePkg.DefaultSlug {
		t.Errorf("titre inconnu doit retomber sur le défaut %q, got %q", titlePkg.DefaultSlug, captured)
	}
	logged := buf.String()
	if !strings.Contains(logged, "level=WARN") || !strings.Contains(logged, "nonexistent_game") {
		t.Errorf("un WARN nommant le titre demandé était attendu, log=%q", logged)
	}
}

// Cœur de l'anti-fuite backend : un header explicite l'emporte sur une session pointant un
// AUTRE titre. C'est ce qui rend le correctif front (toujours envoyer le header) efficace —
// même si la session vaut encore halo_5, halo_infinite affirmé par le header gagne, donc
// aucune donnée H5 ne peut être servie sous la clé Infinite.
func TestTitleExtractor_HeaderBeatsSession(t *testing.T) {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: titlePkg.StatusActive,
	})
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "halo_infinite")
	ctx := middleware.InjectSession(req.Context(), &domain.SessionData{CurrentTitleSlug: "halo_5"})
	handler.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))

	if captured != "halo_infinite" {
		t.Errorf("le header doit primer sur la session, attendu halo_infinite, got %q", captured)
	}
}

// Sans header, la session fait autorité (reprise du dernier titre). Prouve que le chemin de
// résolution par session — le fallback sur lequel reposait la fuite quand le front n'envoyait
// pas de header pour le titre par défaut — fonctionne correctement.
func TestTitleExtractor_SessionFallback_NoHeader(t *testing.T) {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: titlePkg.StatusActive,
	})
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := middleware.InjectSession(req.Context(), &domain.SessionData{CurrentTitleSlug: "halo_5"})
	handler.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))

	if captured != "halo_5" {
		t.Errorf("sans header, la session doit faire autorité, attendu halo_5, got %q", captured)
	}
}
