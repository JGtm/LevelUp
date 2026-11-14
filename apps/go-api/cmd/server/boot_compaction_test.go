//go:build cgo

// boot_compaction_test.go — compaction des passes au démarrage : gardée par la capacité du profil,
// déclenchée par la jauge, copie préalable obligatoire, échec non fatal. Base DuckDB réelle en
// fichier temporaire pour le chemin de production.
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// seedSharedWithPasses crée une base partagée migrée dont match_bomb_stats porte 6 lignes pour 3
// gardées (rapport 2, au-dessus de migration.SeuilPassesParMatch).
func seedSharedWithPasses(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open shared RW: %v", err)
	}
	defer db.Close() //nolint:errcheck // fixture
	if err := prepareMigrationRegistry(""); err != nil {
		t.Fatalf("prepareMigrationRegistry: %v", err)
	}
	if err := migration.RunForDB(db.SQLDb(), migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	for i, l := range []struct{ match, xuid string }{
		{"m1", "x1"}, {"m1", "x1"}, {"m1", "x2"}, {"m2", "x1"}, {"m1", "x1"}, {"m2", "x1"},
	} {
		if _, err := db.SQLDb().Exec(`INSERT INTO match_bomb_stats (match_id, xuid, bomb_arms, written_at)
			VALUES (?, ?, 1, TIMESTAMP '2026-01-01 00:00:00' + to_seconds(?))`, l.match, l.xuid, i); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func lignesBombStats(t *testing.T, path string) int {
	t.Helper()
	db, release, err := duckdb.OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("open read: %v", err)
	}
	defer release()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_bomb_stats`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Au-dessus du seuil : copie préalable puis compaction (6 lignes brutes -> 3), sur la base réelle.
func TestCompactPassesAtBoot_AuDessusDuSeuilCompacteApresCopie(t *testing.T) {
	racine := t.TempDir()
	path := filepath.Join(racine, "warehouse", "shared_matches_v2.duckdb")
	seedSharedWithPasses(t, path)
	copies := filepath.Join(racine, "copies")

	if err := compactPassesAtBoot(context.Background(), newCompactBootDeps(migration.DefaultSlug, path, copies)); err != nil {
		t.Fatalf("compactPassesAtBoot: %v", err)
	}
	if n := lignesBombStats(t, path); n != 3 {
		t.Errorf("match_bomb_stats = %d lignes, attendu 3 après compaction", n)
	}
	entrees, err := os.ReadDir(copies)
	if err != nil || len(entrees) == 0 {
		t.Fatalf("aucune copie préalable dans %s (err=%v)", copies, err)
	}
}

// Sous le seuil (rien à retirer) : ni copie ni compaction.
func TestCompactPassesAtBoot_SousLeSeuilNeFaitRien(t *testing.T) {
	deps := compactBootDeps{
		slug: "s", path: "p", now: time.Now,
		measure: func(context.Context, string, string) ([]migration.JaugePasses, int, error) {
			return []migration.JaugePasses{{Table: "match_bomb_stats", Brutes: 10, Gardees: 10}}, 1, nil
		},
		backup: func(context.Context, migration.BackupParams) error {
			t.Error("copie lancée sous le seuil")
			return nil
		},
		compact: func(context.Context, string) ([]migration.CompactionTable, error) {
			t.Error("compaction lancée sous le seuil")
			return nil, nil
		},
	}
	if err := compactPassesAtBoot(context.Background(), deps); err != nil {
		t.Fatalf("compactPassesAtBoot: %v", err)
	}
}

// Une copie préalable refusée interdit la compaction.
func TestCompactPassesAtBoot_CopieRefuseePasDeCompaction(t *testing.T) {
	deps := compactBootDeps{
		slug: "s", path: "p", now: time.Now,
		measure: func(context.Context, string, string) ([]migration.JaugePasses, int, error) {
			return []migration.JaugePasses{{Table: "match_bomb_stats", Brutes: 20, Gardees: 10}}, 1, nil
		},
		backup: func(context.Context, migration.BackupParams) error { return migration.ErrInsufficientBackupSpace },
		compact: func(context.Context, string) ([]migration.CompactionTable, error) {
			t.Error("compaction lancée sans copie préalable")
			return nil, nil
		},
	}
	err := compactPassesAtBoot(context.Background(), deps)
	if !errors.Is(err, migration.ErrInsufficientBackupSpace) {
		t.Fatalf("err = %v, attendu ErrInsufficientBackupSpace", err)
	}
}

// Seul le profil qui porte la capacité compacte ; un titre sans base partagée est sauté ; un échec
// n'arrête rien (le démarrage continue, les titres suivants sont traités).
func TestCompactPassesAtBootAllTitles_GardeParLaCapacite(t *testing.T) {
	racine := t.TempDir()
	pr := title.NewPathResolver(racine)
	reg := title.DefaultRegistry()
	for _, d := range reg.All() {
		chemin := pr.SharedDBPath(d.Slug)
		if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(chemin, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		nom      string
		capacite bool
		appel    bool
	}{
		{"desktop", true, true},
		{"heberge", false, false},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			var vus []string
			prev := runCompactBoot
			runCompactBoot = func(_ context.Context, d compactBootDeps) error {
				vus = append(vus, d.slug)
				return errors.New("échec simulé")
			}
			t.Cleanup(func() { runCompactBoot = prev })

			var cfg config.AppConfig
			cfg.Capabilities.CompactPassesAtStartup = tc.capacite
			compactPassesAtBootAllTitles(context.Background(), &cfg, pr, reg)

			if tc.appel && len(vus) != len(reg.All()) {
				t.Errorf("titres traités = %v, attendu les %d titres du registre malgré l'échec simulé", vus, len(reg.All()))
			}
			if !tc.appel && len(vus) != 0 {
				t.Errorf("compaction lancée hors du profil qui la porte : %s", strings.Join(vus, ","))
			}
		})
	}
}
