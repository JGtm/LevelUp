package migrations

// shared_end_to_end_guards_test.go — LES ASSERTIONS QUE `internal/migration` NE POUVAIT
// PLUS JOUER, réveillées là où elles tournent (dette H5).
//
// Depuis la Phase 1.5 b23, le schéma de base shared est title-owned : quatre tests de
// `internal/migration/migration_test.go` (`TestRunForDB_Shared_*`) se skippaient donc à
// CHAQUE exécution, parce que leur prédicat `sharedBaseSchemaIsGlobal()` est faux depuis
// et le restera. Quatre assertions dormantes, vertes sans rien vérifier — le motif exact
// qui a produit J4R-4, à l'échelle de quatre tests.
//
// Trois de leurs assertions n'étaient couvertes NULLE PART ailleurs :
//   - la vue `v_killer_victim_full`, supprimée le 2026-08-02, ne doit pas revenir ;
//   - `v_gamertag_lookup` doit replier un bot INCONNU sur son xuid brut (le test
//     end-to-end voisin ne vérifie qu'un bot connu) ;
//   - la chaîne de migration shared doit être idempotente sur plusieurs passes.
//
// Elles vivent ici parce que ce paquet est le seul qui peut à la fois importer
// `migration` ET câbler le fournisseur de steps du titre (`SetTitleStepsProvider`) —
// depuis `internal/migration` lui-même, ce serait un cycle d'import. Les tests d'origine
// ont été supprimés : `git` garde leur histoire, et une coquille qui skippe toujours ne
// protège rien.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// openSharedMigratedDB ouvre une DuckDB in-memory et y joue la chaîne shared complète
// (steps globaux + steps du titre), comme au boot.
func openSharedMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	return db
}

func viewExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_type = 'VIEW'`,
		name).Scan(&n); err != nil {
		t.Fatalf("query vue %s: %v", name, err)
	}
	return n > 0
}

// TestTitleStepsShared_VueSupprimeeNeRevientPas : `v_killer_victim_full` a été retirée du
// schéma le 2026-08-02 — ses deux LEFT JOIN reproduisaient des colonnes déjà portées par
// la table, exécutés à CHAQUE chargement de vue match. La recréer serait une régression
// silencieuse : rien ne casserait, le travail mort reviendrait simplement.
func TestTitleStepsShared_VueSupprimeeNeRevientPas(t *testing.T) {
	db := openSharedMigratedDB(t)
	if viewExists(t, db, "v_killer_victim_full") {
		t.Error(`vue "v_killer_victim_full" présente après migration : elle a été supprimée le ` +
			`2026-08-02, et son unique lecteur (Q20) lit la table canonique directement`)
	}
}

// TestTitleStepsShared_GamertagLookupBotInconnu : un bot dont le numéro n'est pas au
// catalogue 343 doit ressortir en xuid BRUT, pas en chaîne vide ni en NULL.
//
// C'est le repli qui garantit qu'un identifiant reste affichable : le test end-to-end
// voisin ne couvre qu'un bot CONNU (`bid(3.0)` -> « 343 Ellis »), donc il passerait au
// vert même si le CASE perdait sa branche par défaut.
func TestTitleStepsShared_GamertagLookupBotInconnu(t *testing.T) {
	db := openSharedMigratedDB(t)
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time) VALUES ('t_bots', '2026-05-16');
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES
			('t_bots', 'bid(15.0)',  NULL, 1, 3),
			('t_bots', 'bid(999.0)', NULL, 0, 2);
	`); err != nil {
		t.Fatalf("seed bots: %v", err)
	}
	for _, c := range []struct{ xuid, want string }{
		{"bid(15.0)", "343 Mak"},     // bot au catalogue
		{"bid(999.0)", "bid(999.0)"}, // hors catalogue -> xuid brut
	} {
		var got string
		if err := db.QueryRow(
			`SELECT gamertag FROM v_gamertag_lookup WHERE xuid = ?`, c.xuid).Scan(&got); err != nil {
			t.Errorf("query v_gamertag_lookup %s: %v", c.xuid, err)
			continue
		}
		if got != c.want {
			t.Errorf("xuid %s : got %q, want %q", c.xuid, got, c.want)
		}
	}
}

// TestTitleStepsShared_Idempotent : la chaîne shared se rejoue sans rien laisser en
// suspens. Trois passes — la première crée, la deuxième prouve l'idempotence, la
// troisième attrape le step qui n'échouerait qu'à partir d'un état déjà convergé.
func TestTitleStepsShared_Idempotent(t *testing.T) {
	db := openSharedMigratedDB(t) // passe 1
	for pass := 2; pass <= 3; pass++ {
		if err := migration.RunForDB(db, migration.TargetShared); err != nil {
			t.Fatalf("RunForDB(Shared) passe %d: %v", pass, err)
		}
	}
	// Anti-vacuité : sur une table VIDE, « aucune ligne à schema_done=FALSE » est vrai
	// sans rien prouver. On exige d'abord que la chaîne ait effectivement enregistré des
	// migrations — sinon l'assertion suivante passerait au vert sur une DB non migrée.
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&total); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if total == 0 {
		t.Fatal("schema_migrations vide après 3 passes : la chaîne shared n'a rien enregistré")
	}
	var notDone int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE schema_done = FALSE`).Scan(&notDone); err != nil {
		t.Fatalf("query schema_done: %v", err)
	}
	if notDone > 0 {
		t.Errorf("%d migration(s) shared sur %d avec schema_done=FALSE après 3 passes", notDone, total)
	}
}
