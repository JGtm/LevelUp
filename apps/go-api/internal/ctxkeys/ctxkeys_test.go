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

	gotXUID := HaloXUID(ctx)
	if gotXUID != "xuid-1234" {
		t.Errorf("expected xuid-1234, got %q", gotXUID)
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
	got := HaloTokens(ctx)
	if got != nil {
		t.Errorf("expected nil tokens when nil passed, got %+v", got)
	}
	if HaloXUID(ctx) != "xuid-999" {
		t.Errorf("expected xuid-999")
	}
}
