package skill

// skill_v2_capability_test.go — Sprint 3.C : tests du gate capability CapLUSR.

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

func TestSlugHasLUSR(t *testing.T) {
	cases := []struct {
		name string
		slug string
		want bool
	}{
		{"empty_defaults_to_halo_infinite", "", true},
		{"halo_infinite", "halo_infinite", true},
		{"unknown_title", "some_future_title", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SlugHasLUSR(c.slug); got != c.want {
				t.Errorf("SlugHasLUSR(%q) = %v, want %v", c.slug, got, c.want)
			}
		})
	}
}

func TestTitleHasLUSR(t *testing.T) {
	// Contexte par défaut → ctxkeys.TitleSlug = "halo_infinite" → CapLUSR.
	if !titleHasLUSR(context.Background()) {
		t.Error("titleHasLUSR(background) = false, want true (halo_infinite déclare CapLUSR)")
	}
	// Titre inconnu posé dans le contexte → pas de capability.
	ctx := ctxkeys.WithTitleSlug(context.Background(), "some_future_title")
	if titleHasLUSR(ctx) {
		t.Error("titleHasLUSR(future_title) = true, want false")
	}
}

func TestSkipIfNoLUSRCapability(t *testing.T) {
	if skipIfNoLUSRCapability(context.Background(), "test") {
		t.Error("skipIfNoLUSRCapability = true pour halo_infinite, want false")
	}
	ctx := ctxkeys.WithTitleSlug(context.Background(), "nope")
	if !skipIfNoLUSRCapability(ctx, "test") {
		t.Error("skipIfNoLUSRCapability = false pour titre inconnu, want true")
	}
}

// TestRunLUSRV2Shadow_SkipsIfNoLUSRCapability : même avec le flag ENABLED, un
// titre sans CapLUSR fait un no-op (0, nil). Le gate capability passe AVANT le
// check sharedDB==nil, donc passer nil ne déclenche aucune erreur.
func TestRunLUSRV2Shadow_SkipsIfNoLUSRCapability(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	ctx := ctxkeys.WithTitleSlug(context.Background(), "some_future_title")

	var nilShared *sql.DB
	n, err := RunLUSRV2Shadow(ctx, nil, nilShared, "xuid-123")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow sans capability: err = %v, want nil (gate avant check sharedDB)", err)
	}
	if n != 0 {
		t.Errorf("processed = %d, want 0", n)
	}
}
