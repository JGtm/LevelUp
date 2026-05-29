package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestSquadModeResolution_ResolvesUUIDViaAssetAndModeNameTr : un pair_name brut
// = UUID doit être résolu en libellé lisible via la cascade canonique
// (asset_translations[pair_id] → mode_name_tr), comme l'historique des matchs.
// Régression : la colonne "Mode" Escouade affichait l'UUID brut.
func TestSquadModeResolution_ResolvesUUIDViaAssetAndModeNameTr(t *testing.T) {
	const pairUUID = "b08a915b-c862-4ef2-9247-2c3865919a6f"
	rows := []domain.SquadMatchRow{
		{
			MatchID:  "m1",
			MapUI:    "Aquarius",
			PairName: pairUUID, // EN brut non traduit (UUID)
			PairID:   "pair-1",
		},
	}
	repo := &mockSquadRepo{
		assetFR: map[string]map[string]string{
			"pair": {"pair-1": "Arena:Slayer on Aquarius"},
		},
		modeFR: map[string]string{"Slayer": "Assassin"},
	}

	enrichSquadMatchAssets(context.Background(), repo, rows)
	hist := buildSquadMatchHistory(rows, nil)

	if len(hist) != 1 {
		t.Fatalf("want 1 row, got %d", len(hist))
	}
	if hist[0].ModeUI != "Assassin" {
		t.Errorf("ModeUI = %q, want \"Assassin\" (résolu via asset + mode_name_tr)", hist[0].ModeUI)
	}
	// PairName (fallback front) ne doit jamais rester l'UUID brut.
	if hist[0].PairName == pairUUID {
		t.Errorf("PairName ne doit pas rester l'UUID brut, got %q", hist[0].PairName)
	}
}

// TestSquadModeResolution_GuardsResidualUUID : si aucune traduction n'est
// disponible (trou de metadata) et que pair_name reste un UUID, la colonne Mode
// doit dégrader proprement (vide → le front affiche "-"), jamais l'UUID.
func TestSquadModeResolution_GuardsResidualUUID(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{
			MatchID:  "m1",
			MapUI:    "Aquarius",
			PairName: "b08a915b-c862-4ef2-9247-2c3865919a6f",
			PairID:   "pair-unknown",
		},
	}
	repo := &mockSquadRepo{} // aucune trad

	enrichSquadMatchAssets(context.Background(), repo, rows)
	hist := buildSquadMatchHistory(rows, nil)

	if hist[0].ModeUI != "" {
		t.Errorf("ModeUI doit être vide (UUID masqué), got %q", hist[0].ModeUI)
	}
}

// TestSquadModeResolution_FallsBackToNormalizedEN : sans traduction FR mais avec
// un pair_name EN exploitable, le mode est normalisé en EN (sous-mode extrait),
// jamais le libellé technique complet.
func TestSquadModeResolution_FallsBackToNormalizedEN(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{
			MatchID:  "m1",
			MapUI:    "Streets",
			PairName: "Arena:CTF on Streets",
			PairID:   "pair-2",
		},
	}
	repo := &mockSquadRepo{} // aucune trad FR

	enrichSquadMatchAssets(context.Background(), repo, rows)
	hist := buildSquadMatchHistory(rows, nil)

	if hist[0].ModeUI != "CTF" {
		t.Errorf("ModeUI = %q, want \"CTF\" (EN normalisé)", hist[0].ModeUI)
	}
}
