package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
)

func TestTeamSideToID(t *testing.T) {
	t.Parallel()
	t0, tbad, tempty := "t0", "x1", ""
	cases := []struct {
		in   *string
		want int
		ok   bool
	}{
		{&t0, 0, true},
		{&tbad, 0, false},
		{&tempty, 0, false},
		{nil, 0, false},
	}
	for _, c := range cases {
		got, ok := teamSideToID(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("teamSideToID(%v) = (%d,%v), want (%d,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestApplyTeamNames_SetsLocalizedName : l'adapter H5 (capability teamNameResolver)
// renseigne les libellés d'équipe localisés « Rouge/Bleu » sur le scoreboard, en
// laissant vides les team_side inconnus/absents (fallback front préservé).
func TestApplyTeamNames_SetsLocalizedName(t *testing.T) {
	t.Parallel()
	adapter := halo5.NewAssetURLAdapter().WithTeamNameResolver(func(id int, locale string) string {
		switch {
		case id == 0 && locale == "fr":
			return "Rouge"
		case id == 1 && locale == "fr":
			return "Bleu"
		default:
			return ""
		}
	})
	svc := NewMatchViewService(nil, "").WithAssetURL(adapter)

	t0, t1, t9 := "t0", "t1", "t9"
	rows := []domain.MatchScoreboardRow{
		{XUID: "a", TeamSide: &t0},
		{XUID: "b", TeamSide: &t1},
		{XUID: "c", TeamSide: &t9}, // team inconnu → "" → inchangé
		{XUID: "d", TeamSide: nil}, // pas de team_side → inchangé
	}
	svc.applyTeamNames(ctxkeys.WithLocale(context.Background(), "fr"), rows)

	if rows[0].TeamName != "Rouge" || rows[1].TeamName != "Bleu" {
		t.Errorf("noms localisés attendus Rouge/Bleu, got %q/%q", rows[0].TeamName, rows[1].TeamName)
	}
	if rows[2].TeamName != "" || rows[3].TeamName != "" {
		t.Errorf("team_name doit rester vide (team inconnu/absent), got %q/%q", rows[2].TeamName, rows[3].TeamName)
	}
}

// TestApplyTeamNames_SetsIdentityColor : l'adapter H5 (capability teamColorResolver)
// renseigne la couleur d'identité hex sur le scoreboard, en laissant vides les team_side
// inconnus/absents (fallback couleur front préservé).
func TestApplyTeamNames_SetsIdentityColor(t *testing.T) {
	t.Parallel()
	adapter := halo5.NewAssetURLAdapter().WithTeamColorResolver(func(id int) string {
		switch id {
		case 0:
			return "#b00000"
		case 1:
			return "#178dd8"
		default:
			return ""
		}
	})
	svc := NewMatchViewService(nil, "").WithAssetURL(adapter)

	t0, t1, t9 := "t0", "t1", "t9"
	rows := []domain.MatchScoreboardRow{
		{XUID: "a", TeamSide: &t0},
		{XUID: "b", TeamSide: &t1},
		{XUID: "c", TeamSide: &t9}, // team inconnu → "" → inchangé
		{XUID: "d", TeamSide: nil}, // pas de team_side → inchangé
	}
	svc.applyTeamNames(context.Background(), rows)

	if rows[0].TeamColor != "#b00000" || rows[1].TeamColor != "#178dd8" {
		t.Errorf("couleurs attendues #b00000/#178dd8, got %q/%q", rows[0].TeamColor, rows[1].TeamColor)
	}
	if rows[2].TeamColor != "" || rows[3].TeamColor != "" {
		t.Errorf("team_color doit rester vide (team inconnu/absent), got %q/%q", rows[2].TeamColor, rows[3].TeamColor)
	}
}

// TestApplyTeamNames_NoCapabilityIsNoOp : sans adapter exposant teamNameResolver
// (ex. Halo Infinite, ou assetURL nil), applyTeamNames est un no-op sans panique →
// team_name reste vide → le front garde sa résolution existante.
func TestApplyTeamNames_NoCapabilityIsNoOp(t *testing.T) {
	t.Parallel()
	svc := NewMatchViewService(nil, "") // assetURL nil → aucune capability
	t0 := "t0"
	rows := []domain.MatchScoreboardRow{{XUID: "a", TeamSide: &t0}}
	svc.applyTeamNames(context.Background(), rows)
	if rows[0].TeamName != "" {
		t.Errorf("team_name = %q, want \"\" (pas de capability)", rows[0].TeamName)
	}
}
