package teammates

import (
	"context"
	"maps"
	"slices"
	"strings"

	"levelup/go-api/internal/analysis"
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
	// impactSynth : les frags reconstitués que le dépôt ajouterait aux matchs d'un groupe sans
	// frag ni mort natif (règle analysis.ImpactMatchesNeedingKVFallback, appliquée par groupe).
	impactSynth    []domain.ImpactEventRow
	kvPairs        []domain.KVPairRaw
	assistPairs    []domain.SquadAssistPairRaw
	assistMeasured int
	assistErr      error
	killLog        []domain.SquadKillLogRow
	kvErr          error
	synthRows      []legacymatch.SynthesisMatchRow
	synthErr       error
	allyRows       []domain.AllyParticipant
	allyErr        error
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
	// amis / profils / amisLus : les sources des coéquipiers connus du SCÉNARIO (pas des
	// lectures du dépôt Escouade) — les amis déclarés du joueur, le registre des profils du titre
	// et la lecture des amis hors registre (gamertag tel que demandé -> xuid), injectés par
	// avecConnus.
	amis    []string
	profils []domain.PlayerSummary
	amisLus map[string]string
}

// avecConnus branche sur svc les coéquipiers connus du scénario du dépôt : ses amis déclarés
// (quand il en déclare), le registre des profils et la lecture des amis hors registre.
func avecConnus(svc *TeammatesService, m *mockSquadRepo) *TeammatesService {
	if m.amis != nil {
		amis := slices.Clone(m.amis)
		svc.friendGamertags = func(context.Context) []string { return amis }
	}
	profils := slices.Clone(m.profils)
	lus := maps.Clone(m.amisLus)
	return svc.WithCoequipiersConnus(
		func(context.Context) ([]domain.PlayerSummary, error) { return profils, nil },
		func(_ context.Context, gts []string) (map[string]string, error) {
			out := map[string]string{}
			for _, gt := range gts {
				if x, ok := lus[gt]; ok {
					out[gt] = x
				}
			}
			return out, nil
		},
	)
}

// profilSuivi : un profil suivi du titre (sync actif, pas auth_only).
func profilSuivi(xuid, gamertag string) domain.PlayerSummary {
	return domain.PlayerSummary{XUID: xuid, Gamertag: gamertag, PlayerSlug: gamertag, SyncEnabled: true}
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
func (m *mockSquadRepo) LoadImpactEvents(ctx context.Context, ids []string) ([]domain.ImpactEventRow, error) {
	return m.LoadImpactEventsParGroupes(ctx, [][]string{ids})
}

// LoadImpactEventsParGroupes rend impactRows (sans filtre par match, comme LoadImpactEvents
// l'a toujours fait ici), plus les lignes d'impactSynth des matchs que la règle du dépôt ferait
// reconstituer, groupe par groupe.
func (m *mockSquadRepo) LoadImpactEventsParGroupes(_ context.Context, groupes [][]string) ([]domain.ImpactEventRow, error) {
	if m.impactErr != nil || len(m.impactSynth) == 0 {
		return m.impactRows, m.impactErr
	}
	besoin := map[string]bool{}
	for _, id := range analysis.ImpactMatchesNeedingKVFallback(m.impactRows, groupes) {
		besoin[id] = true
	}
	out := slices.Clone(m.impactRows)
	for _, r := range m.impactSynth {
		if besoin[r.MatchID] {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *mockSquadRepo) LoadKVPairs(_ context.Context, _ []string) ([]domain.KVPairRaw, error) {
	return m.kvPairs, m.kvErr
}
func (m *mockSquadRepo) LoadSquadAssistPairs(_ context.Context, _, _ []string) ([]domain.SquadAssistPairRaw, int, error) {
	return m.assistPairs, m.assistMeasured, m.assistErr
}
func (m *mockSquadRepo) LoadSquadKillLog(_ context.Context, _, _ []string) ([]domain.SquadKillLogRow, error) {
	return m.killLog, nil
}
func (m *mockSquadRepo) LoadMainTeamParticipants(_ context.Context, _ string, _ []string) ([]domain.AllyParticipant, error) {
	return m.allyRows, m.allyErr
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
