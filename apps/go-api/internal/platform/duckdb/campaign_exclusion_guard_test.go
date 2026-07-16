package duckdb

import (
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestCampaignExclusionTokenWiredInStatQueries — GARDE-RAIL (item H1, règle
// CLAUDE.md n°6 : centralisation + garde-rail). Chaque requête de stats agrégées
// per-player qui liste/agrège les matchs du joueur DOIT porter le token
// d'exclusion Campagne (résolu au runtime par le caller via le titre). Si un
// refactor déplace/supprime le token d'une de ces requêtes, la fuite Campagne
// (Halo 5) réapparaît silencieusement sur la surface concernée → ce test échoue.
//
// N.B. un token laissé NON résolu resterait un commentaire SQL inoffensif (pas de
// crash) mais NE FILTRERAIT PAS — d'où la vérification comportementale séparée
// (campaign_exclusion_behavior_test.go, tag integration) qui prouve le filtrage
// effectif de bout en bout.
func TestCampaignExclusionTokenWiredInStatQueries(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"Q9TopMatchesSharedTpl (career top matches)", Q9TopMatchesSharedTpl},
		{"Q9bHighlightSharedTpl (career highlights)", Q9bHighlightSharedTpl},
		{"Q22SessionMatches (sessions)", Q22SessionMatches},
		{"Q23StatsMatchesShared (stats/perf)", Q23StatsMatchesShared},
		{"Q26HomeMatchesSharedPart (home matches)", Q26HomeMatchesSharedPart},
		{"Q26gPlaylistPhaseBShared (home last playlists)", Q26gPlaylistPhaseBShared},
		{"Q33SynthesisHeatmap (synthesis heatmap)", Q33SynthesisHeatmap},
		{"Q33bSynthesisSharedQuery (synthesis top weeks)", Q33bSynthesisSharedQuery},
		{"Q26bCountPlayerMatches (home total count)", Q26bCountPlayerMatches},
		{"Q19cTargetRecentMatches (explorer target recent)", Q19cTargetRecentMatches},
		{"Q29HistoryForAvg (match view vs moyenne)", Q29HistoryForAvg},
		{"Q29HistoryForAvgBulkTpl (match view vs moyenne, bulk)", Q29HistoryForAvgBulkTpl},
		{"Q25NeighborMatches (match nav prev/next)", Q25NeighborMatches},
		{"Q1MatchCount (compteur matchs bootstrap/filtres)", Q1MatchCount},
	}
	for _, c := range cases {
		if !strings.Contains(c.query, campaignExclusionToken) {
			t.Errorf("%s : token d'exclusion Campagne absent — la surface fuiterait les matchs Campagne (Halo 5)", c.name)
		}
	}
}

// TestResolveCampaignExclusionDelegation — les wrappers package-local délèguent
// bien à la source unique analysis (title-aware, no-op Infinite).
func TestResolveCampaignExclusionDelegation(t *testing.T) {
	q := "WHERE mp.xuid = ? " + campaignExclusionToken

	h5 := resolveCampaignExclusion(q, "halo_5", "r")
	if strings.Contains(h5, campaignExclusionToken) || !strings.Contains(h5, "NOT IN") {
		t.Errorf("halo_5 : token non résolu en clause NOT IN : %q", h5)
	}
	if hinf := resolveCampaignExclusion(q, "halo_infinite", "r"); strings.Contains(hinf, "game_variant_id") {
		t.Errorf("halo_infinite : aucune clause ne doit être injectée : %q", hinf)
	}
	if lit := excludeCampaignClause("halo_5", "r"); !strings.Contains(lit, "r.game_variant_id") {
		t.Errorf("excludeCampaignClause halo_5 vide/incorrect : %q", lit)
	}
	if lit := excludeCampaignClause("halo_infinite", "r"); lit != "" {
		t.Errorf("excludeCampaignClause halo_infinite doit être vide : %q", lit)
	}
	// Cohérence avec la source unique.
	if len(analysis.CampaignExcludedVariantIDs("halo_5")) != 2 {
		t.Errorf("source unique attendue = 2 GUID Campagne Halo 5")
	}
}
