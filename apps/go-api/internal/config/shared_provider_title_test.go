//go:build cgo

package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb"
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

// makeSharedDBWithMatch crée le shared_matches_v2.duckdb au path title-aware du
// slug donné, avec une table match_registry minimale + une ligne matchID si
// non vide. Schéma minimal (pas de dépendance à internal/sync — cycle d'import).
func makeSharedDBWithMatch(t *testing.T, repoRoot, slug, matchID string) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).SharedDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir warehouse %s: %v", slug, err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite shared %s: %v", slug, err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if _, err := db.SQLDb().ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS match_registry (
			match_id   VARCHAR PRIMARY KEY,
			start_time TIMESTAMP NOT NULL
		)`); err != nil {
		t.Fatalf("create match_registry %s: %v", slug, err)
	}
	if matchID != "" {
		if _, err := db.SQLDb().ExecContext(ctx,
			"INSERT INTO match_registry (match_id, start_time) VALUES (?, CURRENT_TIMESTAMP)",
			matchID); err != nil {
			t.Fatalf("insert match %s: %v", slug, err)
		}
	}
	return path
}
