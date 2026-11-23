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
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakeHistorySync est un port.HistorySyncService qui rend l'état ou le refus configuré.
type fakeHistorySync struct {
	status *domain.HistorySyncStatus
	err    error
	slug   string
}

func (f *fakeHistorySync) HistorySyncStatus(_ context.Context, slug string) (*domain.HistorySyncStatus, error) {
	f.slug = slug
	return f.status, f.err
}

func historyRouter(svc port.HistorySyncService, locale string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			if locale != "" {
				ctx = ctxkeys.WithLocale(ctx, locale)
			}
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Route("/players/{player_slug}", func(r chi.Router) { handlers.NewSyncHistoryHandler(svc).Mount(r) })
	return r
}

func getHistory(r http.Handler, slug string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/"+slug+"/sync/history", nil))
	return w
}

// L'état du service est rendu tel quel (libellés résolus, dates, comptes) pour le joueur du chemin.
func TestSyncHistory_RendLEtatDuJoueur(t *testing.T) {
	next := time.Date(2026, 11, 22, 14, 10, 0, 0, time.UTC)
	oldest := time.Date(2019, 9, 3, 18, 20, 0, 0, time.UTC)
	svc := &fakeHistorySync{status: &domain.HistorySyncStatus{
		AutoSyncEnabled: true,
		NextCycleAt:     &next,
		Titles: []domain.TitleHistoryStatus{{
			TitleSlug: "halo_infinite", TitleName: "Halo Infinite",
			State: domain.HistoryStateCatchingUp, StateLabel: "Rattrapage en cours",
			MatchCount: 6812, OldestMatchAt: &oldest,
			LastPass: &domain.HistoryPass{At: next.Add(-10 * time.Minute), MatchesAdded: 200, Status: "ok", StatusLabel: "Réussie"},
		}},
	}}
	w := getHistory(historyRouter(svc, "fr"), "JGtm")
	if w.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200 (corps %s)", w.Code, w.Body.String())
	}
	if svc.slug != "JGtm" {
		t.Errorf("joueur demandé = %q, attendu JGtm", svc.slug)
	}
	var body domain.HistorySyncStatus
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("corps illisible %q : %v", w.Body.String(), err)
	}
	if !body.AutoSyncEnabled || body.NextCycleAt == nil || !body.NextCycleAt.Equal(next) || len(body.Titles) != 1 {
		t.Fatalf("corps = %+v", body)
	}
	ti := body.Titles[0]
	if ti.State != "catching_up" || ti.StateLabel != "Rattrapage en cours" || ti.MatchCount != 6812 ||
		ti.LastPass == nil || ti.LastPass.MatchesAdded != 200 || ti.LastPass.StatusLabel != "Réussie" {
		t.Errorf("titre = %+v", ti)
	}
	var raw map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	for _, k := range []string{"title_name_fr", "title_name_en", "state_label_fr", "state_label_en"} {
		if _, ok := raw["titles"].([]any)[0].(map[string]any)[k]; ok {
			t.Errorf("la réponse porte une paire de langues %q : un seul libellé résolu attendu", k)
		}
	}
}

// Un joueur sans titre suivi : liste vide (jamais null), 200.
func TestSyncHistory_ListeVide(t *testing.T) {
	svc := &fakeHistorySync{status: &domain.HistorySyncStatus{Titles: []domain.TitleHistoryStatus{}}}
	w := getHistory(historyRouter(svc, ""), "JGtm")
	if w.Code != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", w.Code)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil || string(raw["titles"]) != "[]" {
		t.Errorf("titles = %s (err %v), attendu []", raw["titles"], err)
	}
}

func TestSyncHistory_Refus(t *testing.T) {
	t.Run("service en erreur : 500 au code fixe, jamais le texte de l'erreur", func(t *testing.T) {
		w := getHistory(historyRouter(&fakeHistorySync{err: errors.New("db_profiles.json illisible : C:\\secret")}, ""), "JGtm")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("statut = %d, attendu 500", w.Code)
		}
		code, message := errorBody(t, w)
		if code != "history_sync_failed" || message == "" || message == "db_profiles.json illisible : C:\\secret" {
			t.Errorf("(code, message) = (%q, %q)", code, message)
		}
	})
	t.Run("aucun service câblé : 503", func(t *testing.T) {
		w := getHistory(historyRouter(nil, ""), "JGtm")
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("statut = %d, attendu 503", w.Code)
		}
		if code, _ := errorBody(t, w); code != "scheduler_unavailable" {
			t.Errorf("code = %q, attendu scheduler_unavailable", code)
		}
	})
}
