// Package handlers — match_view_unavailable_e2e_test.go : E4.8a, de bout en bout. Une source de
// la vue match dont la lecture ÉCHOUE (ici les duels, sans raison dédiée) donne, par le vrai
// service et le vrai handler, une réponse 200 qui dit `section_unavailable` ET interdit tout
// cache HTTP ; une fois la source rétablie, la réponse suivante est complète et sans en-tête.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// repoDuelsEnPanne implémente les lectures que le service appelle ; kvErr pilote la panne des
// paires killer/victim (source sans raison dédiée). Tout le reste rend une vue pleine.
type repoDuelsEnPanne struct {
	port.MatchViewRepository
	kvErr error
}

func (r *repoDuelsEnPanne) GetMatchMeta(context.Context, string) (*domain.MatchMetaRaw, error) {
	now := time.Now()
	return &domain.MatchMetaRaw{MatchID: "m1", StartTime: &now}, nil
}
func (r *repoDuelsEnPanne) GetPlayerMatchStats(context.Context, string, string) (*domain.PlayerMatchStatsRaw, error) {
	bits := 1 << 9
	return &domain.PlayerMatchStatsRaw{OutcomeCode: 2, BackfillBits: &bits}, nil
}
func (r *repoDuelsEnPanne) GetMatchEnrichment(context.Context, string) (*domain.MatchEnrichmentRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchScoreboard(context.Context, string) ([]domain.ScoreboardRaw, error) {
	return []domain.ScoreboardRaw{{XUID: "xuid1", Gamertag: "Player1"}}, nil
}
func (r *repoDuelsEnPanne) GetMatchObjectiveScore(context.Context, string, string) (int, error) {
	return 0, nil
}
func (r *repoDuelsEnPanne) GetMatchMedals(context.Context, string, string) ([]domain.MedalRaw, error) {
	return []domain.MedalRaw{{MedalID: 1, Count: 1}}, nil
}
func (r *repoDuelsEnPanne) GetMatchEvents(context.Context, string) ([]domain.EventRaw, error) {
	return []domain.EventRaw{{EventType: "kill"}}, nil
}
func (r *repoDuelsEnPanne) LookupMedalMetaByName(context.Context, []string) (map[string]domain.MedalNameMeta, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchKillSources(context.Context, string) ([]domain.KillSourceRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchKillAssists(context.Context, string) ([]domain.KillAssistRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchAssistPairs(context.Context, string) ([]domain.MatchAssistPairRaw, domain.MatchAssistScopeRaw, error) {
	return nil, domain.MatchAssistScopeRaw{}, nil
}
func (r *repoDuelsEnPanne) GetMatchKVPairs(context.Context, string) ([]domain.KVPairRaw, error) {
	return nil, r.kvErr
}
func (r *repoDuelsEnPanne) GetMatchSkillRank(context.Context, string) (*domain.SkillRankRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchSharedCSRs(context.Context, string) (map[string]*domain.SkillRankRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchEncounters(context.Context, string, string) ([]domain.EncounterRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchEncounterStats(context.Context, string, string) ([]domain.EncounterStatsRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchEncounterAssists(context.Context, string, string) (map[string]domain.RelationAssists, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchMedia(context.Context, string) ([]domain.MediaAssocRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchExpectedStats(context.Context, string, string) (*domain.ExpectedStatsRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchBulkMedals(context.Context, string) ([]domain.BulkMedalRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetMatchBulkWeaponKills(context.Context, string) ([]domain.BulkWeaponKillRaw, error) {
	return nil, nil
}
func (r *repoDuelsEnPanne) GetHistoryForAvg(context.Context, string) ([]domain.MatchHistAvgRow, error) {
	return nil, nil
}

func TestMatchViewHandler_PanneDunChargeurSansRaisonDediee_SectionUnavailableEtNoStore(t *testing.T) {
	repo := &repoDuelsEnPanne{kvErr: errors.New("duels HS")}
	factory := func(context.Context, string) (port.MatchViewService, error) {
		return service.NewMatchViewService(repo, "xuid1"), nil
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

	repo.kvErr = nil // la source est rétablie
	w, resp = get()
	if resp.IsPartial || len(resp.PartialReasons) != 0 {
		t.Errorf("vue complète attendue après rétablissement, raisons %v", resp.PartialReasons)
	}
	if got := w.Header().Get("Cache-Control"); got != "" {
		t.Errorf("aucun en-tête de cache attendu sur la vue complète, obtenu %q", got)
	}
}
