package migration

// steps_shared.go — helpers DDL partagés pour shared_matches_v2.duckdb.
//
// Les 34 migrations TargetShared (create_base_shared_schema … repair_v_gamertag_lookup
// _bots_2026_05_30) ont été migrées vers
// internal/games/halo_infinite/migrations/steps_shared_core.go (Phase 1.5 b23, voie B).
//
// MAIS les 6 helpers RESTENT ici (exportés) car ApplyResolutionViews + ApplyMvPlayerMatchesView
// sont appelés par RebuildMatchParticipantsART (steps_shared_rebuild_match_participants.go),
// util runtime appelé par cmd/server au boot + cmd/force_rebuild_art + internal/sync — qui DOIT
// rester dans le package migration. Les déplacer au titre créerait un cycle migration→titre.
// Les steps title-owned appellent migration.ApplyResolutionViews etc.
//
// Les noms des 34 steps restent dans internal/migration/order.go (canonicalOrder).

import (
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/analysis"
)

// DropAssistsExpectedShared supprime les colonnes assists_expected / assists_stddev
// de match_participants via la stratégie table-rename (évite ALTER TABLE DROP COLUMN
// qui échoue sur DuckDB 1.0 quand des vues ou contraintes dépendent de la table).
func DropAssistsExpectedShared(db *sql.DB) error {
	drop := map[string]bool{"assists_expected": true, "assists_stddev": true}

	rows, err := db.QueryContext(bootCtx(), `PRAGMA table_info('match_participants')`)
	if err != nil {
		return fmt.Errorf("pragma table_info match_participants: %w", err)
	}
	var keep []string
	hasDrop := false
	for rows.Next() {
		var cid int
		var name, typ string
		var nn bool
		var dflt *string
		var pk bool
		if err := rows.Scan(&cid, &name, &typ, &nn, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		if drop[name] {
			hasDrop = true
		} else {
			keep = append(keep, name)
		}
	}
	rows.Close()
	if !hasDrop {
		return nil // colonnes déjà absentes — idempotent
	}

	colList := strings.Join(keep, ", ")
	stmts := []string{
		`DROP TABLE IF EXISTS _mp_backup_assists`,
		fmt.Sprintf(`CREATE TABLE _mp_backup_assists AS SELECT %s FROM match_participants`, colList),
		`DROP TABLE match_participants`,
		fmt.Sprintf(`CREATE TABLE match_participants AS SELECT %s FROM _mp_backup_assists`, colList),
		`ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid)`,
		`DROP TABLE _mp_backup_assists`,
	}
	for _, s := range stmts {
		end := 60
		if end > len(s) {
			end = len(s)
		}
		if _, err := db.ExecContext(bootCtx(), s); err != nil {
			return fmt.Errorf("dropAssistsExpected (%s...): %w", s[:end], err)
		}
	}
	if err := ApplyResolutionViews(db); err != nil {
		return fmt.Errorf("recreate views: %w", err)
	}
	if err := ApplyMvPlayerMatchesView(db); err != nil {
		return fmt.Errorf("recreate mv_player_matches: %w", err)
	}
	for _, ddl := range []string{
		"CREATE INDEX IF NOT EXISTS idx_mp_backfill   ON match_participants(xuid, backfill_bits)",
		"CREATE INDEX IF NOT EXISTS idx_mp_xuid_match ON match_participants(xuid, match_id)",
		"CREATE INDEX IF NOT EXISTS idx_mp_match_xuid ON match_participants(match_id, xuid)",
		"CREATE INDEX IF NOT EXISTS idx_mp_xuid_team  ON match_participants(xuid, team_id, match_id)",
		"CREATE INDEX IF NOT EXISTS idx_mp_xuid       ON match_participants(xuid)",
		"CREATE INDEX IF NOT EXISTS idx_mp_match_id   ON match_participants(match_id)",
	} {
		if _, err := db.ExecContext(bootCtx(), ddl); err != nil {
			return fmt.Errorf("recreate index: %w", err)
		}
	}
	return nil
}

// ApplyHighlightEventsAutoincrement recrée highlight_events avec séquence.
func ApplyHighlightEventsAutoincrement(db *sql.DB) error {
	exists, err := tableExists(db, "highlight_events")
	if err != nil || !exists {
		return execScript(db, `
			CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
			CREATE TABLE IF NOT EXISTS highlight_events (
				id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
				match_id   VARCHAR NOT NULL,
				event_type VARCHAR NOT NULL,
				time_ms    INTEGER,
				xuid       VARCHAR,
				type_hint  INTEGER,
				raw_json   VARCHAR
			);
			CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
		`)
	}

	var colDefault sql.NullString
	err = db.QueryRowContext(bootCtx(),
		"SELECT column_default FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'highlight_events' AND column_name = 'id'",
	).Scan(&colDefault)
	if err == nil && colDefault.Valid && len(colDefault.String) > 4 {
		return nil // Séquence déjà configurée
	}

	var maxID int
	_ = db.QueryRowContext(bootCtx(), "SELECT COALESCE(MAX(id), 0) FROM highlight_events").Scan(&maxID)
	startVal := maxID + 1

	return execScript(db, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS _he_backup AS SELECT * FROM highlight_events;
		DROP TABLE IF EXISTS highlight_events CASCADE;
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq START WITH %d;
		CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR
		);
		INSERT INTO highlight_events SELECT * FROM _he_backup;
		DROP TABLE IF EXISTS _he_backup;
		CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
	`, startVal))
}

// ApplyMedalsBigint migre medals_earned.medal_name_id vers BIGINT.
func ApplyMedalsBigint(db *sql.DB) error {
	var dataType string
	err := db.QueryRowContext(bootCtx(),
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'medals_earned' AND column_name = 'medal_name_id'",
	).Scan(&dataType)
	if err != nil || dataType == "BIGINT" {
		return nil // Déjà BIGINT
	}

	return execScript(db, `
		CREATE TABLE IF NOT EXISTS medals_earned_new (
			match_id      VARCHAR,
			xuid          VARCHAR,
			medal_name_id BIGINT,
			count         SMALLINT,
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (match_id, xuid, medal_name_id)
		);
		INSERT INTO medals_earned_new SELECT match_id, xuid, CAST(medal_name_id AS BIGINT), count, created_at FROM medals_earned;
		DROP TABLE medals_earned;
		ALTER TABLE medals_earned_new RENAME TO medals_earned;
	`)
}

// ApplyDropHighlightEventsGamertag recrée highlight_events sans la colonne gamertag.
func ApplyDropHighlightEventsGamertag(db *sql.DB) error {
	hasCol, err := columnExists(db, "highlight_events", "gamertag")
	if err != nil || !hasCol {
		return err // Colonne absente = rien à faire
	}

	return execScript(db, `
		CREATE TABLE IF NOT EXISTS _he_backup2 AS
			SELECT id, match_id, event_type, time_ms, xuid, type_hint, raw_json FROM highlight_events;
		DROP TABLE IF EXISTS highlight_events CASCADE;
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
		CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR
		);
		INSERT INTO highlight_events SELECT * FROM _he_backup2;
		DROP TABLE IF EXISTS _he_backup2;
		CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
	`)
}

// ApplyResolutionViews crée les vues SQL v6 garanties (v_gamertag_lookup résolveur unifié,
// v_match_full). Appelée par les steps title-owned ET par RebuildMatchParticipantsART (qui
// recrée les vues post-rebuild). DDL de v_gamertag_lookup généré par
// analysis.GamertagLookupViewSQL (source unique partagée avec le boot).
//
// BASCULE DU 2026-08-02 — deux changements, et le second est une suppression :
//
//  1. la dépendance garantie n'est plus `killer_victim_pairs` mais `match_kill_events` : c'est
//     elle que `v_gamertag_lookup` lit désormais, et DuckDB bind les vues à leur création ;
//  2. `v_killer_victim_full` N'EST PLUS CRÉÉE. Elle projetait `kvp.*` PUIS deux colonnes
//     `killer_gamertag`/`victim_gamertag` re-jointes qui portaient déjà ces noms-là : les deux
//     LEFT JOIN sur `v_gamertag_lookup` étaient du travail mort exécuté à chaque chargement de
//     vue match, et la vue ne « marchait » que par le renommage silencieux qu'applique DuckDB
//     aux colonnes homonymes. Son seul lecteur (Q20) lit désormais la TABLE
//     `killer_victim_pairs` directement, et rend les mêmes six colonnes. Aucune vue de
//     compatibilité n'a été posée à sa place : il n'y en a pas.
func ApplyResolutionViews(db *sql.DB) error {
	if err := EnsureMatchKillEvents(db); err != nil {
		return fmt.Errorf("ensure match_kill_events (dep v_gamertag_lookup): %w", err)
	}
	if _, err := db.ExecContext(bootCtx(), `CREATE TABLE IF NOT EXISTS killer_victim_pairs (
		match_id        VARCHAR NOT NULL,
		killer_xuid     VARCHAR NOT NULL,
		killer_gamertag VARCHAR,
		victim_xuid     VARCHAR NOT NULL,
		victim_gamertag VARCHAR,
		kill_count      INTEGER DEFAULT 1,
		time_ms         INTEGER,
		is_validated    BOOLEAN DEFAULT FALSE,
		created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("ensure killer_victim_pairs (lue par Q20 et ~20 sites): %w", err)
	}
	if _, err := db.ExecContext(bootCtx(), analysis.GamertagLookupViewSQL()); err != nil {
		return fmt.Errorf("create v_gamertag_lookup: %w", err)
	}

	_, _ = db.ExecContext(bootCtx(), `
		CREATE OR REPLACE VIEW v_match_full AS
		SELECT mr.*
		FROM match_registry mr
	`)

	return nil
}

