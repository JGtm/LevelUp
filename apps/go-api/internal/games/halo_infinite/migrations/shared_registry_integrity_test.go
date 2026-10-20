package migrations

// shared_registry_integrity_test.go — les deux steps de reconstruction de match_registry
// (shared_registry_participants_bit_v1, title-owned, puis shared_registry_drop_integrity_columns_v1,
// global) joués PAR LE RUNNER : le bit participants n'est posé que pour le titre que borne
// OnlyTitles, les colonnes partent partout. Et, si LEVELUP_E5_REAL_DB est posée, la répétition
// sur une COPIE d'une base partagée réelle (jamais le fichier d'un serveur qui tourne).

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

const (
	stepBitParticipants = "shared_registry_participants_bit_v1"
	stepColonnesRetrait = "shared_registry_drop_integrity_columns_v1"
	slugSansBit         = "halo_5"
)

// baseLegacyParLeRunner : base partagée neuve migrée par le runner, puis remise dans l'état des
// bases d'avant le retrait (deux colonnes, index, vues qui les exposent) et ré-armée pour les deux
// steps (lignes du ledger retirées : écriture de TEST sur une base jetable).
func baseLegacyParLeRunner(t *testing.T, slug string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForTitleDB(db, slug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(%s) base neuve: %v", slug, err)
	}
	var mv string
	if err := db.QueryRow(`SELECT sql FROM duckdb_views() WHERE view_name = 'mv_player_matches'`).Scan(&mv); err != nil {
		t.Fatal(err)
	}
	const ancre = "mr.team_1_ps_score, "
	if !strings.Contains(mv, ancre) {
		t.Fatalf("ancre %q absente de mv_player_matches : %s", ancre, mv)
	}
	mvAncienne := strings.Replace(strings.Replace(mv, ancre, ancre+"mr.player_count, ", 1),
		"CREATE VIEW ", "CREATE OR REPLACE VIEW ", 1)
	for _, stmt := range []string{
		`ALTER TABLE match_registry ADD COLUMN participants_loaded BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE match_registry ADD COLUMN player_count SMALLINT DEFAULT 0`,
		`CREATE INDEX idx_registry_player_count ON match_registry(player_count)`,
		`CREATE OR REPLACE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`,
		mvAncienne,
		`INSERT INTO match_registry (match_id, start_time, backfill_completed, player_count)
			VALUES ('avec', TIMESTAMP '2026-05-25 12:00:00', 1, 8), ('sans', TIMESTAMP '2026-05-26 12:00:00', 0, 0)`,
		`INSERT INTO match_participants (match_id, xuid, team_id, outcome) VALUES ('avec', 'x1', 0, 2), ('avec', 'x2', 1, 3)`,
		`DELETE FROM schema_migrations WHERE name IN ('` + stepBitParticipants + `', '` + stepColonnesRetrait + `')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("mise en état legacy (%.60s): %v", stmt, err)
		}
	}
	return db
}

func scanRegistre(t *testing.T, db *sql.DB, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func TestRegistryIntegrity_ParLeRunner_TitreQuiEcritLeBit(t *testing.T) {
	db := baseLegacyParLeRunner(t, migration.DefaultSlug)
	if err := migration.RunForTitleDB(db, migration.DefaultSlug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB: %v", err)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM duckdb_columns() WHERE table_name = 'match_registry'
		AND column_name IN ('participants_loaded', 'player_count')`); n != 0 {
		t.Errorf("%d colonne(s) d'intégrité restantes", n)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = 'idx_registry_player_count'`); n != 0 {
		t.Error("idx_registry_player_count encore présent")
	}
	if n := scanRegistre(t, db, `SELECT backfill_completed FROM match_registry WHERE match_id = 'avec'`); n != 513 {
		t.Errorf("match à roster : backfill_completed = %d, attendu 513", n)
	}
	if n := scanRegistre(t, db, `SELECT backfill_completed FROM match_registry WHERE match_id = 'sans'`); n != 0 {
		t.Errorf("match sans roster : backfill_completed = %d, attendu 0", n)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE name IN (?, ?)`,
		stepBitParticipants, stepColonnesRetrait); n != 2 {
		t.Errorf("ledger : %d step(s) inscrits, attendu 2", n)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM mv_player_matches WHERE match_id = 'avec'`); n != 2 {
		t.Errorf("mv_player_matches = %d lignes, attendu 2", n)
	}
}

func TestRegistryIntegrity_ParLeRunner_TitreSansLeBit(t *testing.T) {
	db := baseLegacyParLeRunner(t, slugSansBit)
	if err := migration.RunForTitleDB(db, slugSansBit, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB: %v", err)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM duckdb_columns() WHERE table_name = 'match_registry'
		AND column_name IN ('participants_loaded', 'player_count')`); n != 0 {
		t.Errorf("%d colonne(s) d'intégrité restantes", n)
	}
	if n := scanRegistre(t, db, `SELECT backfill_completed FROM match_registry WHERE match_id = 'avec'`); n != 1 {
		t.Errorf("titre sans le bit : backfill_completed = %d, attendu 1 (inchangé)", n)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stepBitParticipants); n != 0 {
		t.Error("le step title-owned ne doit ni s'appliquer ni s'inscrire hors de son titre")
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stepColonnesRetrait); n != 1 {
		t.Error("le step global doit s'inscrire")
	}
}

