//go:build cgo

// shared_weapon_kills_appendonly_test.go — déplacé depuis internal/migration
// (campagne append-only ART #23046). weapon_kills est créée par le créateur de
// schéma shared TITLE-OWNED (steps_shared_core.go) ; la migration append-only
// (generation_id + vue v_weapon_kills) reste GLOBALE (registre internal/migration).
// Ce test câble le provider title-owned (StepsFor) puis RunForDB(TargetShared) pour
// que la table soit créée ET la migration append-only appliquée — sans le provider,
// weapon_kills n'existerait pas dans le binaire de test du package global (cycle
// d'import).
//
// Tests cardinalité de l'append-only weapon_kills + vue v_weapon_kills.
// Verrouille la sémantique « dernière génération par (match_id, xuid) » (DENSE_RANK)
// et la colonne dérivée effective_weapon_id = COALESCE(reconciled_as, weapon_id).
//
// Sert AUSSI de garde anti-régression : si le CREATE agrégé revenait dans le créateur
// shared title-owned, les INSERT per-kill (time_ms) échoueraient ici.
//
// ⚠ CES TESTS MONTENT DÉSORMAIS UNE BASE HALO 5, ET LE FICHIER RESTE ICI. Depuis le
// 2026-09-01, `weapon_kills` est SUPPRIMÉE du fichier Halo Infinite
// (shared_drop_weapon_kills_v1) : monter la base sous le titre par défaut ne rendrait
// plus de table à tester. Halo 5 est le titre qui la CONSERVE — 550 926 lignes natives —
// et c'est donc lui qui porte la sémantique append-only vérifiée ici. Le fichier reste
// dans ce paquet parce que le schéma `shared` de Halo 5 est celui de Halo Infinite : pour
// cette cible, `TitleMigrationSet.OwnsTarget` fait retomber Halo 5 sur le provider
// title-owned d'ici (héritage documenté dans title_set.go).

package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/migration"
)

// setupPerKillWeaponKills monte une DB shared HALO 5 via la chaîne de migration réelle
// (provider title-owned câblé) : weapon_kills per-kill + reconciled_as + append-only
// generation_id/written_at + vue v_weapon_kills.
func setupPerKillWeaponKills(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(halo_5, TargetShared): %v", err)
	}
	return db
}

// TestWeaponKills_FreshSharedDB_IsPerKill — garde anti-régression du finding
// 2026-06-21 : une DB shared NEUVE doit produire le schéma PER-KILL (time_ms,
// generation_id, pas de colonne agrégée `kills`), pas l'ancien schéma agrégé.
func TestWeaponKills_FreshSharedDB_IsPerKill(t *testing.T) {
	db := setupPerKillWeaponKills(t)

	for _, col := range []string{"time_ms", "generation_id", "written_at"} {
		if has, _ := migration.ColumnExists(db, "weapon_kills", col); !has {
			t.Fatalf("weapon_kills.%s absente — DB neuve au schéma agrégé (régression)", col)
		}
	}
	// Colonne agrégée `kills` ABSENTE (sinon = ancien schéma v5 ressorti).
	if has, _ := migration.ColumnExists(db, "weapon_kills", "kills"); has {
		t.Fatal("weapon_kills.kills présente — schéma agrégé v5 ressorti (régression du fix)")
	}
	// Pas de PK fonctionnelle bloquant 2 kills même arme : INSERT per-kill OK.
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	wkInsert(t, db, "m1", "x1", 2000, 100, 0) // même (match,xuid,weapon) → doit passer
	if got := wkCount(t, db, `SELECT COUNT(*) FROM weapon_kills`); got != 2 {
		t.Fatalf("weapon_kills = %d, attendu 2 (per-kill, pas de PK agrégée)", got)
	}
}

