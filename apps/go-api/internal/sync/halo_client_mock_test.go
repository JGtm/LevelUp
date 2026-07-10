// Package sync — halo_client_mock_test.go : mock déterministe de HaloClient pour les tests.
//
// Sprint 52 A5 : implémentation de mockHaloClient satisfaisant l'interface HaloClient.
// Ce fichier ne dépend pas du réseau et ne nécessite pas de build tag integration.
// Les tests qui l'utilisent tournent avec CGO_ENABLED=1 (requis par duckdb-go) mais
// sans accès réseau réel.
//
// A7 — FilmChunkData : le type reste non exporté car HaloClient est consommé uniquement
// dans le package sync. Tant qu'aucun consommateur externe n'utilise GetMatchFilm dans
// une expression de type (ex. var _ sync.HaloClient = &MyMock{}), l'export n'est pas requis.
package sync

import (
	"context"
	"errors"
	"sync/atomic"
)

// mockHaloClient est une implémentation de HaloClient qui retourne des fixtures
// déterministes. Aucun appel réseau n'est effectué.
type mockHaloClient struct {
	// history est la liste de matchs retournée par GetMatchHistory.
	history []MatchHistoryEntry
	// statsBody est le corps JSON retourné par GetMatchStats (indexé par match_id).
	statsBody map[string]map[string]any
	// getHistoryErr simule une erreur de GetMatchHistory si non nil.
	getHistoryErr error
	// getStatsErr simule une erreur de GetMatchStats si non nil.
	getStatsErr error
	// skillBody est la skill data retournée par GetMatchSkill (indexée par match_id).
	skillBody map[string]map[string]*MatchSkillData
	// getSkillErr simule une erreur de GetMatchSkill si non nil.
	getSkillErr error
	// careerData est la progression de carrière retournée par GetCareerRank.
	careerData *CareerRankData
	// getCareerErr simule une erreur de GetCareerRank si non nil.
	getCareerErr error
	// callsGetHistory compte le nombre d'appels à GetMatchHistory.
	// atomic.Int64 : le sync engine appelle ces méthodes depuis plusieurs
	// goroutines errgroup en parallèle (engine.go:798), donc l'incrémentation
	// scalaire serait une data race détectée par -race.
	callsGetHistory atomic.Int64
	// lastHistoryPlayer capture le dernier arg `player` reçu par GetMatchHistory
	// (anti-régression du format URL xuid(NNN) — incident mai 2026). atomic.Value
	// évite tout import "sync" (collision avec le nom du package). Vide tant que
	// GetMatchHistory n'a pas été appelé.
	lastHistoryPlayer atomic.Value // string
	// callsGetStats compte le nombre d'appels à GetMatchStats.
	callsGetStats atomic.Int64
	// callsGetSkill compte le nombre d'appels à GetMatchSkill.
	callsGetSkill atomic.Int64
	// highlightChunkData est le chunk highlight events retourné par
	// GetHighlightEventsChunk (nil = chunk absent).
	highlightChunkData []byte
	// highlightChunkVersion est le FilmMajorVersion associé au chunk.
	highlightChunkVersion int
	// highlightChunkFound indique si le film/chunk est marqué comme présent.
	highlightChunkFound bool
	// getHighlightErr simule une erreur de GetHighlightEventsChunk si non nil.
	getHighlightErr error
	// highlightChunkByMatch permet de différencier la réponse par match_id
	// (prioritaire sur les champs scalaires ci-dessus). Utilisé pour les tests
	// qui rejouent plusieurs matchs avec des comportements distincts (healed,
	// no_film, parse_anomaly).
	highlightChunkByMatch map[string]highlightChunkResponse
}

// highlightChunkResponse encapsule la réponse complète à GetHighlightEventsChunk
// pour un match donné — utilisé via highlightChunkByMatch.
type highlightChunkResponse struct {
	data    []byte
	version int
	found   bool
	err     error
}

// GetMatchHistory retourne la liste de matchs configurée ou une erreur simulée.
func (m *mockHaloClient) GetMatchHistory(
	_ context.Context,
	player, _ string,
	_, _ int,
) ([]MatchHistoryEntry, error) {
	m.callsGetHistory.Add(1)
	m.lastHistoryPlayer.Store(player)
	if m.getHistoryErr != nil {
		return nil, m.getHistoryErr
	}
	return m.history, nil
}

// GetMatchStats retourne le corps JSON configuré pour le match donné.
func (m *mockHaloClient) GetMatchStats(_ context.Context, matchID string) (map[string]any, error) {
	m.callsGetStats.Add(1)
	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}
	if m.statsBody != nil {
		if body, ok := m.statsBody[matchID]; ok {
			return body, nil
		}
	}
	// Corps minimal valide si aucune fixture spécifique n'est définie.
	return map[string]any{
		"MatchInfo": map[string]any{
			"MapVariant":          map[string]any{"AssetId": "map-id-1"},
			"GameVariantCategory": float64(9),
			"PlaylistExperience":  "Slayer",
		},
		"Players": []any{},
	}, nil
}

// GetMatchSkill retourne la skill data configurée pour le match donné.
func (m *mockHaloClient) GetMatchSkill(_ context.Context, matchID string, _ []string) (map[string]*MatchSkillData, error) {
	m.callsGetSkill.Add(1)
	if m.getSkillErr != nil {
		return nil, m.getSkillErr
	}
	if m.skillBody != nil {
		if body, ok := m.skillBody[matchID]; ok {
			return body, nil
		}
	}
	return map[string]*MatchSkillData{}, nil
}

// GetMatchFilm retourne toujours (nil, false, nil) : film absent.
func (m *mockHaloClient) GetMatchFilm(_ context.Context, _ string) (map[int]FilmChunkData, bool, error) {
	return nil, false, nil
}

// GetHighlightEventsChunk retourne le chunk configuré pour le match.
// Priorité : highlightChunkByMatch[matchID] > scalaires globaux.
func (m *mockHaloClient) GetHighlightEventsChunk(_ context.Context, matchID string) ([]byte, int, bool, error) {
	if r, ok := m.highlightChunkByMatch[matchID]; ok {
		return r.data, r.version, r.found, r.err
	}
	if m.getHighlightErr != nil {
		return nil, 0, false, m.getHighlightErr
	}
	return m.highlightChunkData, m.highlightChunkVersion, m.highlightChunkFound, nil
}

// GetCareerRank retourne la progression de carrière configurée.
func (m *mockHaloClient) GetCareerRank(_ context.Context, _ string) (*CareerRankData, error) {
	if m.getCareerErr != nil {
		return nil, m.getCareerErr
	}
	return m.careerData, nil
}

// GetPlayerCSRs retourne une liste vide (pas d'erreur) — le mock ne simule pas de CSR.
func (m *mockHaloClient) GetPlayerCSRs(_ context.Context, _, _ string) ([]PlayerPlaylistCSR, error) {
	return nil, nil
}

func (m *mockHaloClient) GetPlaylistCsr(_ context.Context, _, _, _ string) (*PlayerPlaylistCSR, error) {
	return nil, nil
}

// compile-time : mockHaloClient implémente HaloClient.
var _ HaloClient = (*mockHaloClient)(nil)

// ── Helpers d'assertion ──────────────────────────────────────────────────────

// errNetworkExpected est retournée si un test détecte un vrai appel réseau.
var errNetworkExpected = errors.New("appel réseau inattendu dans le test")
