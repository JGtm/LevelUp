//go:build integration

// progression_i18n_test.go — contrat i18n de GET /milestones (ADR 0040 D-3, plan i18n étape 6, lot B).
// Même stratégie que progression_test.go : DuckDB temporaire + PlayerDB + routeur chi + httptest, mais
// derrière le vrai middleware de locale (X-LevelUp-Locale).

package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
)

func TestListMilestones_TitreEtConditionSuiventLaLangue(t *testing.T) {
	pdb, _ := setupHandlerEnv(t)
	ctx := context.Background()

	catRepo := duckdb.NewMilestoneCatalogRepo(pdb.Metadata)
	for _, e := range []milestones.CatalogEntry{
		{
			ID: "h.matches.100", TitleSlug: progTestTitle, Metric: "matches_played", Threshold: 100,
			Title:         i18n.FromColumns("Centurion", "Centurion FR"),
			ConditionText: i18n.FromColumns("Play 100 matches", "Jouer 100 parties"),
		},
		// Sans condition : le champ est omis, dans toutes les langues.
		{
			ID: "h.wins.10", TitleSlug: progTestTitle, Metric: "wins", Threshold: 10,
			Title: i18n.FromColumns("Winner", "Vainqueur"),
		},
	} {
		if err := catRepo.Upsert(ctx, e); err != nil {
			t.Fatalf("seed catalog %s: %v", e.ID, err)
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.TitleExtractor(titlePkg.NewRegistry()))
	resolver := func(_ context.Context, slug string) (*duckdb.PlayerDB, error) {
		if slug != progTestSlug {
			return nil, errNotFound
		}
		return pdb, nil
	}
	h := handlers.NewProgressionHandler(resolver, progTestTitle)
	r.Route("/api/v1/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })

	get := func(locale string) map[string]map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+progTestSlug+"/milestones", nil)
		if locale != "" {
			req.Header.Set("X-LevelUp-Locale", locale)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("locale %q : status %d : %s", locale, rec.Code, rec.Body.String())
		}
		var body struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode : %v", err)
		}
		byID := map[string]map[string]any{}
		for _, it := range body.Items {
			byID[it["id"].(string)] = it
		}
		return byID
	}

	for _, c := range []struct{ locale, title, condition, winTitle string }{
		{"fr", "Centurion FR", "Jouer 100 parties", "Vainqueur"},
		{"en", "Centurion", "Play 100 matches", "Winner"},
		{"zz", "Centurion FR", "Jouer 100 parties", "Vainqueur"}, // locale inconnue : défaut fr
		{"", "Centurion FR", "Jouer 100 parties", "Vainqueur"},   // en-tête absent : défaut fr
	} {
		items := get(c.locale)
		cent, wins := items["h.matches.100"], items["h.wins.10"]
		if cent["title"] != c.title || cent["condition"] != c.condition || wins["title"] != c.winTitle {
			t.Errorf("locale %q : %v / %v", c.locale, cent, wins)
		}
		if _, ok := wins["condition"]; ok {
			t.Errorf("locale %q : `condition` doit être omis sans condition : %v", c.locale, wins)
		}
		for _, k := range []string{"title_en", "title_fr", "condition_en", "condition_fr"} {
			if _, ok := cent[k]; ok {
				t.Errorf("locale %q : le champ %q ne fait plus partie du contrat", c.locale, k)
			}
		}
	}
}
