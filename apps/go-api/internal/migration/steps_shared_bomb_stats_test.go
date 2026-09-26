//go:build cgo

package migration

// Tests du DDL `match_bomb_stats`. Ils verrouillent trois choses :
//
//  1. l'IDEMPOTENCE du step (deux passages, aucune erreur) — le boot rejoue le DDL sur des
//     bases a des etats differents ;
//  2. la vue retient LA DERNIERE LIGNE par (match_id, xuid) — c'est le contrat append-only
//     d'ADR 0026, et le seul chemin de lecture autorise ;
//  3. les cinq colonnes de mesure acceptent NULL — « absent n'est pas zero » n'est pas qu'une
//     convention de code, c'est une propriete du schema. Une colonne NOT NULL forcerait le
//     persister a ecrire un zero fabrique.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func bombStatsDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTmpDB(t)
	if err := applyMatchBombStats(db); err != nil {
		t.Fatalf("applyMatchBombStats: %v", err)
	}
	if err := applyMatchBombStats(db); err != nil {
		t.Fatalf("applyMatchBombStats (2e passage): %v", err)
	}
	return db
}

// TestVueBombeRetientLaDerniereLigne — deux ecritures successives pour la meme clef : la table
// brute garde les deux (append-only), la vue ne rend que la plus recente.
func TestVueBombeRetientLaDerniereLigne(t *testing.T) {
	db := bombStatsDB(t)

	for _, v := range []int{1, 5} {
		if _, err := db.Exec(`INSERT INTO match_bomb_stats
			(match_id, xuid, bomb_detonations, written_at)
			VALUES ('m1', 'xuid(1)', ?, TIMESTAMP '2026-09-04 10:00:00')`, v); err != nil {
			t.Fatalf("insert %d: %v", v, err)
		}
	}

	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_bomb_stats`).Scan(&brut); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	if brut != 2 {
		t.Errorf("append-only : 2 lignes attendues en table brute, %d", brut)
	}
	var n, deto int
	if err := db.QueryRow(`SELECT COUNT(*), MIN(bomb_detonations)
		FROM match_bomb_stats_latest`).Scan(&n, &deto); err != nil {
		t.Fatalf("lecture _latest: %v", err)
	}
	// written_at identique a la MILLISECONDE pres : c'est `id DESC` qui departage, et c'est
	// exactement le cas que ce tie-break existe pour couvrir.
	if n != 1 || deto != 5 {
		t.Errorf("_latest devrait rendre 1 ligne a 5, got n=%d deto=%d", n, deto)
	}
}

// TestColonnesDeMesureNullables — les cinq mesures acceptent NULL. Une colonne NOT NULL
// obligerait a ecrire un zero qui n'a jamais ete mesure.
func TestColonnesDeMesureNullables(t *testing.T) {
	db := bombStatsDB(t)

	if _, err := db.Exec(`INSERT INTO match_bomb_stats (match_id, xuid, bomb_grabs)
		VALUES ('m2', 'xuid(2)', 0)`); err != nil {
		t.Fatalf("insert avec 4 mesures absentes: %v", err)
	}
	var nulles int
	if err := db.QueryRow(`SELECT
		CAST(bomb_detonations IS NULL AS INT) + CAST(bomb_arms IS NULL AS INT) +
		CAST(time_as_bomb_carrier_seconds IS NULL AS INT) +
		CAST(bomb_carriers_killed IS NULL AS INT)
		FROM match_bomb_stats_latest WHERE match_id = 'm2'`).Scan(&nulles); err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if nulles != 4 {
		t.Errorf("les 4 mesures non renseignees doivent rester NULL, %d le sont", nulles)
	}
}
