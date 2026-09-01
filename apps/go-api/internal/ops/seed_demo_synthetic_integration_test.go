//go:build integration

// Package ops — seed_demo_synthetic_integration_test.go : E2E du générateur
// synthétique (DuckDB live, CGO). Vérifie la structure produite + le déterminisme
// (deux runs → mêmes données).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	assertCount(t, shared, "SELECT COUNT(*) FROM match_csrs", 22) // sessions classées (arène) = 10 + 12
	assertCountAtLeast(t, shared, "SELECT COUNT(*) FROM medals_earned", 1)

	// L'ARME FAVORITE DE L'ACCUEIL se lit dans la SOURCE DE DÉGÂT des morts créditées au
	// joueur (`favoriteWeaponFromSource`), plus dans une table d'armes. Trois exigences,
	// et l'encart disparaît si l'une tombe : des morts SOURCÉES en nombre, une part NON
	// ATTRIBUÉE (la portée est par ligne, comme en production), et une gagnante NETTE —
	// une égalité ferait dépendre l'arme affichée du départage par clé.
	assertCountAtLeast(t, shared, "SELECT COUNT(*) FROM match_kill_events WHERE source_tag IS NOT NULL", 100)
	assertCountAtLeast(t, shared, "SELECT COUNT(*) FROM match_kill_events WHERE source_tag IS NULL", 1)
	assertCount(t, shared, fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT source_tag FROM match_kill_events WHERE source_tag IS NOT NULL
			GROUP BY source_tag ORDER BY COUNT(*) DESC LIMIT 1
		) WHERE source_tag = %d`, demoSourceTagBR75), 1)
	// Invariant du schéma : source_tag / source_category / diverges voyagent ENSEMBLE.
	assertCount(t, shared, `
		SELECT COUNT(*) FROM match_kill_events
		WHERE (source_tag IS NULL) <> (source_category IS NULL)
		   OR (source_tag IS NULL) <> (diverges IS NULL)`, 0)

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
	// LE DERNIER MAILLON DE L'ARME FAVORITE. `resolveWeaponKeyDimensions` joint le registre
	// pour rendre l'identifiant NUMÉRIQUE de la clé, et `favoriteWeaponFromSource` ÉCARTE
	// toute arme dont cet identifiant vaut 0 (objet hors arsenal). Une clé présente dans
	// `weapons` mais absente de `weapon_ids` ferait donc disparaître l'encart alors même que
	// les morts portent leur source — panne muette que ce compte attrape.
	assertCount(t, meta, `
		SELECT COUNT(*) FROM weapons w
		JOIN weapon_ids wi ON wi.title_slug = w.title_slug AND wi.weapon_key = w.weapon_key
		WHERE w.title_slug = 'halo_infinite'
		  AND w.weapon_key IN ('hinf_br75', 'hinf_ma40_ar', 'hinf_bandit')
		  AND TRY_CAST(wi.id_value AS UBIGINT) IS NOT NULL
		  AND TRY_CAST(wi.id_value AS UBIGINT) <> 0`, 4) // le Bandit porte 2 identifiants
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM medal_definitions", 4)
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM career_ranks", 4)
	assertCountAtLeast(t, meta, "SELECT COUNT(*) FROM citation_mappings", 10)
}

// detDBs : DBs comparées entre deux runs (toutes les surfaces produites par le
// seeder, pas seulement shared_matches_v2). Le déterminisme byte-identique doit
// couvrir métadonnées + les 3 player DBs, sinon une dérive d'horloge / d'ordre de
// map dans un writer player/meta passe inaperçue (piège fetched_at).
var detDBs = []string{
	filepath.Join("warehouse", "metadata.duckdb"),
	filepath.Join("warehouse", "shared_matches_v2.duckdb"),
	filepath.Join("warehouse", "shared_social.duckdb"),
	filepath.Join("players", "DEMO", "stats.duckdb"),
	filepath.Join("players", "DEMO2", "stats.duckdb"),
	filepath.Join("players", "DEMO3", "stats.duckdb"),
}

// detAuditCols : colonnes d'AUDIT (bookkeeping) à défaut horloge non ancrable
// (DEFAULT CURRENT_TIMESTAMP dans les DDL de migration, jamais posées par le
// seeder ni assertées par une spec E2E). Exclues du dump — toutes les colonnes
// MÉTIER (kills, match_id, rank, CSR, fetched_at, written_at, recorded_at,
// start_time, id…) sont comparées octet à octet.
var detAuditCols = map[string]bool{"created_at": true, "updated_at": true}

// detSeededTables : tables réellement ÉCRITES par le seeder synthétique (grep des
// INSERT de seed_demo_synthetic*.go) — le périmètre du déterminisme. On NE compare
// PAS les tables de référence seedées par les migrations (asset_translations,
// lusr_hyperparams_v2, …) : leur colonne d'horodatage (written_at CURRENT_TIMESTAMP)
// reflète l'instant d'exécution des migrations, pas la donnée fixture, et diverge
// légitimement d'un run à l'autre. Ni les registres du runner (schema_migrations,
// title_schema_version). Une nouvelle table écrite par le seeder doit être ajoutée
// ici (et rendue déterministe).
var detSeededTables = map[string]bool{
	"career_progression":      true,
	"career_ranks":            true,
	"highlight_events":        true,
	"killer_victim_pairs":     true,
	"match_citations":         true,
	"match_csrs":              true,
	"match_kill_events":       true,
	"match_participants":      true,
	"match_registry":          true,
	"match_skill_rank":        true,
	"medal_definitions":       true,
	"medals_earned":           true,
	"player_csr_snapshots":    true,
	"player_match_enrichment": true,
	"sessions":                true,
	"sync_meta":               true,
	"xuid_aliases":            true,
}

// TestSeedDemoSynthetic_Deterministic : deux générations → DONNÉES identiques,
// table par table, sur toutes les DBs. Compare le contenu réel (chaque ligne
// sérialisée, triée) et non une empreinte agrégée faible — un writer qui laisse
// fuiter l'horloge (time.Now / DEFAULT CURRENT_TIMESTAMP non ancré) ou un ordre de
// map dans une colonne métier fait diverger deux runs et échoue ici.
func TestSeedDemoSynthetic_Deterministic(t *testing.T) {
	setSyntheticProviders()
	dump := func() map[string]string {
		out := t.TempDir()
		if _, err := SeedDemoSynthetic(context.Background(), SyntheticDemoOptions{OutDir: out, ServiceTag: "DEMO"}); err != nil {
			t.Fatalf("SeedDemoSynthetic: %v", err)
		}
		sigs := map[string]string{}
		for _, rel := range detDBs {
			db := openRO(t, filepath.Join(out, rel))
			sigs[rel] = dumpDBData(t, db)
			db.Close()
		}
		return sigs
	}
	a, b := dump(), dump()
	for _, rel := range detDBs {
		if a[rel] != b[rel] {
			t.Errorf("DB %s non déterministe entre deux runs:\n%s", rel, firstDivergentTable(a[rel], b[rel]))
		}
	}
}

// dumpDBData sérialise TOUTES les tables de base d'une DB (contenu réel, lignes
// triées) — pour comparaison byte-à-byte entre deux runs.
func dumpDBData(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'main' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		t.Fatalf("liste tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()

	var sb strings.Builder
	for _, table := range tables {
		if !detSeededTables[table] {
			continue
		}
		sb.WriteString("### ")
		sb.WriteString(table)
		sb.WriteByte('\n')
		sb.WriteString(dumpTableData(t, db, table))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// dumpTableData sérialise le contenu d'une table (colonnes métier uniquement,
// lignes triées) en une chaîne stable comparable entre deux runs.
func dumpTableData(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	colRows, err := db.Query(
		`SELECT column_name FROM information_schema.columns
		 WHERE table_schema = 'main' AND table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		t.Fatalf("colonnes %s: %v", table, err)
	}
	var cols []string
	for colRows.Next() {
		var c string
		if err := colRows.Scan(&c); err != nil {
			colRows.Close()
			t.Fatalf("scan colonne %s: %v", table, err)
		}
		if !detAuditCols[c] {
			cols = append(cols, c)
		}
	}
	colRows.Close()
	if len(cols) == 0 {
		return "<aucune colonne métier>"
	}

	exprs := make([]string, len(cols))
	for i, c := range cols {
		exprs[i] = `COALESCE("` + c + `"::VARCHAR, '∅')`
	}
	q := `SELECT CONCAT_WS('|', ` + strings.Join(exprs, ", ") + `) FROM "` + table + `"`
	dataRows, err := db.Query(q)
	if err != nil {
		t.Fatalf("dump %s: %v", table, err)
	}
	defer dataRows.Close()
	var lines []string
	for dataRows.Next() {
		var s sql.NullString
		if err := dataRows.Scan(&s); err != nil {
			t.Fatalf("scan ligne %s: %v", table, err)
		}
		lines = append(lines, s.String)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// firstDivergentTable retourne le bloc de la première table qui diffère (aide au
// diagnostic sans noyer la sortie de test).
func firstDivergentTable(a, b string) string {
	as := strings.Split(a, "### ")
	bs := strings.Split(b, "### ")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return "run1[" + truncate(as[i]) + "]\nrun2[" + truncate(bs[i]) + "]"
		}
	}
	return "(divergence hors table alignée)"
}

func truncate(s string) string {
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
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
