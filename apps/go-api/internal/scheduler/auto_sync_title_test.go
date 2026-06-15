// auto_sync_title_test.go — oracle PMT-3 / MT-11 : BuildEngine route le moteur
// vers les DB du titre porté par le ctx (ctxkeys.TitleSlug).

package scheduler_test

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
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
