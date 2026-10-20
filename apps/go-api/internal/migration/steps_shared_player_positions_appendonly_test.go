//go:build cgo

package migration

// Tests de la CONVERSION append-only de `match_player_positions` (decision 1 du plan v2).
//
// # POURQUOI CE FICHIER EXISTE (constat C4 de la revue A-R1)
//
// La migration n'avait AUCUN test, alors que toutes les autres conversions append-only en ont
// un. Sa propriete centrale est ecrite dans son en-tete — « les lignes deja en base survivent,
// EN UNE PASSE » — et elle n'etait gardee par rien : muter `SyntheticCols` en
// `CAST(written_at AS VARCHAR) AS positions_pass` laissait TOUTE la suite verte (unitaire ET
// `-tags=integration -p 1`), pendant qu'en production `match_player_positions_latest` n'aurait
// plus servi qu'UNE SEULE LIGNE par match au lieu de toutes les lignes preexistantes : la carte
// de chaleur de chaque match deja rempli se serait reduite a un point.
//
// # PAS UNE DDL RECOPIEE
//
// La table de depart est celle que la MIGRATION REELLE cree (`shared_match_player_positions_v1`,
// resolue par son nom dans le registre). Une DDL recopiee dans le test aurait derive du jour ou
// la vraie aurait bouge, et le test serait reste vert sur un schema qui n'existe plus (lecon du
// depot : « DDL de test recopiees = derive indetectable »).
//
// Base sur FICHIER temporaire et non `:memory:` : `database/sql` gere un POOL, et une base
// DuckDB en memoire est propre a chaque connexion — le CTAS et la relecture pourraient tomber
// sur deux bases differentes. C'est la convention des autres tests de migration (`openTmpDB`).

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// stepNomme rend la migration enregistree sous ce nom. Elle DOIT exister : un renommage de
// step qui laisserait ce test chercher un fantome doit rougir, pas se taire.
func stepNomme(t *testing.T, nom string) Migration {
	t.Helper()
	for _, m := range All() {
		if m.Name == nom {
			return m
		}
	}
	t.Fatalf("migration %q absente du registre", nom)
	return Migration{}
}

// basePositionsLegacy rend une base portant la table `match_player_positions` DANS SA FORME
// D'ORIGINE (pas de PK, pas de passe, pas de vue) avec `n` lignes ecrites a des instants
// DIFFERENTS — c'est exactement ce que l'ancien `-write` du diagnostic laissait derriere lui,
// et c'est ce qui fait la difference entre « une generation » et « n generations d'une ligne ».
func basePositionsLegacy(t *testing.T, n int) *sql.DB {
	t.Helper()
	db := openTmpDB(t)
	if err := stepNomme(t, "shared_match_player_positions_v1").ApplySchema(db); err != nil {
		t.Fatalf("creation de match_player_positions: %v", err)
	}
	for i := 0; i < n; i++ {
		if _, err := db.Exec(`INSERT INTO match_player_positions
			(match_id, time_ms, x, y, z, team, written_at)
			VALUES ('m1', ?, ?, ?, 0, -1, TIMESTAMP '2026-08-01 10:00:00' + INTERVAL (?) SECOND)`,
			i*20_000, float32(i), float32(2*i), i); err != nil {
			t.Fatalf("insert legacy %d: %v", i, err)
		}
	}
	return db
}

// compte rend le nombre de lignes d'une table ou d'une vue.
func comptePositions(t *testing.T, db *sql.DB, relation string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + relation).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", relation, err)
	}
	return n
}

