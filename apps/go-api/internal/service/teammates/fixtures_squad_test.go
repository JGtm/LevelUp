package teammates

import (
	"context"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fixtures_squad_test.go — fixtures de test dupliquées de service/squad_service_v2_test.go
// (K3b : packages de test disjoints après extraction du sous-package teammates).

// fakeSquadLoader mock de squadagg.SquadV2Loader pour les tests.
type fakeSquadLoader struct {
	mu         sync.Mutex
	rowsByGT   map[string][]canonical.PlayerMatchRow
	errByGT    map[string]error
	calls      []string // gamertags appeles dans l'ordre
	delayPerGT time.Duration
	// objByGT : si renseigné, LoadObjectiveScores retourne ces scores par gamertag.
	objByGT map[string]map[string]int
}

func (f *fakeSquadLoader) LoadFor(
	_ context.Context,
	_, gamertag string,
	_ port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	f.mu.Lock()
	f.calls = append(f.calls, gamertag)
	f.mu.Unlock()
	if f.delayPerGT > 0 {
		time.Sleep(f.delayPerGT)
	}
	if err, ok := f.errByGT[gamertag]; ok {
		return nil, err
	}
	return f.rowsByGT[gamertag], nil
}

func (f *fakeSquadLoader) LoadHighlightEvents(
	_ context.Context, _ string, _ port.HighlightEventFilters,
) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (f *fakeSquadLoader) LoadWeaponKills(
	_ context.Context, _ string, _ port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (f *fakeSquadLoader) LoadKillMechanics(
	_ context.Context, _ string, _ port.WeaponKillFilters,
) ([]port.KillMechanicsRow, error) {
	return nil, nil
}

func (f *fakeSquadLoader) LoadMedals(
	_ context.Context, _ string, _ port.MedalsByXUIDFilters,
) ([]port.MedalRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

func (f *fakeSquadLoader) LoadEmblemURLs(
	_ context.Context, _ string, _ []string,
) map[string]string {
	return nil
}

func (f *fakeSquadLoader) LoadMapStatsForSquad(
	_ context.Context, _, _ string, _ []string,
) (map[string]domain.MapSquadStats, error) {
	return nil, nil
}

func (f *fakeSquadLoader) LoadObjectiveScores(
	_ context.Context, _, gamertag string, _ []string,
) (map[string]int, error) {
	if f.objByGT != nil {
		if scores, ok := f.objByGT[gamertag]; ok {
			return scores, nil
		}
	}
	return map[string]int{}, nil
}

// rowWithStats construit une PlayerMatchRow avec les stats Self renseignées.
func rowWithStats(matchID string, ts time.Time, outcome canonical.Outcome,
	kills, deaths, assists, timePlayed int, accuracy float64,
	performanceScore float64,
) canonical.PlayerMatchRow {
	k, d, a, tp := kills, deaths, assists, timePlayed
	acc := accuracy
	perf := performanceScore
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: ts,
			Outcome:      outcome,
		},
		Self: canonical.MatchParticipant{
			Outcome:    outcome,
			Kills:      &k,
			Deaths:     &d,
			Assists:    &a,
			TimePlayed: &tp,
			Accuracy:   &acc,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: &perf,
		},
	}
}

// rowWithStatsXUID est rowWithStats avec un xuid renseigne sur Self.Identity.
func rowWithStatsXUID(xuid, matchID string, ts time.Time, outcome canonical.Outcome,
	kills, deaths, assists, timePlayed int, accuracy float64, perfScore float64,
) canonical.PlayerMatchRow {
	row := rowWithStats(matchID, ts, outcome, kills, deaths, assists, timePlayed, accuracy, perfScore)
	row.Self.Identity.XUID = xuid
	return row
}
