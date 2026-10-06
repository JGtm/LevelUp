package service

// session_usage_mock_test.go — le résumé d'usage en mémoire (port.SessionUsageRepository) partagé par
// les tests des blocs de la page Sessions et des Séries temporelles. Comme le dépôt réel, il ne rend
// que les lignes des matchs DEMANDÉS, et il note les identifiants de chaque lecture : une lecture faite
// sur la mauvaise liste de matchs se voit dans le contenu servi ET dans `lectures`.

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
	// lectures : les identifiants reçus par chaque lecture, par méthode (« films », « players »,
	// « participants », « pad_tiers »), dans l'ordre des appels.
	lectures map[string][][]string
}

// noter garde les identifiants d'une lecture et rend l'ensemble des matchs demandés.
func (m *mockSessionUsageRepo) noter(methode string, ids []string) map[string]bool {
	if m.lectures == nil {
		m.lectures = map[string][][]string{}
	}
	m.lectures[methode] = append(m.lectures[methode], append([]string(nil), ids...))
	demandes := make(map[string]bool, len(ids))
	for _, id := range ids {
		demandes[id] = true
	}
	return demandes
}

func (m *mockSessionUsageRepo) LoadUsageFilms(_ context.Context, ids []string) (map[string]sessionusage.FilmRow, error) {
	demandes := m.noter("films", ids)
	if m.filmsErr != nil {
		return nil, m.filmsErr
	}
	out := map[string]sessionusage.FilmRow{}
	for id, f := range m.films {
		if demandes[id] {
			out[id] = f
		}
	}
	return out, nil
}

func (m *mockSessionUsageRepo) LoadUsagePlayers(_ context.Context, ids []string) ([]sessionusage.PlayerRow, error) {
	return filtrerParMatch(m.players, m.noter("players", ids), func(r sessionusage.PlayerRow) string { return r.MatchID }), nil
}

func (m *mockSessionUsageRepo) LoadParticipants(_ context.Context, ids []string) ([]sessionusage.ParticipantRow, error) {
	return filtrerParMatch(m.participants, m.noter("participants", ids), func(r sessionusage.ParticipantRow) string { return r.MatchID }), nil
}

// LoadPadTiers — les prises de socle par niveau d'arme. `padTiers` nil = aucune ligne, donc
// « non mesuré » : l'état par défaut de tous les témoins qui n'en parlent pas.
func (m *mockSessionUsageRepo) LoadPadTiers(_ context.Context, ids []string) ([]sessionusage.PadTierRow, error) {
	demandes := m.noter("pad_tiers", ids)
	if m.padTiersErr != nil {
		return nil, m.padTiersErr
	}
	return filtrerParMatch(m.padTiers, demandes, func(r sessionusage.PadTierRow) string { return r.MatchID }), nil
}

// filtrerParMatch — les lignes des seuls matchs demandés, dans leur ordre.
func filtrerParMatch[T any](rows []T, demandes map[string]bool, matchID func(T) string) []T {
	var out []T
	for _, r := range rows {
		if demandes[matchID(r)] {
			out = append(out, r)
		}
	}
	return out
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
