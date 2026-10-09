// Package handlers_test — teammates_sessions_test.go : GET /pages/teammates/sessions, la
// lecture légère des sessions de la composition (lot perf L4b, 2026-09-23). Même sous-routeur
// que POST /pages/teammates (Mount unique) ; les erreurs 499/503 communes aux pages sont
// couvertes par page_service_errors_test.go.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// serveTeammatesSessions monte le handler teammates et appelle la route légère.
func serveTeammatesSessions(mock *mockTeammatesService, slug, query string) *httptest.ResponseRecorder {
	r := newTeammatesRouter(func(_ context.Context, s string) (port.TeammatesService, string, string, error) {
		if s != testPlayerSlug {
			return nil, "", "", errors.New("player_not_found")
		}
		return mock, testXUID1, testGamertag, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/players/"+slug+"/pages/teammates/sessions"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTeammatesSessions_RendLesSessionsDuService(t *testing.T) {
	start := time.Date(2026, 9, 20, 19, 0, 0, 0, time.UTC)
	mock := &mockTeammatesService{
		sessions: []domain.CompositionSessionEntry{{
			SessionLabelEntry: domain.SessionLabelEntry{Label: "S2 (3)", StartedAt: start, EndedAt: start, MatchCount: 2},
			MatchCountRoster:  3,
			ExcludedByExactComposition: []domain.CompositionExcludedMatch{
				{MatchID: "m3", StartTime: start, MapUI: "Bazaar", ExtraGamertags: []string{"Carol"}},
			},
		}},
		latest: "S2 (3)",
	}
	w := serveTeammatesSessions(mock, testPlayerSlug, "?teammates=Alice,%20Bob,,&exact=true")
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	// La composition arrive nettoyée (rognée, sans vide), avec l'option et le joueur de la page.
	if !slices.Equal(mock.gotTeammates, []string{"Alice", "Bob"}) || !mock.gotExact || mock.gotXUID != testXUID1 {
		t.Errorf("appel du service : xuid %q, composition %q, exact %v", mock.gotXUID, mock.gotTeammates, mock.gotExact)
	}
	var body struct {
		Sessions []struct {
			Label    string `json:"label"`
			Count    int    `json:"match_count"`
			Roster   int    `json:"match_count_roster"`
			Excluded []struct {
				MatchID string   `json:"match_id"`
				Extra   []string `json:"extra_gamertags"`
			} `json:"excluded_by_exact_composition"`
		} `json:"composition_sessions"`
		Latest string `json:"latest_composition_session"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("décodage : %v", err)
	}
	if len(body.Sessions) != 1 || body.Latest != "S2 (3)" {
		t.Fatalf("corps : %s", w.Body.String())
	}
	s := body.Sessions[0]
	if s.Label != "S2 (3)" || s.Count != 2 || s.Roster != 3 || len(s.Excluded) != 1 ||
		s.Excluded[0].MatchID != "m3" || !slices.Equal(s.Excluded[0].Extra, []string{"Carol"}) {
		t.Errorf("session publiée : %+v", s)
	}
}

// TestTeammatesSessions_SansCoequipier : composition absente = aucun coéquipier, option
// absente = false (comme filter_exact_composition absent du corps de la page) ; les deux
// champs sont TOUJOURS présents, une liste vide n'est jamais null.
func TestTeammatesSessions_SansCoequipier(t *testing.T) {
	for _, query := range []string{"", "?teammates=", "?teammates=,%20"} {
		t.Run(query, func(t *testing.T) {
			mock := &mockTeammatesService{}
			w := serveTeammatesSessions(mock, testPlayerSlug, query)
			if w.Code != http.StatusOK {
				t.Fatalf("statut %d : %s", w.Code, w.Body.String())
			}
			if len(mock.gotTeammates) != 0 || mock.gotExact {
				t.Errorf("appel du service : composition %q, exact %v", mock.gotTeammates, mock.gotExact)
			}
			raw := w.Body.String()
			if !strings.Contains(raw, `"composition_sessions":[]`) || !strings.Contains(raw, `"latest_composition_session":""`) {
				t.Errorf("champs absents ou null : %s", raw)
			}
		})
	}
}

func TestTeammatesSessions_JoueurInconnu_404(t *testing.T) {
	w := serveTeammatesSessions(&mockTeammatesService{}, "inconnu", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("statut %d, attendu 404", w.Code)
	}
}

// TestTeammatesSessions_CapabilityAbsente_503 : un titre sans la capability rend un 503
// propre (MapCapabilityError), jamais un 500.
func TestTeammatesSessions_CapabilityAbsente_503(t *testing.T) {
	mock := &mockTeammatesService{pageErr: fmt.Errorf("historique: %w", games.ErrCapabilityNotSupported)}
	w := serveTeammatesSessions(mock, testPlayerSlug, "?teammates=Alice")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("statut %d, attendu 503 : %s", w.Code, w.Body.String())
	}
	if code := errorCode(t, w); code != "capability_not_supported" {
		t.Errorf("code %q, attendu capability_not_supported", code)
	}
}

func TestTeammatesSessions_ErreurService_500(t *testing.T) {
	w := serveTeammatesSessions(&mockTeammatesService{pageErr: errors.New("db")}, testPlayerSlug, "?teammates=Alice")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("statut %d, attendu 500", w.Code)
	}
	if code := errorCode(t, w); code != "teammates_sessions_error" {
		t.Errorf("code %q, attendu teammates_sessions_error", code)
	}
}

func TestTeammatesSessions_OptionIllisible_422(t *testing.T) {
	w := serveTeammatesSessions(&mockTeammatesService{}, testPlayerSlug, "?teammates=Alice&exact=peut-etre")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("statut %d, attendu 422 : %s", w.Code, w.Body.String())
	}
}
