//go:build integration

// Tests pour la migration rebuild_match_participants_defeat_art_corruption.
//
// Objectifs :
//   - Préservation 100% des rows (côté data)
//   - Préservation de toutes les colonnes (robustesse aux ALTER TABLE futurs)
//   - Recréation de la PRIMARY KEY (match_id, xuid)
//   - Recréation des 6 indexes idx_mp_*
//   - Idempotence via sentinel sync_meta `match_participants_rebuilt_v1`
//   - No-op gracieux si table absente
//
// Note : ce test ne reproduit PAS la corruption ART elle-même (impossible à
// déclencher délibérément sur :memory: — c'est un bug de plan d'exécution
// déterministe selon le contenu de la table). Il garantit la mécanique du
// rebuild, pas le fix de corruption. Le test E2E TestART_FilterPushdown_*
// (à venir) couvre la détection de corruption sur dataset hétérogène.
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedMatchParticipantsForRebuild crée la table avec son schéma complet +
// sync_meta + tables dépendantes (xuid_aliases, match_registry) requises par
// applyResolutionViews + applyMvPlayerMatchesView post-rebuild + quelques rows
// représentatives (multi-joueurs, multi-matchs).
func seedMatchParticipantsForRebuild(t *testing.T, db *sql.DB) {
	t.Helper()
	// Schémas calqués sur la prod (steps_shared.go). On instancie toutes les
	// colonnes référencées par applyResolutionViews + applyMvPlayerMatchesView.
	if _, err := db.Exec(`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			team_id INTEGER,
			outcome INTEGER,
			rank INTEGER,
			score INTEGER,
			kills INTEGER DEFAULT 0,
			deaths INTEGER DEFAULT 0,
			assists INTEGER DEFAULT 0,
			shots_fired INTEGER DEFAULT 0,
			shots_hit INTEGER DEFAULT 0,
			damage_dealt DOUBLE DEFAULT 0,
			damage_taken DOUBLE DEFAULT 0,
			kd DOUBLE DEFAULT 0,
			kda DOUBLE DEFAULT 0,
			accuracy DOUBLE DEFAULT 0,
			personal_score INTEGER DEFAULT 0,
			time_played_seconds INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0,
			headshot_kills SMALLINT DEFAULT 0,
			max_killing_spree SMALLINT DEFAULT 0,
			grenade_kills SMALLINT DEFAULT 0,
			melee_kills SMALLINT DEFAULT 0,
			power_weapon_kills SMALLINT DEFAULT 0,
			kills_expected DOUBLE,
			deaths_expected DOUBLE,
			team_mmr DOUBLE,
			enemy_mmr DOUBLE,
			backfill_bits INTEGER DEFAULT 0,
			created_at TIMESTAMP,
			PRIMARY KEY (match_id, xuid)
		);
		CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR,
			updated_at TIMESTAMP
		);
		-- Dépendances pour applyResolutionViews (v_gamertag_lookup).
		CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR,
			last_seen TIMESTAMP,
			source VARCHAR DEFAULT 'sync',
			updated_at TIMESTAMP
		);
		-- Dépendances pour applyMvPlayerMatchesView (toutes les colonnes match_registry référencées).
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			start_time_utc TIMESTAMP,
			end_time_utc TIMESTAMP,
			playlist_id VARCHAR,
			playlist_name VARCHAR,
			playlist_name_fr VARCHAR,
			map_id VARCHAR,
			map_name VARCHAR,
			map_name_fr VARCHAR,
			pair_name VARCHAR,
			pair_name_fr VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			game_variant_name VARCHAR,
			mode_category VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER,
			team_0_score SMALLINT,
			team_1_score SMALLINT,
			team_0_ps_score INTEGER,
			team_1_ps_score INTEGER,
			player_count INTEGER
		);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}

	// 2 matchs, 5 joueurs chacun = 10 rows.
	matches := []string{"match_alpha", "match_beta"}
	for _, mid := range matches {
		for i := 1; i <= 5; i++ {
			if _, err := db.Exec(`
				INSERT INTO match_participants
					(match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, assists, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
			`, mid, mid+"_xuid_"+itoa(i), "player_"+itoa(i), i%2, 2, i, 1000+i*10, 10+i, 5, 3); err != nil {
				t.Fatalf("insert %s xuid_%d: %v", mid, i, err)
			}
		}
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return ""
}

// TestRebuildMatchParticipants_PreservesAllRows : 10 rows seed → 10 rows after.
func TestRebuildMatchParticipants_PreservesAllRows(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	var before int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&before); err != nil {
		t.Fatalf("count before: %v", err)
	}
	if before != 10 {
		t.Fatalf("seed expected 10 rows, got %d", before)
	}

	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	var after int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&after); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != before {
		t.Fatalf("expected %d rows after rebuild, got %d", before, after)
	}
}

// TestRebuildMatchParticipants_PreservesAllColumns : toutes les colonnes
// présentes pre-rebuild restent présentes post-rebuild (incl. ordre).
func TestRebuildMatchParticipants_PreservesAllColumns(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	before, err := loadMatchParticipantsColumns(bootCtx(), db)
	if err != nil {
		t.Fatalf("columns before: %v", err)
	}
	if len(before) != 31 {
		t.Fatalf("seed expected 31 columns, got %d (%v)", len(before), before)
	}

	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	after, err := loadMatchParticipantsColumns(bootCtx(), db)
	if err != nil {
		t.Fatalf("columns after: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("expected %d columns after rebuild, got %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("column[%d] mismatch: before=%s after=%s", i, before[i], after[i])
		}
	}
}

// TestRebuildMatchParticipants_RecreatesPrimaryKey : la PK est bien recréée.
func TestRebuildMatchParticipants_RecreatesPrimaryKey(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	// Tentative d'INSERT dupliqué : doit échouer si la PK est présente.
	_, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, assists, created_at)
		VALUES ('match_alpha', 'match_alpha_xuid_1', 'dup', 0, 2, 1, 100, 1, 1, 1, NOW())
	`)
	if err == nil {
		t.Fatal("PK absente : INSERT dupliqué a réussi alors qu'il aurait dû violer la PK")
	}
}

// TestRebuildMatchParticipants_Idempotent : 2e exécution = no-op.
func TestRebuildMatchParticipants_Idempotent(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild 1: %v", err)
	}

	// Sentinel doit être présent.
	var marker sql.NullString
	if err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = ?`,
		matchParticipantsRebuildMetaKey).Scan(&marker); err != nil {
		t.Fatalf("sentinel non écrit : %v", err)
	}
	if !marker.Valid || marker.String != "true" {
		t.Fatalf("sentinel attendu 'true', got %v", marker)
	}

	// Compter rows avant 2e passe.
	var rowsBetween int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rowsBetween); err != nil {
		t.Fatalf("count between: %v", err)
	}

	// 2e exécution doit être no-op.
	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild 2 (doit être no-op): %v", err)
	}

	var rowsAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rowsAfter); err != nil {
		t.Fatalf("count after rebuild 2: %v", err)
	}
	if rowsAfter != rowsBetween {
		t.Fatalf("2e passe a modifié rows: between=%d after=%d", rowsBetween, rowsAfter)
	}
}

// TestRebuildMatchParticipants_NoTableNoOp : pas de crash si table absente,
// le sentinel est posé.
func TestRebuildMatchParticipants_NoTableNoOp(t *testing.T) {
	db := openMemDB(t)
	if _, err := db.Exec(`
		CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR,
			updated_at TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("seed sync_meta: %v", err)
	}

	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild (no table): %v", err)
	}

	// Sentinel posé pour éviter de re-check à chaque boot.
	var marker sql.NullString
	if err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = ?`,
		matchParticipantsRebuildMetaKey).Scan(&marker); err != nil {
		t.Fatalf("sentinel non écrit : %v", err)
	}
	if !marker.Valid {
		t.Fatal("sentinel doit être posé même quand table absente")
	}
}

// TestRebuildMatchParticipants_NoSyncMetaNoOp : pas de crash si sync_meta
// absent (cas très early boot où aucune migration n'a tourné).
func TestRebuildMatchParticipants_NoSyncMetaNoOp(t *testing.T) {
	db := openMemDB(t)
	// Ni match_participants, ni sync_meta — migration doit retourner sans erreur.
	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild (no sync_meta, no table): %v", err)
	}
}

// TestRebuildMatchParticipants_AvecVuesDejaPresentes : LE CAS RÉEL, et celui que les
// cinq tests ci-dessus ne couvraient pas — ils partaient tous d'une base SANS vues, où
// le rebuild les crée pour la première fois.
//
// En production la table porte TOUJOURS des vues dépendantes, et DuckDB NE LES SUPPRIME
// PAS au `DROP TABLE`, même avec CASCADE (mesuré le 2026-08-02, constat J4R-7) : elles
// survivent au catalogue, liées à une table qui n'existe plus. C'est exactement là que
// l'ancien `cmd/rebuild_mp` avortait — son `CREATE VIEW` de recréation tombait sur
// « View with name "v" already exists! », rollback, réparation impossible.
//
// Ce test exerce ce chemin de bout en bout : les vues existent AVANT, le rebuild doit
// passer, et les vues doivent être REBINDÉES sur la table neuve (donc interrogeables et
// rendant les lignes reconstruites). Il garde la propriété dont dépend `cmd/rebuild_mp`
// depuis qu'il délègue ici (dette H4).
func TestRebuildMatchParticipants_AvecVuesDejaPresentes(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	// Les vues sont posées AVANT le rebuild, comme sur une base de production.
	if err := ApplyResolutionViews(db); err != nil {
		t.Fatalf("pose des vues avant rebuild: %v", err)
	}
	if err := ApplyMvPlayerMatchesView(db); err != nil {
		t.Fatalf("pose de mv_player_matches avant rebuild: %v", err)
	}
	var avant int
	if err := db.QueryRow(`SELECT COUNT(*) FROM v_gamertag_lookup`).Scan(&avant); err != nil {
		t.Fatalf("v_gamertag_lookup interrogeable avant rebuild: %v", err)
	}

	if err := applyRebuildMatchParticipants(db); err != nil {
		t.Fatalf("rebuild avec vues déjà présentes — c'est le mode de panne de J4R-7: %v", err)
	}

	// Les vues doivent être VIVANTES, pas seulement présentes au catalogue : une vue
	// restée liée à la table détruite existe encore mais échoue à la lecture.
	var apres int
	if err := db.QueryRow(`SELECT COUNT(*) FROM v_gamertag_lookup`).Scan(&apres); err != nil {
		t.Fatalf("v_gamertag_lookup illisible APRÈS rebuild (vue non rebindée): %v", err)
	}
	if apres != avant {
		t.Errorf("v_gamertag_lookup: %d lignes avant, %d après — la vue ne voit plus les mêmes joueurs", avant, apres)
	}
	var mv int
	if err := db.QueryRow(`SELECT COUNT(*) FROM mv_player_matches`).Scan(&mv); err != nil {
		t.Fatalf("mv_player_matches illisible APRÈS rebuild (vue non rebindée): %v", err)
	}
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rows); err != nil {
		t.Fatalf("count match_participants: %v", err)
	}
	if rows != 10 {
		t.Errorf("match_participants: 10 lignes attendues après rebuild, got %d", rows)
	}
}