// TestRegistryIntegrity_RepetitionSurCopie : LEVELUP_E5_REAL_DB = chemin d'une base partagée
// réelle (lue seulement : le test en fait une copie), LEVELUP_E5_REAL_SLUG = son titre. Joue le
// runner COMPLET (toutes les étapes en attente, comme au boot), journalise comptes, index,
// colonnes, durée ; second passage = no-op.
func TestRegistryIntegrity_RepetitionSurCopie(t *testing.T) {
	src, slug := os.Getenv("LEVELUP_E5_REAL_DB"), os.Getenv("LEVELUP_E5_REAL_SLUG")
	if src == "" || slug == "" {
		t.Skip("LEVELUP_E5_REAL_DB / LEVELUP_E5_REAL_SLUG non posées : répétition non demandée")
	}
	// Nom du fichier de production : le catalogue DuckDB prend le nom du fichier, et un catalogue
	// « shared » rendrait ambiguë la référence au schéma `shared` (shared_drop_weapon_kills_v1).
	dst := filepath.Join(t.TempDir(), "shared_matches_v2.duckdb")
	if err := copierFichier(src, dst); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", dst)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test
	migration.SetTitleStepsProvider(StepsFor)

	journaliserEtat(t, db, "AVANT")
	if os.Getenv("LEVELUP_E5_STEPS_SEULS") != "" {
		jouerLesDeuxStepsSeuls(t, db, slug)
		journaliserEtat(t, db, "APRÈS (steps seuls)")
		oid := scanRegistre(t, db, `SELECT table_oid FROM duckdb_tables() WHERE table_name = 'match_registry'`)
		jouerLesDeuxStepsSeuls(t, db, slug)
		if got := scanRegistre(t, db, `SELECT table_oid FROM duckdb_tables() WHERE table_name = 'match_registry'`); got != oid {
			t.Errorf("second passage des steps seuls : table reconstruite (oid %d -> %d)", oid, got)
		}
		return
	}
	ledgerAvant := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations`)
	debut := time.Now()
	if err := migration.RunForTitleDB(db, slug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(%s): %v", slug, err)
	}
	t.Logf("DURÉE runner complet : %s", time.Since(debut))
	t.Logf("steps appliqués : %d", scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations`)-ledgerAvant)
	for _, nom := range []string{stepBitParticipants, stepColonnesRetrait} {
		t.Logf("ledger %s : %d", nom, scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, nom))
	}
	journaliserEtat(t, db, "APRÈS")
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM duckdb_columns() WHERE table_name = 'match_registry'
		AND column_name IN ('participants_loaded', 'player_count')`); n != 0 {
		t.Errorf("%d colonne(s) d'intégrité restantes", n)
	}

	oid := scanRegistre(t, db, `SELECT table_oid FROM duckdb_tables() WHERE table_name = 'match_registry'`)
	ledger := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations`)
	debut = time.Now()
	if err := migration.RunForTitleDB(db, slug, migration.TargetShared); err != nil {
		t.Fatalf("second passage: %v", err)
	}
	// Les deux steps rejoués hors ledger, bornés par OnlyTitles comme le runner : aucune écriture.
	jouerLesDeuxStepsSeuls(t, db, slug)
	t.Logf("DURÉE second passage : %s", time.Since(debut))
	if got := scanRegistre(t, db, `SELECT table_oid FROM duckdb_tables() WHERE table_name = 'match_registry'`); got != oid {
		t.Errorf("second passage : table reconstruite (oid %d -> %d)", oid, got)
	}
	if got := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations`); got != ledger {
		t.Errorf("second passage : %d step(s) de plus au ledger", got-ledger)
	}
}

// jouerLesDeuxStepsSeuls : les deux steps, dans l'ordre canonique et selon OnlyTitles comme le
// runner, sans les autres étapes en attente — mesure la durée propre de chacun.
func jouerLesDeuxStepsSeuls(t *testing.T, db *sql.DB, slug string) {
	t.Helper()
	global, ok := migration.ByName(stepColonnesRetrait)
	if !ok {
		t.Fatalf("%s absent du registre", stepColonnesRetrait)
	}
	for _, m := range []migration.Migration{stepRegistryParticipantsBit(), global} {
		if len(m.OnlyTitles) > 0 && !slices.Contains(m.OnlyTitles, slug) {
			t.Logf("%s : hors du titre %s", m.Name, slug)
			continue
		}
		debut := time.Now()
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("%s: %v", m.Name, err)
		}
		t.Logf("DURÉE %s : %s", m.Name, time.Since(debut))
	}
}

func journaliserEtat(t *testing.T, db *sql.DB, quand string) {
	t.Helper()
	t.Logf("%s matchs=%d bit512=%d roster_sans_bit=%d backfill_null=%d", quand,
		scanRegistre(t, db, `SELECT COUNT(*) FROM match_registry`),
		scanRegistre(t, db, `SELECT COUNT(*) FROM match_registry WHERE (COALESCE(backfill_completed, 0) & 512) <> 0`),
		scanRegistre(t, db, `SELECT COUNT(*) FROM match_registry WHERE match_id IN (SELECT match_id FROM match_participants)
			AND (COALESCE(backfill_completed, 0) & 512) = 0`),
		scanRegistre(t, db, `SELECT COUNT(*) FROM match_registry WHERE backfill_completed IS NULL`))
	for _, q := range []string{
		`SELECT string_agg(column_name, ',' ORDER BY column_name) FROM duckdb_columns()
			WHERE table_name = 'match_registry' AND column_name IN ('participants_loaded', 'player_count')`,
		`SELECT string_agg(index_name, ',' ORDER BY index_name) FROM duckdb_indexes() WHERE table_name = 'match_registry'`,
		`SELECT CAST(COUNT(*) AS VARCHAR) || ' colonnes' FROM duckdb_columns() WHERE table_name = 'match_registry'`,
		`SELECT string_agg(view_name, ',' ORDER BY view_name) FROM duckdb_views() WHERE NOT internal AND sql ILIKE '%match_registry%'`,
	} {
		var s sql.NullString
		if err := db.QueryRow(q).Scan(&s); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		t.Logf("%s %s", quand, s.String)
	}
}

func copierFichier(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck // lecture
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
