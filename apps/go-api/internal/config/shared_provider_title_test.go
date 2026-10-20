//go:build cgo

package config

import (
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestSharedProviderForTitle : le titre par défaut garde le provider du boot, un autre titre
// reçoit le SIEN (jamais celui du titre par défaut), et sans Manager un autre titre n'a pas de
// provider plutôt que celui d'un autre titre.
func TestSharedProviderForTitle(t *testing.T) {
	repoRoot := t.TempDir()
	hinfPath := makeSharedDBWithMatch(t, repoRoot, titlePkg.DefaultSlug, "")
	_ = makeSharedDBWithMatch(t, repoRoot, halo5.TitleSlug, "")

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()
	boot, err := mgr.For(hinfPath)
	if err != nil {
		t.Fatalf("For(hinf): %v", err)
	}
	cfg := &AppConfig{RepoRoot: repoRoot, SharedProvider: boot, SharedManager: mgr}

	if got := cfg.SharedProviderForTitle(titlePkg.DefaultSlug); got != boot {
		t.Errorf("titre par défaut : attendu le provider du boot")
	}
	if got := cfg.SharedProviderForTitle(""); got != boot {
		t.Errorf("titre vide : attendu le provider du boot")
	}
	h5 := cfg.SharedProviderForTitle(halo5.TitleSlug)
	if h5 == nil || h5 == boot {
		t.Fatalf("halo_5 : attendu un provider propre, distinct du boot (got %v)", h5)
	}
	if again := cfg.SharedProviderForTitle(halo5.TitleSlug); again != h5 {
		t.Errorf("halo_5 : le Manager doit rendre le même provider à chaque appel")
	}

	legacy := &AppConfig{RepoRoot: repoRoot, SharedProvider: boot}
	if got := legacy.SharedProviderForTitle(halo5.TitleSlug); got != nil {
		t.Errorf("sans Manager, halo_5 ne doit pas recevoir le provider d'un autre titre (got %v)", got)
	}
	if got := legacy.SharedProviderForTitle(titlePkg.DefaultSlug); got != boot {
		t.Errorf("sans Manager, le titre par défaut garde le provider du boot")
	}
}
