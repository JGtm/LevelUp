// Package handlers — match_view_unavailable_lot_b_e2e_test.go : E4.8b, de bout en bout. Les sources
// du lot B (rang, CSR partagés, médias, stats attendues, médailles et armes de tous les joueurs,
// historique, extras des amis) dont la lecture ÉCHOUE donnent, par le vrai service et le vrai
// handler, un 200 qui dit `section_unavailable` ET interdit tout cache HTTP ; une fois la source
// rétablie, la réponse suivante est complète et sans en-tête.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// repoLotB : la vue pleine de repoDuelsEnPanne, avec une lecture du lot B pilotable par son nom.
type repoLotB struct {
	repoDuelsEnPanne
	panne string // nom de la méthode en panne, "" = aucune
}

func (r *repoLotB) echec(nom string) error {
	if r.panne == nom {
		return errors.New(nom + " HS")
	}
	return nil
}

// GetMatchScoreboard porte un ami en plus du joueur : sans lui, aucun extra d'ami n'est demandé.
func (r *repoLotB) GetMatchScoreboard(context.Context, string) ([]domain.ScoreboardRaw, error) {
	return []domain.ScoreboardRaw{{XUID: "xuid1", Gamertag: "Player1"}, {XUID: "xuid2", Gamertag: "Friend"}}, nil
}
func (r *repoLotB) GetMatchSkillRank(context.Context, string) (*domain.SkillRankRaw, error) {
	return nil, r.echec("GetMatchSkillRank")
}
func (r *repoLotB) GetMatchSharedCSRs(context.Context, string) (map[string]*domain.SkillRankRaw, error) {
	return nil, r.echec("GetMatchSharedCSRs")
}
func (r *repoLotB) GetMatchMedia(context.Context, string) ([]domain.MediaAssocRaw, error) {
	return nil, r.echec("GetMatchMedia")
}
func (r *repoLotB) GetMatchExpectedStats(context.Context, string, string) (*domain.ExpectedStatsRaw, error) {
	return nil, r.echec("GetMatchExpectedStats")
}
func (r *repoLotB) GetMatchBulkMedals(context.Context, string) ([]domain.BulkMedalRaw, error) {
	return nil, r.echec("GetMatchBulkMedals")
}
func (r *repoLotB) GetMatchBulkWeaponKills(context.Context, string) ([]domain.BulkWeaponKillRaw, error) {
	return nil, r.echec("GetMatchBulkWeaponKills")
}
func (r *repoLotB) GetHistoryForAvg(context.Context, string) ([]domain.MatchHistAvgRow, error) {
	return nil, r.echec("GetHistoryForAvg")
}

func TestMatchViewHandler_PanneDUneSourceDuLotB_SectionUnavailableEtNoStore(t *testing.T) {
	sources := []string{
		"GetMatchSkillRank", "GetMatchSharedCSRs", "GetMatchMedia", "GetMatchExpectedStats",
		"GetMatchBulkMedals", "GetMatchBulkWeaponKills", "GetHistoryForAvg", "amis",
	}
	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			repo := &repoLotB{panne: source}
			// Les extras des amis (hors errgroup) : un ami lisible, dont la player DB échoue.
			amis := func(context.Context, string, string, []string) (map[string]port.FriendMatchExtras, error) {
				if repo.panne == "amis" {
					return nil, errors.New("player DB d'un ami HS")
				}
				return nil, nil
			}
			factory := func(context.Context, string) (port.MatchViewService, error) {
				return service.NewMatchViewService(repo, "xuid1").WithFriendsExtras(amis), nil
			}
			get := func() (*httptest.ResponseRecorder, domain.MatchViewResponse) {
				w := httptest.NewRecorder()
				newMatchViewRouter(factory).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m1", nil))
				if w.Code != http.StatusOK {
					t.Fatalf("200 partiel attendu, obtenu %d : %s", w.Code, w.Body.String())
				}
				var resp domain.MatchViewResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("unmarshal : %v", err)
				}
				return w, resp
			}

			w, resp := get()
			if !slices.Equal(resp.PartialReasons, []string{"section_unavailable"}) {
				t.Errorf("raisons = %v, attendu [section_unavailable]", resp.PartialReasons)
			}
			if got := w.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, no-store attendu sur une vue en panne partielle", got)
			}

			repo.panne = "" // la source est rétablie
			w, resp = get()
			if resp.IsPartial || len(resp.PartialReasons) != 0 {
				t.Errorf("vue complète attendue après rétablissement, raisons %v", resp.PartialReasons)
			}
			if got := w.Header().Get("Cache-Control"); got != "" {
				t.Errorf("aucun en-tête de cache attendu sur la vue complète, obtenu %q", got)
			}
		})
	}
}