// ApplyMvPlayerMatchesView crée ou recrée mv_player_matches.
func ApplyMvPlayerMatchesView(db *sql.DB) error {
	_, err := db.ExecContext(bootCtx(), `
		CREATE OR REPLACE VIEW mv_player_matches AS
		SELECT
			mr.match_id,
			mr.start_time,
			mr.end_time,
			mr.start_time_utc,
			mr.end_time_utc,
			mr.playlist_id,
			mr.playlist_name,
			mr.playlist_name_fr,
			mr.map_id,
			mr.map_name,
			mr.map_name_fr,
			mr.pair_name,
			mr.pair_name_fr,
			mr.pair_id,
			mr.game_variant_id,
			mr.game_variant_name,
			mr.mode_category,
			mr.is_ranked,
			mr.is_firefight,
			mr.duration_seconds,
			mr.playable_duration_seconds,
			mr.team_0_score,
			mr.team_1_score,
			mr.team_0_ps_score,
			mr.team_1_ps_score,
			mr.player_count,
			mp.xuid,
			mp.gamertag,
			mp.team_id,
			mp.outcome,
			mp.rank,
			mp.score,
			mp.kills,
			mp.deaths,
			mp.assists,
			CASE WHEN mp.deaths > 0 THEN ROUND(CAST(mp.kills AS DOUBLE) / mp.deaths, 2) ELSE mp.kills END AS kd_ratio,
			mp.kda,
			mp.accuracy,
			mp.shots_fired,
			mp.shots_hit,
			mp.damage_dealt,
			mp.damage_taken,
			mp.personal_score,
			mp.time_played_seconds,
			mp.avg_life_seconds,
			mp.headshot_kills,
			mp.max_killing_spree,
			mp.grenade_kills,
			mp.melee_kills,
			mp.power_weapon_kills,
			mp.team_mmr,
			mp.enemy_mmr,
			mp.kills_expected,
			mp.deaths_expected,
			mp.backfill_bits
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
	`)
	return err
}
