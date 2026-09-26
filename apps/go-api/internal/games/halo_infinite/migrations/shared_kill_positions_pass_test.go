//go:build cgo

// shared_kill_positions_pass_test.go — lot 1.7 (2026-09-09) : `kill_positions` arbitre par
// PASSE DE DÉCODAGE et non plus par clé. Ces tests verrouillent la SPEC de la bascule (la
// mécanique générique du swap CTAS est déjà couverte par
// internal/migration/append_only_rebuild_test.go) :
//
//  1. une DB neuve porte `decode_pass` NOT NULL ;
//  2. les lignes DÉJÀ EN BASE reçoivent une passe synthétique PAR MATCH, donc la vue les sert
//     TOUTES — aucun match n'est vidé par la migration, aucun n'est affecté par un autre ;
//  3. une passe neuve PLUS COURTE RÉTRACTE la position que le re-décodage ne retrouve plus —
//     LE défaut que ce lot ferme ;
//  4. la passe synthétique ne RESSUSCITE pas un doublon de clé que l'ancienne vue masquait.
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// killPositionsLegacyDB monte une DB portant kill_positions au schéma PRÉ-G.2 (celui où
// Halo 5 écrit depuis toujours), y sème des lignes, puis applique la conversion append-only
// G.2 SEULE. C'est l'état de production exact au moment où le step de ce lot s'applique.
func killPositionsLegacyDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE kill_positions (
			match_id    VARCHAR NOT NULL,
			killer_xuid VARCHAR,
			time_ms     INTEGER,
			killer_x    DOUBLE, killer_y DOUBLE, killer_z DOUBLE,
			victim_x    DOUBLE, victim_y DOUBLE, victim_z DOUBLE
		)`); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO kill_positions (match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z)
		VALUES
			('mA', 'K1', 1000, 1.0, 1.0, 0.0),
			('mA', 'K2', 2000, 2.0, 2.0, 0.0),
			('mB', 'K3', 3000, 3.0, 3.0, 0.0)`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	if err := applyAppendOnlyKillPositions(db); err != nil {
		t.Fatalf("G.2 applyAppendOnlyKillPositions: %v", err)
	}
	return db
}

// TestKillPositions_DBNeuve_PorteDecodePassNotNull : une DB shared NEUVE, migrée par la chaîne
// RÉELLE (canonicalOrder), doit porter `decode_pass` NOT NULL. Le NOT NULL est la moitié utile
// de la colonne : une ligne sans passe ne pourrait être ni retenue ni écartée par la vue.
func TestKillPositions_DBNeuve_PorteDecodePassNotNull(t *testing.T) {
	db := setupKillPositionsSharedDB(t)

	has, err := migration.ColumnExists(db, "kill_positions", "decode_pass")
	if err != nil {
		t.Fatalf("ColumnExists(decode_pass): %v", err)
	}
	if !has {
		t.Fatal("kill_positions.decode_pass absente sur DB neuve (step absent de canonicalOrder ?)")
	}

	_, err = db.Exec(`INSERT INTO kill_positions (match_id, killer_xuid, time_ms, killer_x)
		VALUES ('m-sans-passe', 'K1', 1000, 1.0)`)
	if err == nil {
		t.Fatal("un INSERT sans decode_pass a été accepté, attendu un refus (NOT NULL)")
	}
}

// TestKillPositions_LignesLegacy_UnePasseParMatch : le point délicat de la migration. Les
// lignes déjà en base n'ont jamais été écrites en passes ; elles reçoivent `legacy-<match_id>`
// — UNE passe par match — donc la vue les sert TOUTES, exactement comme avant la migration.
// Une passe synthétique CONSTANTE (inter-matchs) aurait fait retirer de la vue les lignes de
// tous les autres matchs dès la première passe neuve sur un seul d'entre eux.
func TestKillPositions_LignesLegacy_UnePasseParMatch(t *testing.T) {
	db := killPositionsLegacyDB(t)

	if err := applyKillPositionsDecodePass(db); err != nil {
		t.Fatalf("applyKillPositionsDecodePass: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest`).Scan(&n); err != nil {
		t.Fatalf("count vue: %v", err)
	}
	if n != 3 {
		t.Fatalf("kill_positions_latest = %d lignes, attendu 3 (aucune ligne legacy perdue)", n)
	}
	var distincts int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT decode_pass) FROM kill_positions`).Scan(&distincts); err != nil {
		t.Fatalf("count passes: %v", err)
	}
	if distincts != 2 {
		t.Fatalf("%d decode_pass distincts, attendu 2 (une passe synthétique PAR MATCH)", distincts)
	}
	var passeA string
	if err := db.QueryRow(
		`SELECT DISTINCT decode_pass FROM kill_positions WHERE match_id = 'mA'`).Scan(&passeA); err != nil {
		t.Fatalf("select passe mA: %v", err)
	}
	if passeA != "legacy-mA" {
		t.Errorf("decode_pass de mA = %q, attendu \"legacy-mA\"", passeA)
	}

	// Idempotence : le marqueur `decode_pass` fait no-oper le second passage (la vue est
	// seulement rafraîchie), aucune ligne perdue ni dupliquée.
	if err := applyKillPositionsDecodePass(db); err != nil {
		t.Fatalf("2e passe (idempotence): %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions`).Scan(&n); err != nil {
		t.Fatalf("count après 2e passe: %v", err)
	}
	if n != 3 {
		t.Fatalf("kill_positions après 2e passe = %d, attendu 3 (idempotence)", n)
	}
}

