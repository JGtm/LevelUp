//go:build integration

// Package ops — seed_demo_synthetic_integration_test.go : E2E du générateur
// synthétique (DuckDB live, CGO). Vérifie la structure produite + le déterminisme
// (deux runs → mêmes données).
package ops

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setSyntheticProviders câble les providers de migration title-owned (parité
// cmd/levelup/main.go) — requis pour que RunForTitleDB(TargetMetadata) seede
// weapon_labels / mode_name_tr / career_rank_translations.
func setSyntheticProviders() {
	migration.SetCareerRankTranslationsProvider(halomigrations.CareerRankTranslations)
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
}

func TestSeedDemoSynthetic_Structure(t *testing.T) {
	setSyntheticProviders()
	out := t.TempDir()
	res, err := SeedDemoSynthetic(context.Background(), SyntheticDemoOptions{OutDir: out, ServiceTag: "DEMO"})
	if err != nil {
		t.Fatalf("SeedDemoSynthetic: %v", err)
	}
	if res.Matches != 60 {
		t.Errorf("Matches = %d, want 60", res.Matches)
	}
	if res.Players != 3 {
		t.Errorf("Players = %d, want 3", res.Players)
	}

	// Fichiers attendus (layout plat titre par défaut).
	for _, rel := range []string{
		"warehouse/metadata.duckdb", "warehouse/shared_matches_v2.duckdb",
		"warehouse/shared_social.duckdb", "players/DEMO/stats.duckdb",
		"players/DEMO2/stats.duckdb", "players/DEMO3/stats.duckdb",
		"db_profiles.json", "app_settings.json",
	} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("fichier attendu absent: %s: %v", rel, err)
		}
	}

	// Comptages shared.
	shared := openRO(t, filepath.Join(out, "warehouse", "shared_matches_v2.duckdb"))
	defer shared.Close()
	assertCount(t, shared, "SELECT COUNT(*) FROM match_registry", 60)
	assertCountAtLeast(t, shared, "SELECT COUNT(*) FROM match_participants", 200)
	assertCountAtLeast(t, shared, "SELECT COUNT(*) FROM weapon_kills", 100)
	assertCount(t, shared, "SELECT COUNT(*) FROM match_csrs", 22) // sessions classées (arène) = 10 + 12
	assertCountAtLeast(t, shared, "SELECT COUNT(*) FROM medals_earned", 1)

	// Comptages player principal + vues _latest (append-only OK).
	player := openRO(t, filepath.Join(out, "players", "DEMO", "stats.duckdb"))
	defer player.Close()
	assertCount(t, player, "SELECT COUNT(*) FROM player_match_enrichment_latest", 60)
	assertCount(t, player, "SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type='CSR'", 22)
	assertCount(t, player, "SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type='LUSR'", 38)
	assertCount(t, player, "SELECT COUNT(*) FROM career_progression WHERE rank_tier='Gold'", 1)
	assertCount(t, player, "SELECT COUNT(*) FROM sessions", 5)
	assertCountAtLeast(t, player, "SELECT COUNT(*) FROM match_citations_latest", 1)
	assertCountAtLeast(t, player, "SELECT COUNT(*) FROM player_csr_snapshots_latest", 1)

	// Metadata : référentiels seedés + synthétiques.
	meta := openRO(t, filepath.Join(out, "warehouse", "metadata.duckdb"))
	defer meta.Close()
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM weapon_labels", 10)
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM medal_definitions", 4)
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM career_ranks", 4)
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM citation_mappings", 10)
}

// TestSeedDemoSynthetic_Deterministic : deux générations → données identiques
// (agrégats stables). DuckDB peut différer byte-à-byte ; on compare la donnée.
func TestSeedDemoSynthetic_Deterministic(t *testing.T) {
	setSyntheticProviders()
	sig := func() string {
		out := t.TempDir()
		if _, err := SeedDemoSynthetic(context.Background(), SyntheticDemoOptions{OutDir: out, ServiceTag: "DEMO"}); err != nil {
			t.Fatalf("SeedDemoSynthetic: %v", err)
		}
		db := openRO(t, filepath.Join(out, "warehouse", "shared_matches_v2.duckdb"))
		defer db.Close()
		var s string
		// Empreinte : somme des stats + concat ordonnée des match_ids/maps.
		if err := db.QueryRow(`
			SELECT CONCAT(
				(SELECT SUM(kills*1000 + deaths*10 + assists) FROM match_participants), '|',
				(SELECT STRING_AGG(match_id, ',' ORDER BY match_id) FROM match_registry), '|',
				(SELECT STRING_AGG(map_name || pair_name, ',' ORDER BY match_id) FROM match_registry)
			)`).Scan(&s); err != nil {
			t.Fatalf("signature: %v", err)
		}
		return s
	}
	a, b := sig(), sig()
	if a != b {
		t.Errorf("génération non déterministe:\n run1=%.120s\n run2=%.120s", a, b)
	}
}

// ── helpers ──

func openRO(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	return db
}

func assertCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if n != want {
		t.Errorf("%q = %d, want %d", query, n, want)
	}
}

func assertCountAtLeast(t *testing.T, db *sql.DB, query string, min int) {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if n < min {
		t.Errorf("%q = %d, want >= %d", query, n, min)
	}
}
