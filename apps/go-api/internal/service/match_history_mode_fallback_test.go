package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestEnrichRow_ModeFallbackToGameVariant : sur la voie liste Explorer/Historique, le
// mode se dérive du pair_name SINON du game_variant. Halo 5 n'a pas de pair_name → son
// mode (game_variant) doit remonter (sinon « aucun mode renseigné », signalement #3).
func TestEnrichRow_ModeFallbackToGameVariant(t *testing.T) {
	assassin := "Assassin"
	// H5 : pas de pair, game_variant FR présent → mode résolu (non vide).
	h5 := domain.MatchHistoryRawRow{MatchID: "m1", GameVariantNameFR: &assassin}
	if got := enrichRow(h5, nil, rowFormatters{}); got.ModeUI == nil || *got.ModeUI == "" {
		t.Errorf("H5: mode vide alors que game_variant présent (ModeUI=%v)", got.ModeUI)
	}

	// Infinite : pair_name présent → c'est lui qui prime (game_variant ignoré).
	pair, variant := "Strongholds", "Slayer"
	inf := domain.MatchHistoryRawRow{MatchID: "m2", PairName: &pair, GameVariantName: &variant}
	got := enrichRow(inf, nil, rowFormatters{})
	if got.ModeUI == nil || *got.ModeUI != "Strongholds" {
		t.Errorf("Infinite: mode = %v, attendu Strongholds (depuis pair, pas game_variant)", got.ModeUI)
	}

	// Aucune source (ni pair ni game_variant) → ModeUI nil (dégradation propre).
	empty := domain.MatchHistoryRawRow{MatchID: "m3"}
	if got := enrichRow(empty, nil, rowFormatters{}); got.ModeUI != nil {
		t.Errorf("sans source de mode, ModeUI attendu nil, obtenu %v", *got.ModeUI)
	}
}

// TestFilterByExplorerModeNames_GameVariantFallback : la voie Explorer (filtre
// modes) matche sur le game_variant quand le pair est absent (H5) ; le pair prime
// pour les rows Infinite (non-régression).
func TestFilterByExplorerModeNames_GameVariantFallback(t *testing.T) {
	assassin := "Assassin"
	strongholds := "Strongholds"
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "h5", GameVariantNameFR: &assassin}, // pair absent → variant
		{MatchID: "inf", PairName: &strongholds},      // pair présent
	}

	got := filterByExplorerModeNames(rows, []string{"Assassin"})
	if len(got) != 1 || got[0].MatchID != "h5" {
		t.Errorf("variant fallback: attendu h5, obtenu %v", got)
	}

	gotInf := filterByExplorerModeNames(rows, []string{"Strongholds"})
	if len(gotInf) != 1 || gotInf[0].MatchID != "inf" {
		t.Errorf("pair Infinite: attendu inf, obtenu %v", gotInf)
	}
}

// TestComputeExplorerAvailableOptions_GameVariantMode : les facettes modes
// incluent le game_variant des rows sans pair (H5) — sinon « Modes » resterait vide.
func TestComputeExplorerAvailableOptions_GameVariantMode(t *testing.T) {
	assassin := "Assassin"
	strongholds := "Strongholds"
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "h5", GameVariantNameFR: &assassin},
		{MatchID: "inf", PairName: &strongholds},
	}

	_, _, _, modes := computeExplorerAvailableOptions(rows)
	hasMode := func(want string) bool {
		for _, m := range modes {
			if m == want {
				return true
			}
		}
		return false
	}
	if !hasMode("Assassin") {
		t.Errorf("facette modes doit inclure 'Assassin' (game_variant H5), obtenu %v", modes)
	}
	if !hasMode("Strongholds") {
		t.Errorf("facette modes doit inclure 'Strongholds' (pair Infinite), obtenu %v", modes)
	}
}
