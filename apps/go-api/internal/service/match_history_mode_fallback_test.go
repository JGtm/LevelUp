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
