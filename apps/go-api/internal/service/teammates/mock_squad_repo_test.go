package teammates

import (
	"context"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// mockSquadRepo est un mock de port.SquadRepository pour les tests TeammatesService.
// Dupliqué de service/squad_service_test.go (K3b : packages de test disjoints après
// extraction du sous-package teammates ; un mock _test.go n'est pas partageable).
type mockSquadRepo struct {
	topRows   []domain.TopTeammateRow
	topErr    error
	squadRows []domain.SquadMatchRow
	// squadRowsByTeammate (optionnel) : matchs communs par XUID de coéquipier.
	squadRowsByTeammate map[string][]domain.SquadMatchRow
	squadErr            error
	tmRows              []domain.TeammateMatchRow
	tmErr               error
	impactRows          []domain.ImpactEventRow
	impactErr           error
	kvPairs             []domain.KVPairRaw
	assistPairs         []domain.SquadAssistPairRaw
	assistMeasured      int
	assistErr           error
	kvErr               error
	heatmapRows         []domain.SynthesisHeatmapRow
	heatmapErr          error
	synthRows           []legacymatch.SynthesisMatchRow
	synthErr            error
	allyRows            []domain.AllyParticipant
	allyErr             error
	// mapStats + captures : renvoyé par LoadMapStatsForSquad ; les slices capturent
	// les derniers arguments reçus (composition exacte : test de l'anti-join pool).
	mapStats             map[string]domain.MapSquadStats
	mapStatsErr          error
	mapStatsSquadXUIDs   []string
	mapStatsExcludeXUIDs []string
	// lookupAliases : gamertag normalisé (lowercase) -> xuid. Vide -> ("", false, nil).
	lookupAliases map[string]string
	lookupErr     error
	// assetFR : traductions FR par type d'asset -> (asset_id -> libellé FR).
	assetFR map[string]map[string]string
	// modeFR : mode_name_tr FR (mode EN normalisé -> FR).
	modeFR map[string]string
}

func (m *mockSquadRepo) LoadTopTeammates(_ context.Context, _ string) ([]domain.TopTeammateRow, error) {
	return m.topRows, m.topErr
}
func (m *mockSquadRepo) LookupXUIDByGamertag(_ context.Context, gamertag string) (string, bool, error) {
	if m.lookupErr != nil {
		return "", false, m.lookupErr
	}
	if x, ok := m.lookupAliases[strings.ToLower(gamertag)]; ok {
		return x, true, nil
	}
	return "", false, nil
}
func (m *mockSquadRepo) LoadSquadMatches(_ context.Context, _, teammateXUID string) ([]domain.SquadMatchRow, error) {
	if m.squadErr != nil {
		return nil, m.squadErr
	}
	if m.squadRowsByTeammate != nil {
		return m.squadRowsByTeammate[teammateXUID], nil
	}
	return m.squadRows, nil
}
func (m *mockSquadRepo) LoadTeammateMatches(_ context.Context, _, _ string) ([]domain.TeammateMatchRow, error) {
	return m.tmRows, m.tmErr
}
func (m *mockSquadRepo) LoadImpactEvents(_ context.Context, _ []string) ([]domain.ImpactEventRow, error) {
	return m.impactRows, m.impactErr
}
func (m *mockSquadRepo) LoadKVPairs(_ context.Context, _ []string) ([]domain.KVPairRaw, error) {
	return m.kvPairs, m.kvErr
}
func (m *mockSquadRepo) LoadSquadAssistPairs(_ context.Context, _, _ []string) ([]domain.SquadAssistPairRaw, int, error) {
	return m.assistPairs, m.assistMeasured, m.assistErr
}
func (m *mockSquadRepo) LoadMainTeamParticipants(_ context.Context, _ string, _ []string) ([]domain.AllyParticipant, error) {
	return m.allyRows, m.allyErr
}
func (m *mockSquadRepo) LoadSynthesisHeatmap(_ context.Context, _ string) ([]domain.SynthesisHeatmapRow, error) {
	return m.heatmapRows, m.heatmapErr
}
func (m *mockSquadRepo) LoadAssetTranslationsFR(_ context.Context, assetType string, _ []string) (map[string]string, error) {
	if m.assetFR == nil {
		return nil, nil
	}
	return m.assetFR[assetType], nil
}
func (m *mockSquadRepo) LoadModeTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return m.modeFR, nil
}
func (m *mockSquadRepo) LoadMapStatsForSquad(_ context.Context, _ string, squadXUIDs, excludeXUIDs []string) (map[string]domain.MapSquadStats, error) {
	m.mapStatsSquadXUIDs = squadXUIDs
	m.mapStatsExcludeXUIDs = excludeXUIDs
	return m.mapStats, m.mapStatsErr
}
func (m *mockSquadRepo) LoadSynthesisMatches(_ context.Context, _ string) ([]legacymatch.SynthesisMatchRow, error) {
	return m.synthRows, m.synthErr
}
