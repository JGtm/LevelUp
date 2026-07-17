package analysis

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCampaignExclusionSingleQuotingPath (F6, revue 2026-07-17) : garde-rail
// anti-re-triplication. Le quoting SQL des GUID (idiome ReplaceAll("'","”")) ne
// doit apparaître QU'UNE fois dans campaign_exclusion.go — dans quotedIDList, seul
// point de quoting. Les 3 fonctions d'exclusion passent par quotedIDList /
// sqlExcludeByMatchIDSubquery ; ré-inliner la boucle casse ce test.
func TestCampaignExclusionSingleQuotingPath(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	src := filepath.Join(filepath.Dir(thisFile), "campaign_exclusion.go")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("lecture %s: %v", src, err)
	}
	if n := strings.Count(string(data), `ReplaceAll(id, "'", "''")`); n != 1 {
		t.Errorf("quoting SQL des GUID attendu 1× (dans quotedIDList), trouvé %d× — "+
			"ne pas ré-inliner la boucle de quoting (F6)", n)
	}
}

// TestSQLExcludeCampaignVariants : la clause littérale n'est émise QUE pour les
// titres ayant des game_variant Campagne (Halo 5), avec le bon alias, les 2 GUID
// inlinés comme littéraux (aucun placeholder), et vide (no-op) sinon (Infinite).
func TestSQLExcludeCampaignVariants(t *testing.T) {
	// Titre sans mode masqué → no-op.
	if got := SQLExcludeCampaignVariants("halo_infinite", "r"); got != "" {
		t.Errorf("halo_infinite: attendu \"\", obtenu %q", got)
	}
	if got := SQLExcludeCampaignVariants("", "r"); got != "" {
		t.Errorf("titre vide: attendu \"\", obtenu %q", got)
	}

	// Halo 5 → clause NOT IN littérale avec les 2 GUID + alias injecté.
	clause := SQLExcludeCampaignVariants("halo_5", "r")
	if !strings.Contains(clause, "r.game_variant_id") {
		t.Errorf("alias non injecté: %q", clause)
	}
	if !strings.Contains(clause, "NOT IN") {
		t.Errorf("clause NOT IN attendue: %q", clause)
	}
	// Aucun placeholder — les GUID sont des littéraux (pas d'arg à aligner).
	if strings.Contains(clause, "?") {
		t.Errorf("la clause littérale ne doit contenir aucun placeholder: %q", clause)
	}
	for _, id := range CampaignExcludedVariantIDs("halo_5") {
		if !strings.Contains(clause, "'"+id+"'") {
			t.Errorf("GUID %q non inliné comme littéral dans %q", id, clause)
		}
	}

	// Alias paramétrable.
	if c2 := SQLExcludeCampaignVariants("halo_5", "mr"); !strings.Contains(c2, "mr.game_variant_id") {
		t.Errorf("alias 'mr' non injecté: %q", c2)
	}
}

// TestSQLExcludeCampaignByMatchID : forme sous-requête (participants-only), title-aware.
func TestSQLExcludeCampaignByMatchID(t *testing.T) {
	if got := SQLExcludeCampaignByMatchID("halo_infinite", "mp.match_id"); got != "" {
		t.Errorf("halo_infinite: attendu \"\", obtenu %q", got)
	}
	clause := SQLExcludeCampaignByMatchID("halo_5", "mp.match_id")
	if !strings.Contains(clause, "mp.match_id NOT IN (SELECT match_id FROM match_registry") {
		t.Errorf("forme sous-requête attendue: %q", clause)
	}
	if strings.Contains(clause, "?") {
		t.Errorf("aucun placeholder attendu (GUID littéraux): %q", clause)
	}
	for _, id := range CampaignExcludedVariantIDs("halo_5") {
		if !strings.Contains(clause, "'"+id+"'") {
			t.Errorf("GUID %q absent: %q", id, clause)
		}
	}
}

// TestSQLExcludeAllCampaignByMatchID : forme TITLE-AGNOSTIC (tous titres), pour les
// lecteurs sans contexte de titre. Contient les GUID Campagne connus, no placeholder.
func TestSQLExcludeAllCampaignByMatchID(t *testing.T) {
	clause := SQLExcludeAllCampaignByMatchID("mp.match_id")
	if !strings.Contains(clause, "mp.match_id NOT IN (SELECT match_id FROM match_registry") {
		t.Errorf("forme sous-requête attendue: %q", clause)
	}
	if strings.Contains(clause, "?") {
		t.Errorf("aucun placeholder attendu: %q", clause)
	}
	for _, id := range CampaignExcludedVariantIDs("halo_5") {
		if !strings.Contains(clause, "'"+id+"'") {
			t.Errorf("GUID Campagne %q absent de la variante title-agnostic: %q", id, clause)
		}
	}
}

// TestCampaignExcludedVariantIDs : accès à la source unique, copie défensive.
func TestCampaignExcludedVariantIDs(t *testing.T) {
	ids := CampaignExcludedVariantIDs("halo_5")
	if len(ids) != 2 {
		t.Fatalf("halo_5: attendu 2 GUID Campagne, obtenu %d", len(ids))
	}
	if CampaignExcludedVariantIDs("halo_infinite") != nil {
		t.Errorf("halo_infinite: attendu nil (aucun mode masqué)")
	}
	// Mutation du retour ne doit pas corrompre la source.
	ids[0] = "MUTATED"
	if CampaignExcludedVariantIDs("halo_5")[0] == "MUTATED" {
		t.Errorf("CampaignExcludedVariantIDs doit retourner une copie défensive")
	}
}

// TestSQLResolveCampaignExclusion : remplacement du token par la clause (Halo 5)
// ou par vide (Infinite / titre inconnu). Le token survit à un fmt.Sprintf
// préalable (aucun %).
func TestSQLResolveCampaignExclusion(t *testing.T) {
	query := "SELECT match_id FROM match_participants mp JOIN match_registry r ON r.match_id = mp.match_id WHERE mp.xuid = ? " +
		CampaignExclusionToken + " ORDER BY 1"

	// Halo 5 : token remplacé par la clause NOT IN.
	h5 := SQLResolveCampaignExclusion(query, "halo_5", "r")
	if strings.Contains(h5, CampaignExclusionToken) {
		t.Errorf("token non résolu pour halo_5: %q", h5)
	}
	if !strings.Contains(h5, "r.game_variant_id") || !strings.Contains(h5, "NOT IN") {
		t.Errorf("clause absente après résolution halo_5: %q", h5)
	}

	// Infinite : token remplacé par vide (no-op) — la requête reste valide.
	hinf := SQLResolveCampaignExclusion(query, "halo_infinite", "r")
	if strings.Contains(hinf, CampaignExclusionToken) {
		t.Errorf("token non retiré pour halo_infinite: %q", hinf)
	}
	if strings.Contains(hinf, "game_variant_id") {
		t.Errorf("aucune clause ne doit être injectée pour halo_infinite: %q", hinf)
	}

	// Le token ne contient aucun % → survit à fmt.Sprintf.
	if strings.ContainsAny(CampaignExclusionToken, "%") {
		t.Errorf("le token ne doit contenir aucun %% (survie à fmt.Sprintf): %q", CampaignExclusionToken)
	}
}
