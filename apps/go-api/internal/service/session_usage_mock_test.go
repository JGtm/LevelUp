package service

// session_usage_mock_test.go — le résumé d'usage en mémoire (port.SessionUsageRepository) partagé par
// les tests des blocs de la page Sessions et des Séries temporelles.

import (
	"context"

	"levelup/go-api/internal/analysis/sessionusage"
)

type mockSessionUsageRepo struct {
	films        map[string]sessionusage.FilmRow
	players      []sessionusage.PlayerRow
	participants []sessionusage.ParticipantRow
	filmsErr     error
	padTiers     []sessionusage.PadTierRow
	padTiersErr  error
}

func (m *mockSessionUsageRepo) LoadUsageFilms(_ context.Context, _ []string) (map[string]sessionusage.FilmRow, error) {
	return m.films, m.filmsErr
}
func (m *mockSessionUsageRepo) LoadUsagePlayers(_ context.Context, _ []string) ([]sessionusage.PlayerRow, error) {
	return m.players, nil
}
func (m *mockSessionUsageRepo) LoadParticipants(_ context.Context, _ []string) ([]sessionusage.ParticipantRow, error) {
	return m.participants, nil
}

// LoadPadTiers — les prises de socle par niveau d'arme. `padTiers` nil = aucune ligne, donc
// « non mesuré » : l'état par défaut de tous les témoins qui n'en parlent pas.
func (m *mockSessionUsageRepo) LoadPadTiers(_ context.Context, _ []string) ([]sessionusage.PadTierRow, error) {
	return m.padTiers, m.padTiersErr
}

func teamp(v int) *int { return &v }

// usageTestRepoMock — deux matchs, trois joueurs (P et A d'un camp, E1 de l'autre), un film sur m1.
func usageTestRepoMock() *mockSessionUsageRepo {
	return &mockSessionUsageRepo{
		films: map[string]sessionusage.FilmRow{
			"m1": {MatchID: "m1", DurationMS: 600000},
		},
		players: []sessionusage.PlayerRow{
			{MatchID: "m1", XUID: "P", PadPickups: 2},
			{MatchID: "m1", XUID: "A", PadPickups: 1},
			{MatchID: "m1", XUID: "E1", PadPickups: 1},
		},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
		},
	}
}
