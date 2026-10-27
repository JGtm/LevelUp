package duckdb

import (
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

func variantScan(variantID, variantName, pairID, pairName, pairNameFR string) playerMatchScanResult {
	return playerMatchScanResult{
		matchID: "m1", variantID: variantID, variantName: i18n.FromColumns(variantName, ""),
		pairID: pairID, pairName: i18n.FromColumns(pairName, pairNameFR),
	}
}

func TestProjectMatchSummary_Mode(t *testing.T) {
	t.Run("type de jeu sans partie : le mode est le type de jeu", func(t *testing.T) {
		s := projectMatchSummary(variantScan("gv-1", "Slayer", "", "", ""), "", nil, title.ModeSourceGameVariant)
		if s.PairMode == nil || s.PairMode.ID != "gv-1" || s.PairMode.DefaultLabel != "Slayer" {
			t.Fatalf("PairMode = %+v, want le type de jeu gv-1/Slayer", s.PairMode)
		}
		if s.PairMode.Kind != "game_variant" {
			t.Errorf("Kind = %q, want game_variant (choisit le dictionnaire de traduction)", s.PairMode.Kind)
		}
		if s.GameVariant == nil || s.GameVariant.ID != "gv-1" {
			t.Errorf("GameVariant = %+v, doit rester exposé", s.GameVariant)
		}
	})
	t.Run("la copie est indépendante de GameVariant", func(t *testing.T) {
		s := projectMatchSummary(variantScan("gv-1", "Slayer", "", "", ""), "", nil, title.ModeSourceGameVariant)
		s.PairMode.Labels["fr"] = "Assassin"
		if _, ok := s.GameVariant.Labels["fr"]; ok {
			t.Error("écrire le libellé du mode a modifié GameVariant : la référence n'est pas copiée")
		}
	})
	t.Run("une partie présente garde la priorité", func(t *testing.T) {
		s := projectMatchSummary(variantScan("gv-1", "Slayer", "pair-1", "Arena:CTF", "Arène : CTF"), "", nil,
			title.ModeSourceGameVariant)
		if s.PairMode == nil || s.PairMode.ID != "pair-1" || s.PairMode.Kind != "pair_mode" {
			t.Fatalf("PairMode = %+v, want la partie pair-1", s.PairMode)
		}
	})
	t.Run("type de jeu sans nom : référence sans libellé, aucun identifiant affichable", func(t *testing.T) {
		s := projectMatchSummary(variantScan("gv-9", "", "", "", ""), "", nil, title.ModeSourceGameVariant)
		if s.PairMode == nil || s.PairMode.DefaultLabel != "" || len(s.PairMode.Labels) != 0 {
			t.Fatalf("PairMode = %+v, want l'identifiant seul, sans libellé", s.PairMode)
		}
	})
	t.Run("ni partie ni type de jeu : pas de mode", func(t *testing.T) {
		s := projectMatchSummary(variantScan("", "", "", "", ""), "", nil, title.ModeSourceGameVariant)
		if s.PairMode != nil {
			t.Fatalf("PairMode = %+v, want nil", s.PairMode)
		}
	})
	t.Run("source partie (Infinite) : le type de jeu ne devient jamais le mode", func(t *testing.T) {
		s := projectMatchSummary(variantScan("gv-1", "Slayer", "", "", ""), "", nil, title.ModeSourcePair)
		if s.PairMode != nil {
			t.Fatalf("PairMode = %+v, want nil", s.PairMode)
		}
		s = projectMatchSummary(variantScan("gv-1", "Slayer", "pair-1", "Arena:Slayer", ""), "", nil, title.ModeSourcePair)
		if s.PairMode == nil || s.PairMode.ID != "pair-1" {
			t.Fatalf("PairMode = %+v, want la partie", s.PairMode)
		}
	})
}

func TestModeAssetReference_NilVariant(t *testing.T) {
	if got := ModeAssetReference(title.ModeSourceGameVariant, nil, nil); got != nil {
		t.Errorf("got %+v, want nil", got)
	}
	pair := &canonical.AssetReference{ID: "p"}
	if got := ModeAssetReference(title.ModeSourceGameVariant, pair, nil); got != pair {
		t.Errorf("la partie doit être rendue telle quelle")
	}
}
