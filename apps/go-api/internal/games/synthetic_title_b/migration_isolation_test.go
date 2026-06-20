//go:build integration

package synthetic_title_b

// migration_isolation_test.go — PMT-9 oracle (b) : un titre non-défaut route SON
// propre jeu de migrations et JAMAIS celui de Halo, même quand le provider Halo
// est câblé (cas de production). Sans ce test, la factorisation title-aware
// resterait cosmétique.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// registerSyntheticSet enregistre un jeu de migrations minimal pour le titre B :
// 2 steps shared (create + alter) dans un ordre canonique propre, et AUCUN step
// pour les autres targets (dégradation no-op). Ne combine jamais le registre
// global Halo → isolation totale.
func registerSyntheticSet() {
	migration.RegisterMigrationSet(migration.TitleMigrationSet{
		Slug:           TitleSlug,
		CanonicalOrder: []string{"synthb_create_base_shared", "synthb_add_score_ms"},
		Steps: func(target migration.TargetDB) []migration.Migration {
			if target != migration.TargetShared {
				return nil // titre B ne couvre que shared → autres targets = no-op
			}
			return []migration.Migration{
				{
					Name:        "synthb_create_base_shared",
					TargetDB:    migration.TargetShared,
					Description: "titre B — table de base shared",
					ApplySchema: func(db *sql.DB) error {
						_, err := db.Exec("CREATE TABLE IF NOT EXISTS synthb_matches (match_id VARCHAR PRIMARY KEY)")
						return err
					},
				},
				{
					Name:        "synthb_add_score_ms",
					TargetDB:    migration.TargetShared,
					Description: "titre B — ajoute score_ms (stocké en ms, cf. fields.toml)",
					ApplySchema: func(db *sql.DB) error {
						_, err := db.Exec("ALTER TABLE synthb_matches ADD COLUMN IF NOT EXISTS score_ms BIGINT")
						return err
					},
				},
			}
		},
	})
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", name,
	).Scan(&n); err != nil {
		t.Fatalf("tableExists(%s): %v", name, err)
	}
	return n == 1
}

// TestSyntheticTitleB_MigrationIsolation : le titre B applique SES steps et jamais
// ceux de Halo, avec son propre title_schema_version — même provider Halo câblé.
func TestSyntheticTitleB_MigrationIsolation(t *testing.T) {
	// Provider Halo câblé comme en prod : prouve que le routage par set (map)
	// l'emporte et que le titre B n'hérite JAMAIS du registre global Halo.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	registerSyntheticSet()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migration.RunForTitleDB(db, TitleSlug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(%s, shared): %v", TitleSlug, err)
	}

	// 1. Les steps du titre B sont appliqués, dans l'ordre (la colonne prouve l'ALTER après le CREATE).
	if !tableExists(t, db, "synthb_matches") {
		t.Fatal("synthb_matches absente — le set du titre B n'a pas été appliqué")
	}
	var nCol int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'synthb_matches' AND column_name = 'score_ms'",
	).Scan(&nCol); err != nil {
		t.Fatalf("query score_ms: %v", err)
	}
	if nCol != 1 {
		t.Error("synthb_matches.score_ms absente — ordre create→alter du set non respecté")
	}

	// 2. AUCUNE table Halo : ni le god-file shared (match_registry), ni les vues.
	for _, haloTable := range []string{"match_registry", "match_participants", "medals_earned", "v_gamertag_lookup"} {
		if tableExists(t, db, haloTable) {
			t.Errorf("table Halo %q présente dans la DB du titre B — fuite du registre global (isolation cassée)", haloTable)
		}
	}

	// 3. Ledger title_schema_version : version du titre B = len de SON ordre (2), distincte de Halo.
	var version int
	if err := db.QueryRow(
		"SELECT version FROM title_schema_version WHERE title_slug = ? AND target = ?",
		TitleSlug, string(migration.TargetShared),
	).Scan(&version); err != nil {
		t.Fatalf("read title_schema_version(%s): %v", TitleSlug, err)
	}
	if version != 2 {
		t.Errorf("title_schema_version(%s, shared).version = %d, want 2", TitleSlug, version)
	}

	// 4. schema_migrations tracé sous le slug du titre B.
	var slug string
	if err := db.QueryRow(
		"SELECT title_slug FROM schema_migrations WHERE name = 'synthb_create_base_shared'",
	).Scan(&slug); err != nil {
		t.Fatalf("read title_slug: %v", err)
	}
	if slug != TitleSlug {
		t.Errorf("schema_migrations.title_slug = %q, want %q", slug, TitleSlug)
	}
}

// TestSyntheticTitleB_EmptyTargetNoOp : une target non couverte par le titre B
// dégrade en no-op propre (0 step, 0 table Halo, 0 erreur).
func TestSyntheticTitleB_EmptyTargetNoOp(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	registerSyntheticSet()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// metadata : le titre B n'a aucun step → no-op, surtout PAS les seeds Halo.
	if err := migration.RunForTitleDB(db, TitleSlug, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForTitleDB(%s, metadata) devrait être un no-op propre: %v", TitleSlug, err)
	}
	if tableExists(t, db, "asset_translations") {
		t.Error("asset_translations (Halo metadata) présente — fuite sur un target vide du titre B")
	}
	// La version est enregistrée même à 0 step (= len de l'ordre du set, 2 ici) : le
	// cycle a tourné proprement. On vérifie surtout l'absence de tables Halo.
	var nMig int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE title_slug = ?", TitleSlug,
	).Scan(&nMig); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if nMig != 0 {
		t.Errorf("metadata: %d steps tracés pour le titre B, want 0 (aucun step metadata)", nMig)
	}
}

// TestAdditionalTitleInheritsHISharedSchema — contrepartie de l'isolation : un
// titre additionnel qui n'enregistre AUCUN set retombe sur les migrations shared
// de Halo Infinite et obtient le schéma uniforme (match_registry, medals_earned,
// killer_victim_pairs, highlight_events, …) sans erreur de seed/backfill sur un
// warehouse frais. C'est la garantie d'UNIFORMITÉ inter-titres (Halo 5 a choisi
// d'hériter du schéma HI plutôt qu'un set propre) — symétrique du test
// d'isolation ci-dessus (un titre AVEC set n'hérite jamais de Halo).
func TestAdditionalTitleInheritsHISharedSchema(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Slug jamais enregistré dans migrationSets → fallback HI garanti.
	if err := migration.RunForTitleDB(db, "test_inherit_title", migration.TargetShared); err != nil {
		t.Fatalf("hériter du schéma shared HI (fallback): %v", err)
	}
	for _, tbl := range []string{
		"match_registry", "match_participants", "medals_earned",
		"highlight_events", "killer_victim_pairs", "weapon_kills", "xuid_aliases",
	} {
		if !tableExists(t, db, tbl) {
			t.Errorf("table HI %q absente — héritage du schéma shared cassé", tbl)
		}
	}
}
