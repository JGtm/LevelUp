package ctxkeys

import (
	"context"
	"testing"
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
