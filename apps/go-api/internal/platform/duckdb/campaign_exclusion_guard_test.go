package duckdb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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
		// Ajouts 2026-07-18 (fuite d'affichage Campagne H5, item backlog H1) :
		{"Q5SharedHistory (LISTE historique + compteur)", Q5SharedHistory},
		{"Q4SharedMatchesForFilters (cascade filtres, v_match_full)", Q4SharedMatchesForFilters},
		{"Q4MVSharedMatchesForFilters (cascade filtres, MV)", Q4MVSharedMatchesForFilters},
		{"Q26CareerTopEncountersTpl (career encounters)", Q26CareerTopEncountersTpl},
		{"Q28RelationsTpl (hub Relations)", Q28RelationsTpl},
		{"Q28RelationsScopedTpl (hub Relations scopé)", Q28RelationsScopedTpl},
		{"QRelationsCoreFormTpl (forme récente noyau dur)", QRelationsCoreFormTpl},
	}
	for _, c := range cases {
		if !strings.Contains(c.query, campaignExclusionToken) {
			t.Errorf("%s : token d'exclusion Campagne absent — la surface fuiterait les matchs Campagne (Halo 5)", c.name)
		}
	}
}

// TestCampaignExclusionStructuralCoverage — GARDE-RAIL STRUCTUREL (durcissement
// item H1, 2026-07-18). La whitelist ci-dessus était un angle mort : elle
// n'ÉNUMÈRE pas les lecteurs, donc un nouveau reader per-player oublié fuitait
// silencieusement (cause de cette régression : Q5SharedHistory + Q4/Q4MV +
// relations/career jamais ajoutés à la liste). Ce test balaye par AST TOUTES les
// constantes de requête `Q…` du package : toute requête qui liste/agrège les
// matchs d'un joueur (source per-player match_participants OU mv_player_matches,
// filtrée par `xuid = ?`) DOIT porter le token campagne — sauf exception
// explicitement JUSTIFIÉE dans allowlistedNoToken. Une nouvelle requête de ce
// type sans exclusion fait échouer ce test.
func TestCampaignExclusionStructuralCoverage(t *testing.T) {
	// Exceptions justifiées : constantes per-player SANS token DANS la constante.
	allowlistedNoToken := map[string]string{
		"Q25MatchParticipants": "sur-lecture inoffensive : les participants sont joints EN GO au set de matchs de Q23 (déjà purgé de la Campagne) ; les lignes de matchs Campagne surnuméraires ne sont jamais restituées.",
		"QRelationsPlayerWinRateTpl": "filtré au CALL SITE via excludeCampaignByMatchID (relations_core_engagement_repo.go, queryPlayerWinRate) — la clause vit hors de la constante.",
		"Q25NeighborMatchesTemplate": "filtré au CALL SITE : la clause Campagne est injectée dans /*EXTRA_WHERE*/ via excludeCampaignClause (match_view_repo_neighbors_skill.go).",
		"Q17PlayerMatchStats": "requête MONO-MATCH (WHERE match_id = ? AND xuid = ?) : aucune agrégation d'historique → la Campagne ne peut pas polluer un affichage de liste.",
		"Q17bIsParticipant":   "check de participation à UN match (EXISTS WHERE match_id = ? AND xuid = ?) — pas de listing ni d'agrégat.",
		"Q26MatchExpectedStats": "requête MONO-MATCH (WHERE match_id = ? AND xuid = ?) — pas d'agrégation d'historique.",
	}
	isPerPlayerMatchReader := func(text string) bool {
		hasSource := strings.Contains(text, "match_participants") || strings.Contains(text, "mv_player_matches")
		return hasSource && strings.Contains(text, "xuid = ?")
	}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("read %s: %v", name, rerr)
		}
		f, perr := parser.ParseFile(fset, name, src, 0)
		if perr != nil {
			t.Fatalf("parse %s: %v", name, perr)
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || (gd.Tok != token.VAR && gd.Tok != token.CONST) {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || len(vs.Values) == 0 {
					continue
				}
				qname := vs.Names[0].Name
				if !strings.HasPrefix(qname, "Q") {
					continue // convention : les constantes de requête sont Q<...>
				}
				start := fset.Position(vs.Values[0].Pos()).Offset
				end := fset.Position(vs.Values[len(vs.Values)-1].End()).Offset
				text := string(src[start:end])
				if !isPerPlayerMatchReader(text) {
					continue
				}
				scanned++
				if strings.Contains(text, "campaignExclusionToken") {
					continue
				}
				if _, ok := allowlistedNoToken[qname]; ok {
					continue
				}
				t.Errorf("%s (%s) : requête per-player (match_participants/mv_player_matches + `xuid = ?`) "+
					"SANS token d'exclusion Campagne ni entrée allowlist — fuite Campagne (Halo 5) potentielle. "+
					"Ajouter campaignExclusionToken + résoudre au call site, OU justifier dans allowlistedNoToken.", qname, name)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucune requête per-player scannée — le garde-rail structurel ne couvre rien (régression du scan ?)")
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
