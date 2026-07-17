package ctxkeys

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestWithTitleSlug_RoundTrip(t *testing.T) {
	ctx := WithTitleSlug(context.Background(), "halo_mcc")
	got := TitleSlug(ctx)
	if got != "halo_mcc" {
		t.Errorf("expected halo_mcc, got %s", got)
	}
}

func TestTitleSlug_DefaultEmpty(t *testing.T) {
	got := TitleSlug(context.Background())
	if got != "halo_infinite" {
		t.Errorf("expected halo_infinite default, got %s", got)
	}
}

func TestTitleSlug_EmptyString(t *testing.T) {
	ctx := WithTitleSlug(context.Background(), "")
	got := TitleSlug(ctx)
	if got != "halo_infinite" {
		t.Errorf("expected default for empty string, got %s", got)
	}
}

func TestWithHaloAuth_RoundTrip(t *testing.T) {
	tokens := &domain.HaloTokens{SpartanToken: "spartan_abc", ClearanceToken: "clear_xyz"}
	ctx := WithHaloAuth(context.Background(), tokens, "xuid-1234")

	gotTokens := HaloTokens(ctx)
	if gotTokens == nil {
		t.Fatal("expected non-nil tokens")
	}
	if gotTokens.SpartanToken != "spartan_abc" {
		t.Errorf("expected spartan_abc, got %q", gotTokens.SpartanToken)
	}
	if gotTokens.ClearanceToken != "clear_xyz" {
		t.Errorf("expected clear_xyz, got %q", gotTokens.ClearanceToken)
	}
	if HaloXUID(ctx) != "xuid-1234" {
		t.Errorf("expected xuid-1234, got %q", HaloXUID(ctx))
	}
}

func TestHaloTokens_AbsentReturnsNil(t *testing.T) {
	got := HaloTokens(context.Background())
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestHaloXUID_AbsentReturnsEmpty(t *testing.T) {
	got := HaloXUID(context.Background())
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestWithHaloAuth_NilTokens(t *testing.T) {
	ctx := WithHaloAuth(context.Background(), nil, "xuid-999")
	if HaloTokens(ctx) != nil {
		t.Errorf("expected nil tokens when nil passed")
	}
	if HaloXUID(ctx) != "xuid-999" {
		t.Errorf("expected xuid-999")
	}
}

// TestWithHaloAuth_SetsTokensOwner : WithHaloAuth pose le porteur (tokensOwnerXUID)
// en même temps que le sujet (finding ID3).
func TestWithHaloAuth_SetsTokensOwner(t *testing.T) {
	ctx := WithHaloAuth(context.Background(), &domain.HaloTokens{SpartanToken: "s"}, "owner-1")
	if got := TokensOwnerXUID(ctx); got != "owner-1" {
		t.Errorf("expected owner-1, got %q", got)
	}
}

func TestTokensOwnerXUID_AbsentReturnsEmpty(t *testing.T) {
	if got := TokensOwnerXUID(context.Background()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// TestForcePageSubject_PreservesTokensOwner verrouille l'invariant central du
// finding ID3 : forcer le SUJET vers le joueur de la page (WithHaloXUID) ne doit
// JAMAIS déplacer le PORTEUR des tokens — sinon le budget API du compte connecté
// serait débité sur le mauvais bucket.
func TestForcePageSubject_PreservesTokensOwner(t *testing.T) {
	// Compte connecté "owner" porte les tokens.
	ctx := WithHaloAuth(context.Background(), &domain.HaloTokens{SpartanToken: "s"}, "connected-owner")
	// Forçage du sujet vers la page (autre joueur consulté).
	ctx = WithHaloXUID(ctx, "page-subject")

	if got := HaloXUID(ctx); got != "page-subject" {
		t.Errorf("subject: expected page-subject, got %q", got)
	}
	if got := TokensOwnerXUID(ctx); got != "connected-owner" {
		t.Errorf("owner: expected connected-owner (préservé), got %q", got)
	}
}

// TestWithTokensOwnerXUID_OverridesOwnerNotSubject : la correction du porteur
// (path background carrière) n'altère pas le sujet.
func TestWithTokensOwnerXUID_OverridesOwnerNotSubject(t *testing.T) {
	ctx := WithHaloAuth(context.Background(), &domain.HaloTokens{SpartanToken: "s"}, "subject-x")
	ctx = WithTokensOwnerXUID(ctx, "real-owner-y")
	if got := HaloXUID(ctx); got != "subject-x" {
		t.Errorf("subject should stay subject-x, got %q", got)
	}
	if got := TokensOwnerXUID(ctx); got != "real-owner-y" {
		t.Errorf("owner should be real-owner-y, got %q", got)
	}
}
