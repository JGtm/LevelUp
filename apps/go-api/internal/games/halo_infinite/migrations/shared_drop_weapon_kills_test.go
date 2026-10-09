package migrations

// shared_drop_weapon_kills_test.go — LA PROPRIÉTÉ QUI PROTÈGE HALO 5.
//
// Ce lot supprime `weapon_kills` du fichier Halo Infinite (D2). Halo 5 y tient 550 926
// lignes `confidence = 'native'`, issues de la timeline de son API — donnée autoritaire.
// Les deux titres ont des FICHIERS distincts (ADR 0008) mais le MÊME SCHÉMA, et pour la
// cible `shared` Halo 5 hérite DÉLIBÉRÉMENT des steps de Halo Infinite
// (`TitleMigrationSet.OwnsTarget` ne lui donne que `metadata`).
//
// Sans le garde `OnlyTitles`, ce DROP aurait donc tourné sur les DEUX fichiers. Le test
// fait tourner la migration sur DEUX bases forgées, une par titre, et exige les deux
// résultats OPPOSÉS. Il est le seul endroit où cette propriété se vérifie.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/migration"
)

// baseSharedMigree ouvre une base `shared` neuve et y applique les migrations du titre.
func baseSharedMigree(t *testing.T, slug string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForTitleDB(db, slug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(%s, shared): %v", slug, err)
	}
	return db
}

// compteObjet rend le nombre d'objets de ce nom dans le catalogue, tous schémas confondus.
func compteObjet(t *testing.T, db *sql.DB, requete, nom string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(requete, nom).Scan(&n); err != nil {
		t.Fatalf("catalogue(%s): %v", nom, err)
	}
	return n
}

const (
	requeteTables    = `SELECT COUNT(*) FROM duckdb_tables() WHERE table_name = ?`
	requeteVues      = `SELECT COUNT(*) FROM duckdb_views() WHERE view_name = ?`
	requeteSequences = `SELECT COUNT(*) FROM duckdb_sequences() WHERE sequence_name = ?`
)

// TestDropWeaponKills_PartSurHaloInfinite : sur le titre par défaut, la table, ses DEUX
// vues, la table morte v3 et la séquence disparaissent — et le reste du schéma reste debout.
func TestDropWeaponKills_PartSurHaloInfinite(t *testing.T) {
	db := baseSharedMigree(t, migration.DefaultSlug)

	for _, nom := range []string{"weapon_kills", "weapon_kills_v3"} {
		if n := compteObjet(t, db, requeteTables, nom); n != 0 {
			t.Errorf("table %s encore présente (%d) après la migration de suppression", nom, n)
		}
	}
	// LES DEUX SCHÉMAS. Mesuré en production : la vue existe en `main` ET en `shared` ;
	// un `DROP VIEW IF EXISTS v_weapon_kills` nu n'en tombe qu'UNE et le nom survit.
	if n := compteObjet(t, db, requeteVues, "v_weapon_kills"); n != 0 {
		t.Errorf("vue v_weapon_kills encore présente dans %d schéma(s) — le DROP ne qualifie "+
			"pas les deux", n)
	}
	if n := compteObjet(t, db, requeteSequences, "weapon_kills_generation_seq"); n != 0 {
		t.Errorf("séquence weapon_kills_generation_seq encore présente (%d)", n)
	}

	// Le reste du schéma shared n'a pas bougé : la suppression est CIBLÉE.
	for _, nom := range []string{"match_registry", "match_participants", "match_kill_events"} {
		if n := compteObjet(t, db, requeteTables, nom); n != 1 {
			t.Errorf("table %s = %d, attendue 1 — la migration a débordé de sa cible", nom, n)
		}
	}
}