// TestPositionsAppendOnly_LesLignesLegacySurviventEnUnePasse — LA propriete du constat C4.
//
// Trois lignes ecrites a trois instants differents forment UNE generation apres conversion :
// les trois sont servies par la vue. Avec une passe par ligne, la vue n'en servirait qu'une.
func TestPositionsAppendOnly_LesLignesLegacySurviventEnUnePasse(t *testing.T) {
	db := basePositionsLegacy(t, 3)
	if err := applyAppendOnlyPlayerPositions(db); err != nil {
		t.Fatalf("conversion append-only: %v", err)
	}

	if got := comptePositions(t, db, "match_player_positions"); got != 3 {
		t.Fatalf("%d ligne(s) en base apres conversion, attendu 3 — la conversion a PERDU des "+
			"lignes", got)
	}
	if got := comptePositions(t, db, "match_player_positions_latest"); got != 3 {
		t.Fatalf("%d ligne(s) servie(s) par match_player_positions_latest, attendu 3 — les "+
			"lignes ecrites a des instants differents ont ete eclatees en autant de PASSES "+
			"d'une ligne, et la carte de chaleur de chaque match deja rempli se reduit a un "+
			"point (constat C4)", got)
	}
	var passes int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT positions_pass) FROM match_player_positions`).
		Scan(&passes); err != nil {
		t.Fatalf("count passes: %v", err)
	}
	if passes != 1 {
		t.Errorf("%d passes distinctes pour la generation legacy, attendu 1", passes)
	}
	var pass string
	if err := db.QueryRow(`SELECT DISTINCT positions_pass FROM match_player_positions`).
		Scan(&pass); err != nil {
		t.Fatalf("lecture de la passe: %v", err)
	}
	if pass != legacyPositionsPass {
		t.Errorf("passe legacy = %q, attendu %q — un operateur qui lit la table doit voir d'ou "+
			"viennent ces lignes", pass, legacyPositionsPass)
	}
}

// TestPositionsAppendOnly_Idempotente — le DDL est rejoue a CHAQUE boot, sur des bases a des
// etats differents. Un second passage ne doit ni echouer ni retoucher les lignes.
func TestPositionsAppendOnly_Idempotente(t *testing.T) {
	db := basePositionsLegacy(t, 3)
	for i := 0; i < 2; i++ {
		if err := applyAppendOnlyPlayerPositions(db); err != nil {
			t.Fatalf("conversion append-only (passage %d): %v", i+1, err)
		}
	}
	if got := comptePositions(t, db, "match_player_positions"); got != 3 {
		t.Errorf("%d ligne(s) apres deux passages, attendu 3", got)
	}
	if got := comptePositions(t, db, "match_player_positions_latest"); got != 3 {
		t.Errorf("%d ligne(s) servie(s) apres deux passages, attendu 3", got)
	}
}

// TestPositionsAppendOnly_UnePasseNeuveSupersedeLaGenerationLegacy — l'autre moitie du
// contrat : la premiere projection d'un artefact remplace TOUTE la generation legacy, sans
// effacer une seule ligne (append-only).
func TestPositionsAppendOnly_UnePasseNeuveSupersedeLaGenerationLegacy(t *testing.T) {
	db := basePositionsLegacy(t, 3)
	if err := applyAppendOnlyPlayerPositions(db); err != nil {
		t.Fatalf("conversion append-only: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`INSERT INTO match_player_positions
			(match_id, time_ms, x, y, z, team, positions_pass, written_at)
			VALUES ('m1', ?, 9, 9, 9, 0, 'passe-neuve', TIMESTAMP '2026-09-06 12:00:00')`,
			i*20_000); err != nil {
			t.Fatalf("insert passe neuve %d: %v", i, err)
		}
	}
	if got := comptePositions(t, db, "match_player_positions"); got != 5 {
		t.Errorf("%d ligne(s) en base, attendu 5 — append-only : rien n'est efface", got)
	}
	var n int
	var pass string
	if err := db.QueryRow(`SELECT COUNT(*), MIN(positions_pass)
		FROM match_player_positions_latest`).Scan(&n, &pass); err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	if n != 2 || pass != "passe-neuve" {
		t.Errorf("_latest sert %d ligne(s) de la passe %q, attendu 2 de passe-neuve — la vue "+
			"doit retenir LA DERNIERE PASSE PAR MATCH", n, pass)
	}
}
