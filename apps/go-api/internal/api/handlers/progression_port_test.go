// Package handlers_test — progression_port_test.go : les routes /streaks, /records et /milestones sur un
// faux port.ProgressionService : correspondance des erreurs typées, paramètres transmis, projection JSON.
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
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// fakeProgressionService enregistre les appels et rend les valeurs préparées.
type fakeProgressionService struct {
	streaks    []streaks.Streak
	records    port.PlayerRecords
	milestones []port.MilestoneStatus
	err        error

	gotPlayer, gotTitle string
	gotLimit            int
}

func (f *fakeProgressionService) Streaks(_ context.Context, player, title string) ([]streaks.Streak, error) {
	f.gotPlayer, f.gotTitle = player, title
	return f.streaks, f.err
}

func (f *fakeProgressionService) Records(_ context.Context, player, title string, limit int) (port.PlayerRecords, error) {
	f.gotPlayer, f.gotTitle, f.gotLimit = player, title, limit
	return f.records, f.err
}

func (f *fakeProgressionService) Milestones(_ context.Context, player, title string) ([]port.MilestoneStatus, error) {
	f.gotPlayer, f.gotTitle = player, title
	return f.milestones, f.err
}

func progressionRouter(svc port.ProgressionService) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewProgressionHandler(svc, "")
	r.Route("/api/v1/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	return r
}

func getJSON(t *testing.T, r http.Handler, path, locale string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if locale != "" {
		req.Header.Set("X-LevelUp-Locale", locale)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var body map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("corps illisible (%d) : %v : %s", rec.Code, err, rec.Body.String())
		}
	}
	return rec.Code, body
}

func TestProgressionHandler_StreaksProjetteLaListeEtPoseLeTitreParDefaut(t *testing.T) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	fake := &fakeProgressionService{streaks: []streaks.Streak{{
		ID: "s1", Type: streaks.StreakTypeDailyPlay, StartedAt: now, CurrentLength: 8, BestLength: 10,
		Status: streaks.StreakStatusActive,
	}}}
	code, body := getJSON(t, progressionRouter(fake), "/api/v1/players/jg/streaks", "")
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	items := body["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != "s1" || items[0].(map[string]any)["pp_multiplier"].(float64) != 1.25 {
		t.Fatalf("items = %v", items)
	}
	if fake.gotPlayer != "jg" || fake.gotTitle != "halo_infinite" {
		t.Fatalf("joueur %q, titre %q ; attendu jg et le titre par défaut", fake.gotPlayer, fake.gotTitle)
	}
}

func TestProgressionHandler_RecordsTransmetLaLimiteParDefaut(t *testing.T) {
	fake := &fakeProgressionService{records: port.PlayerRecords{
		PersonalBests: []records.PersonalRecord{{Metric: "performance_score", Period: records.RecordPeriodAllTime, Value: 92}},
		History:       []records.RecordHistory{{ID: "h1", Metric: "performance_score", Period: records.RecordPeriodAllTime, Value: 92}},
	}}
	r := progressionRouter(fake)
	code, body := getJSON(t, r, "/api/v1/players/jg/records", "")
	if code != http.StatusOK || fake.gotLimit != 50 {
		t.Fatalf("status %d, limite %d ; attendu 200 et 50", code, fake.gotLimit)
	}
	if len(body["personal_bests"].([]any)) != 1 || len(body["history"].([]any)) != 1 {
		t.Fatalf("corps = %v", body)
	}
	if code, _ := getJSON(t, r, "/api/v1/players/jg/records?history_limit=7", ""); code != http.StatusOK || fake.gotLimit != 7 {
		t.Fatalf("history_limit=7 : status %d, limite %d", code, fake.gotLimit)
	}
	if code, _ := getJSON(t, r, "/api/v1/players/jg/records?history_limit=201", ""); code != http.StatusUnprocessableEntity {
		t.Fatalf("history_limit=201 : status %d ; attendu 422 (borne du schéma)", code)
	}
}

func TestProgressionHandler_MilestonesResoutLeLibelleDansLaLangue(t *testing.T) {
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	fake := &fakeProgressionService{milestones: []port.MilestoneStatus{
		{Entry: milestones.CatalogEntry{ID: "m10", Metric: "matches_played", Threshold: 10,
			Title: i18n.FromColumns("Decimus", "Decimus FR")}, Earned: true, EarnedAt: &at},
		{Entry: milestones.CatalogEntry{ID: "m100", Metric: "matches_played", Threshold: 100,
			Title: i18n.FromColumns("Centurion", "Centurion FR")}, Earned: true},
		{Entry: milestones.CatalogEntry{ID: "m500", Metric: "matches_played", Threshold: 500,
			Title: i18n.FromColumns("Quingentus", "Quingentus FR")}},
	}}
	r := progressionRouter(fake)
	code, body := getJSON(t, r, "/api/v1/players/jg/milestones", "fr")
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	items := body["items"].([]any)
	if len(items) != 3 {
		t.Fatalf("items = %v", items)
	}
	first, second, third := items[0].(map[string]any), items[1].(map[string]any), items[2].(map[string]any)
	if first["title"] != "Decimus FR" || first["earned"] != true || first["earned_at"] == nil {
		t.Errorf("jalon franchi daté : %v", first)
	}
	if second["earned"] != true || second["earned_at"] != nil {
		t.Errorf("jalon franchi sans date : %v", second)
	}
	if third["earned"] != false || third["earned_at"] != nil {
		t.Errorf("jalon non franchi : %v", third)
	}
}

func TestProgressionHandler_ErreursTypeesEnReponsesHTTP(t *testing.T) {
	cause := errors.New("boom")
	for _, c := range []struct {
		nom    string
		path   string
		err    error
		status int
		code   string
	}{
		{"joueur introuvable", "streaks", &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: cause}, http.StatusNotFound, "player_not_found"},
		{"lecture des séries", "streaks", &port.ProgressionError{Kind: port.ProgErrListStreaks, Err: cause}, http.StatusInternalServerError, "list_streaks_error"},
		{"lecture des records", "records", &port.ProgressionError{Kind: port.ProgErrListPersonalBests, Err: cause}, http.StatusInternalServerError, "list_pbs_error"},
		{"lecture de l'historique", "records", &port.ProgressionError{Kind: port.ProgErrListRecordHistory, Err: cause}, http.StatusInternalServerError, "list_history_error"},
		{"lecture du catalogue", "milestones", &port.ProgressionError{Kind: port.ProgErrListCatalog, Err: cause}, http.StatusInternalServerError, "list_catalog_error"},
		{"lecture des jalons franchis", "milestones", &port.ProgressionError{Kind: port.ProgErrListEarned, Err: cause}, http.StatusInternalServerError, "list_earned_error"},
		{"erreur hors type", "streaks", cause, http.StatusInternalServerError, "progression_error"},
	} {
		r := progressionRouter(&fakeProgressionService{err: c.err})
		code, body := getJSON(t, r, "/api/v1/players/jg/"+c.path, "")
		if code != c.status || body["code"] != c.code {
			t.Errorf("%s : %d %v ; attendu %d %s", c.nom, code, body, c.status, c.code)
		}
		if msg, _ := body["message"].(string); msg == "boom" {
			t.Errorf("%s : le texte de la cause part au client", c.nom)
		}
	}
}
