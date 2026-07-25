//go:build cgo

// Package prestige — prestige_social_cross_title_test.go : régression « 500 sur
// GET /prestige/me sans title_slug pour un joueur sans PP » (V721-14a / D-05).
//
// GetUserPrestigeCrossTitle agrège `SELECT COALESCE(SUM(total_pp), 0),
// MAX(updated_at) FROM user_prestige_latest WHERE user_id = ?`. Un agrégat sans
// GROUP BY rend TOUJOURS une ligne — mais pour un joueur qui n'a aucune ligne de
// prestige, MAX(updated_at) vaut NULL. Le scan direct dans un `time.Time`
// échouait alors (« unsupported Scan, storing driver.Value type <nil> into type
// *time.Time »), et PrestigeHandler.serviceError ne reconnaissant aucune
// sentinelle, la branche `default` rendait un 500 masqué.
//
// Oracle : joueur inconnu → prestige VIDE sans erreur (même contrat que
// GetUserPrestige, qui traite déjà sql.ErrNoRows comme « prestige vide »).
// Le code pré-fix FAIL sur ce test ; le code post-fix PASS.
package prestige

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/prestige"
)

// TestGetUserPrestigeCrossTitle_NoPP_ReturnsEmptyNotError : le cas NULL.
func TestGetUserPrestigeCrossTitle_NoPP_ReturnsEmptyNotError(t *testing.T) {
	db := newPrestigeSocialDB(t)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()

	up, err := repo.GetUserPrestigeCrossTitle(ctx, "u_sans_aucun_pp")
	if err != nil {
		t.Fatalf("GetUserPrestigeCrossTitle(joueur sans PP) : erreur %v — "+
			"MAX(updated_at) NULL doit rendre un prestige vide, pas un 500", err)
	}
	if up.UserID != "u_sans_aucun_pp" {
		t.Errorf("user_id = %q, want u_sans_aucun_pp", up.UserID)
	}
	if up.TotalPP != 0 {
		t.Errorf("total_pp = %d, want 0", up.TotalPP)
	}
	if !up.UpdatedAt.IsZero() {
		t.Errorf("updated_at = %v, want zéro (aucune ligne de prestige)", up.UpdatedAt)
	}
}

// TestGetUserPrestigeCrossTitle_WithPP_KeepsUpdatedAt : contrôle négatif — le
// correctif NULL ne doit pas écraser la date quand elle existe.
func TestGetUserPrestigeCrossTitle_WithPP_KeepsUpdatedAt(t *testing.T) {
	db := newPrestigeSocialDB(t)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.EmitEvent(ctx, prestige.PrestigeEvent{
		ID: "pe_cross", UserID: "u_avec_pp", TitleSlug: "halo_infinite",
		SourceType: prestige.SourceChallenge, SourceID: "ch_x",
		PPAmount: 90, Tier: prestige.TierHeroic, CreatedAt: now,
	}); err != nil {
		t.Fatalf("EmitEvent: %v", err)
	}

	up, err := repo.GetUserPrestigeCrossTitle(ctx, "u_avec_pp")
	if err != nil {
		t.Fatalf("GetUserPrestigeCrossTitle: %v", err)
	}
	if up.TotalPP != 90 {
		t.Errorf("total_pp = %d, want 90", up.TotalPP)
	}
	if up.UpdatedAt.IsZero() {
		t.Error("updated_at zéro alors qu'un événement PP existe — le correctif NULL a mangé la date")
	}
}