func wkInsert(t *testing.T, db *sql.DB, match, xuid string, timeMs int, weaponID uint64, gen int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, generation_id)
		 VALUES (?, ?, ?, ?, ?)`,
		match, xuid, timeMs, weaponID, gen,
	); err != nil {
		t.Fatalf("insert weapon_kills (%s,%s,gen%d): %v", match, xuid, gen, err)
	}
}

func wkCount(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	return n
}

// TestWeaponKillsAppendOnly_GenerationSupersedes : une nouvelle génération (id
// supérieur) remplace l'ENSEMBLE des kills d'un (match,xuid) dans v_weapon_kills,
// par INSERT pur — la table physique croît, la vue ne montre que la dernière génération.
func TestWeaponKillsAppendOnly_GenerationSupersedes(t *testing.T) {
	db := setupPerKillWeaponKills(t)

	// Append-only appliqué : colonnes generation_id + written_at présentes.
	for _, col := range []string{"generation_id", "written_at"} {
		if has, _ := migration.ColumnExists(db, "weapon_kills", col); !has {
			t.Fatalf("colonne %s absente — append-only weapon_kills non appliqué", col)
		}
	}

	// Génération 0 (legacy) : 2 kills arme 100 pour (m1,x1).
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	wkInsert(t, db, "m1", "x1", 2000, 100, 0)
	// Indépendance par clé : (m1,x2) garde sa génération 0.
	wkInsert(t, db, "m1", "x2", 1500, 555, 0)

	// Génération 1 (réécriture) : 3 kills arme 200 pour (m1,x1).
	wkInsert(t, db, "m1", "x1", 1100, 200, 1)
	wkInsert(t, db, "m1", "x1", 2100, 200, 1)
	wkInsert(t, db, "m1", "x1", 3100, 200, 1)

	// Table physique : append pur → 6 rows (2+1+3), rien supprimé.
	if got := wkCount(t, db, `SELECT COUNT(*) FROM weapon_kills`); got != 6 {
		t.Fatalf("weapon_kills physique = %d, attendu 6 (append pur)", got)
	}

	// Vue : (m1,x1) ne montre que la génération 1 = 3 kills, tous arme 200.
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1'`); got != 3 {
		t.Fatalf("v_weapon_kills (m1,x1) = %d, attendu 3 (dernière génération)", got)
	}
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1' AND weapon_id=200`); got != 3 {
		t.Fatalf("v_weapon_kills (m1,x1) arme 200 = %d, attendu 3 (gen 0 supersédée)", got)
	}
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1' AND weapon_id=100`); got != 0 {
		t.Fatalf("v_weapon_kills (m1,x1) arme 100 = %d, attendu 0 (génération périmée)", got)
	}

	// Indépendance : (m1,x2) intacte (sa génération 0 reste la MAX pour cette clé).
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x2'`); got != 1 {
		t.Fatalf("v_weapon_kills (m1,x2) = %d, attendu 1 (clé indépendante)", got)
	}
}

// TestWeaponKillsAppendOnly_EffectiveWeaponID : effective_weapon_id = reconciled_as
// quand présent, sinon weapon_id brut.
func TestWeaponKillsAppendOnly_EffectiveWeaponID(t *testing.T) {
	db := setupPerKillWeaponKills(t)

	// Kill arme brute 100 sans reconciled_as → effective = 100.
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	// Kill arme brute 200 reconciliée vers 999 → effective = 999.
	if _, err := db.Exec(
		`INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, reconciled_as, generation_id)
		 VALUES ('m2','x1',1000,200,999,0)`); err != nil {
		t.Fatalf("insert reconciled: %v", err)
	}

	var eff uint64
	if err := db.QueryRow(
		`SELECT effective_weapon_id FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1'`).Scan(&eff); err != nil {
		t.Fatalf("select effective m1: %v", err)
	}
	if eff != 100 {
		t.Fatalf("effective_weapon_id (sans reconcile) = %d, attendu 100", eff)
	}
	if err := db.QueryRow(
		`SELECT effective_weapon_id FROM v_weapon_kills WHERE match_id='m2' AND xuid='x1'`).Scan(&eff); err != nil {
		t.Fatalf("select effective m2: %v", err)
	}
	if eff != 999 {
		t.Fatalf("effective_weapon_id (reconciled_as=999) = %d, attendu 999", eff)
	}
}

// TestWeaponKillsAppendOnly_Idempotent : 2e application = no-op, vue + colonnes stables.
func TestWeaponKillsAppendOnly_Idempotent(t *testing.T) {
	db := setupPerKillWeaponKills(t)
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	if err := migration.ApplyAppendOnlyWeaponKills(db); err != nil {
		t.Fatalf("ApplyAppendOnlyWeaponKills pass2 (idempotence): %v", err)
	}
	if got := wkCount(t, db, `SELECT COUNT(*) FROM weapon_kills`); got != 1 {
		t.Fatalf("weapon_kills après pass2 = %d, attendu 1 (données préservées)", got)
	}
	if got := wkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills`); got != 1 {
		t.Fatalf("v_weapon_kills après pass2 = %d, attendu 1", got)
	}
}