// TestKillPositions_PasseNeuvePlusCourte_Retracte : LE défaut que ce lot ferme. Une passe
// neuve qui ne retrouve QU'UNE des deux positions du match ne doit PAS laisser survivre
// l'autre — la position rétractée disparaît de la vue, la table brute la garde (append-only).
// Le match voisin, lui, n'est pas touché.
func TestKillPositions_PasseNeuvePlusCourte_Retracte(t *testing.T) {
	db := killPositionsLegacyDB(t)
	if err := applyKillPositionsDecodePass(db); err != nil {
		t.Fatalf("applyKillPositionsDecodePass: %v", err)
	}

	// Re-décodage de mA : seul le frag K1 est encore localisable.
	if _, err := db.Exec(`
		INSERT INTO kill_positions (match_id, decode_pass, killer_xuid, time_ms, killer_x, killer_y, killer_z, written_at)
		VALUES ('mA', 'passe-neuve', 'K1', 1000, 9.0, 9.0, 0.0, TIMESTAMP '2099-01-01 00:00:00')`); err != nil {
		t.Fatalf("insert passe neuve: %v", err)
	}

	var n int
	var kx float64
	if err := db.QueryRow(
		`SELECT COUNT(*), MAX(killer_x) FROM kill_positions_latest WHERE match_id = 'mA'`).Scan(&n, &kx); err != nil {
		t.Fatalf("select vue mA: %v", err)
	}
	if n != 1 || kx != 9.0 {
		t.Fatalf("vue mA = %d ligne(s) / killer_x %v, attendu 1 / 9 — la passe neuve fait foi ENTIÈRE, "+
			"la position que le re-décodage ne retrouve plus ne doit PAS survivre", n, kx)
	}
	var survivante int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = 'mA' AND time_ms = 2000`).Scan(&survivante); err != nil {
		t.Fatalf("select rétractée: %v", err)
	}
	if survivante != 0 {
		t.Errorf("%d ligne(s) à t=2000 dans la vue, attendu 0 (la vue arbitre par PASSE, pas par clé)",
			survivante)
	}
	// Rien n'est supprimé : la table brute garde les 4 lignes.
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions`).Scan(&n); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	if n != 4 {
		t.Errorf("table brute = %d lignes, attendu 4 (append-only : rien n'est supprimé)", n)
	}
	// Le match voisin n'est PAS affecté par le re-décodage de mA.
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = 'mB'`).Scan(&n); err != nil {
		t.Fatalf("select vue mB: %v", err)
	}
	if n != 1 {
		t.Errorf("vue mB = %d ligne(s), attendu 1 (un re-décodage d'un autre match ne retire rien ici)", n)
	}
}

// TestKillPositions_PasseLegacy_NeRessuscitePasUnDoublonDeCle : la raison du second étage de
// la vue. Deux écritures successives sur le MÊME match avant ce lot (re-décodage post-G.2,
// re-insertion builder) ont laissé deux lignes de même clé, que l'ancienne vue PAR CLÉ
// masquait. Les fondre dans une seule passe synthétique ne doit pas les rendre TOUTES LES DEUX
// à la vue : ce serait une régression introduite par la migration elle-même (morts
// double-comptées chez tous les lecteurs).
func TestKillPositions_PasseLegacy_NeRessuscitePasUnDoublonDeCle(t *testing.T) {
	db := killPositionsLegacyDB(t)

	// Écriture postérieure à G.2 sur une clé DÉJÀ présente (position affinée).
	if _, err := db.Exec(`
		INSERT INTO kill_positions (match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, written_at)
		VALUES ('mA', 'K1', 1000, 5.0, 5.0, 0.0, TIMESTAMP '2099-01-01 00:00:00')`); err != nil {
		t.Fatalf("insert doublon post-G.2: %v", err)
	}
	if err := applyKillPositionsDecodePass(db); err != nil {
		t.Fatalf("applyKillPositionsDecodePass: %v", err)
	}

	var n int
	var kx float64
	if err := db.QueryRow(`SELECT COUNT(*), MAX(killer_x) FROM kill_positions_latest
		WHERE match_id = 'mA' AND killer_xuid = 'K1' AND time_ms = 1000`).Scan(&n, &kx); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 1 {
		t.Fatalf("vue = %d ligne(s) pour la clé, attendu 1 — la passe synthétique ne doit pas "+
			"ressusciter le doublon que l'ancienne vue par clé masquait", n)
	}
	if kx != 5.0 {
		t.Errorf("killer_x = %v, attendu 5 (la ligne la plus récente de la passe legacy)", kx)
	}
	// Les autres lignes du match restent servies : le second étage borne l'INTÉRIEUR d'une
	// passe, il ne retient pas « une ligne par match ».
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = 'mA'`).Scan(&n); err != nil {
		t.Fatalf("select vue mA: %v", err)
	}
	if n != 2 {
		t.Errorf("vue mA = %d ligne(s), attendu 2 (K1 dédoublonné + K2 intact)", n)
	}
}