// TestDropWeaponKills_TombeSurLesDeuxSchemas : LA FORME RÉELLE DE LA PRODUCTION.
//
// Mesuré le 2026-09-01 sur `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` :
// `v_weapon_kills` existe en `main` ET en `shared` — cette seconde est une vue LEGACY
// (`SELECT *, COALESCE(...)`, d'avant l'append-only) qu'AUCUNE migration courante ne crée.
// Une base forgée par les seules migrations ne porte donc que la première : sans ce test,
// la moitié du problème resterait non couverte, et un `DROP VIEW IF EXISTS v_weapon_kills`
// nu (qui ne résout que sur le search_path) passerait au vert tout en laissant le nom
// vivant dans l'autre schéma.
func TestDropWeaponKills_TombeSurLesDeuxSchemas(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// On reproduit la forme de production À LA MAIN, sans passer par les migrations :
	// c'est la seule façon d'obtenir la vue legacy du schéma `shared`.
	for _, ddl := range []string{
		`CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT,
			reconciled_as UBIGINT, generation_id BIGINT DEFAULT 0)`,
		`CREATE SEQUENCE weapon_kills_generation_seq START 1`,
		`CREATE VIEW v_weapon_kills AS
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id FROM weapon_kills`,
		`CREATE SCHEMA shared`,
		`CREATE VIEW shared.v_weapon_kills AS
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id FROM weapon_kills`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("forge (%s): %v", ddl, err)
		}
	}
	if n := compteObjet(t, db, requeteVues, "v_weapon_kills"); n != 2 {
		t.Fatalf("la base forgée porte %d vue(s) v_weapon_kills, attendu 2 — le test ne "+
			"mesure rien s'il ne reproduit pas la forme de production", n)
	}

	if err := dropWeaponKillsHINF(db); err != nil {
		t.Fatalf("dropWeaponKillsHINF: %v", err)
	}

	if n := compteObjet(t, db, requeteVues, "v_weapon_kills"); n != 0 {
		t.Errorf("%d vue(s) v_weapon_kills survivent — le DROP ne qualifie pas les DEUX schémas", n)
	}
	if n := compteObjet(t, db, requeteTables, "weapon_kills"); n != 0 {
		t.Errorf("table weapon_kills survivante (%d)", n)
	}
	if n := compteObjet(t, db, requeteSequences, "weapon_kills_generation_seq"); n != 0 {
		t.Errorf("séquence survivante (%d)", n)
	}
}

// TestDropWeaponKills_Idempotent : rejouer la suppression sur une base qui ne porte plus
// rien ne doit pas échouer — un `DROP … IF EXISTS` sur un schéma ABSENT est une erreur, pas
// un no-op, et c'est ce qui a imposé de lire `duckdb_views()` avant de qualifier.
func TestDropWeaponKills_Idempotent(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for i := 0; i < 2; i++ {
		if err := dropWeaponKillsHINF(db); err != nil {
			t.Fatalf("dropWeaponKillsHINF passe %d sur base vierge: %v", i+1, err)
		}
	}
}

// TestDropWeaponKills_SurvitSurHalo5 : LE GARDE-FOU DU LOT. Sur le second titre, la table
// et ses vues SURVIVENT — parce que le step porte `OnlyTitles`, pas parce qu'il serait
// title-owned (pour `shared`, Halo 5 hérite des steps de Halo Infinite).
func TestDropWeaponKills_SurvitSurHalo5(t *testing.T) {
	db := baseSharedMigree(t, halo5.TitleSlug)

	if n := compteObjet(t, db, requeteTables, "weapon_kills"); n != 1 {
		t.Fatalf("weapon_kills = %d sur Halo 5, attendue 1 — le DROP a franchi la frontière "+
			"de titre et effacerait 550 926 lignes NATIVES en production", n)
	}
	if n := compteObjet(t, db, requeteVues, "v_weapon_kills"); n == 0 {
		t.Error("v_weapon_kills absente sur Halo 5 — les lecteurs du second titre lisent la vue")
	}
	// La colonne propre à Halo 5 (shared_h5_weapon_kill_kind_v1) est toujours là : c'est
	// elle qui prouve que le schéma H5 est complet, pas seulement présent.
	if has, err := migration.ColumnExists(db, "weapon_kills", "kill_kind"); err != nil || !has {
		t.Errorf("weapon_kills.kill_kind absente sur Halo 5 (err=%v) — le schéma du second "+
			"titre est incomplet", err)
	}
}
