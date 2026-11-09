// auto_sync_title_test.go — oracle PMT-3 / MT-11 : BuildEngine route le moteur
// vers les DB du titre porté par le ctx (ctxkeys.TitleSlug).

package scheduler_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

func TestBuildEngine_TitleSlugFromCtx(t *testing.T) {
	s, _ := newFullyWiredScheduler(t)

	// (a) Parité : ctx sans titre → halo_infinite (byte-identique au legacy).
	eHalo := s.BuildEngine(context.Background(), "JGtm", "123")
	if got := eHalo.TitleSlugForTest(); got != "halo_infinite" {
		t.Errorf("ctx sans titre → titleSlug = %q, want halo_infinite", got)
	}

	// (b) Routing : ctx avec slug synthétique → moteur écrit dans les DB du titre.
	ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_test_title")
	eSyn := s.BuildEngine(ctx, "JGtm", "123")
	if got := eSyn.TitleSlugForTest(); got != "synthetic_test_title" {
		t.Errorf("titleSlug = %q, want synthetic_test_title", got)
	}
	if p := eSyn.PlayerDBPathForTest(); !strings.Contains(p, "synthetic_test_title") {
		t.Errorf("playerDBPath = %q ne route pas vers le titre synthétique", p)
	}
	if eSyn.PlayerDBPathForTest() == eHalo.PlayerDBPathForTest() {
		t.Errorf("playerDBPath synthetic == halo : routing non effectif (%q)", eSyn.PlayerDBPathForTest())
	}
}

// TestBuildEngine_SharedProviderIsTitleScoped : le titre par défaut (ou sans titre) reçoit le
// provider du boot ; un autre titre n'en reçoit JAMAIS le provider du boot (il écrirait et
// lirait la base partagée du mauvais titre). Sans Manager pour le servir, il reste sans provider.
func TestBuildEngine_SharedProviderIsTitleScoped(t *testing.T) {
	s, cfg := newFullyWiredScheduler(t)
	boot := sharedprovider.FromInMemoryDB((*sql.DB)(nil), ":memory:")
	cfg.SharedProvider = boot

	if got := s.BuildEngine(context.Background(), "JGtm", "123").SharedProviderForTest(); got != boot {
		t.Errorf("ctx sans titre : attendu le provider du boot, got %v", got)
	}
	ctxDefault := ctxkeys.WithTitleSlug(context.Background(), title.DefaultSlug)
	if got := s.BuildEngine(ctxDefault, "JGtm", "123").SharedProviderForTest(); got != boot {
		t.Errorf("titre par défaut : attendu le provider du boot, got %v", got)
	}

	ctxOther := ctxkeys.WithTitleSlug(context.Background(), "synthetic_test_title")
	eOther := s.BuildEngine(ctxOther, "JGtm", "123")
	if got := eOther.SharedProviderForTest(); got == boot {
		t.Fatal("autre titre : le provider du boot (titre par défaut) a été câblé sur le moteur")
	}
	if eOther.HasSharedProvider() {
		t.Error("autre titre sans Manager : le moteur doit rester sans provider (mode legacy)")
	}
}
