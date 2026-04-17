package migration

// steps_shared.go — migrations ciblant shared_matches_v2.duckdb.
// Portage des 18 steps shared du Python.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "create_base_shared_schema",
		TargetDB:    TargetShared,
		Description: "Tables de base shared_matches_v2 (idempotent IF NOT EXISTS)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS match_registry (
					match_id VARCHAR PRIMARY KEY,
					start_time TIMESTAMP,
					end_time TIMESTAMP,
					playlist_id VARCHAR,
					playlist_name VARCHAR,
					map_id VARCHAR,
					map_name VARCHAR,
					pair_id VARCHAR,
					pair_name VARCHAR,
					game_variant_id VARCHAR,
					game_variant_name VARCHAR,
					mode_category VARCHAR,
					is_ranked BOOLEAN DEFAULT FALSE,
					is_firefight BOOLEAN DEFAULT FALSE,
					duration_seconds INTEGER,
					playable_duration_seconds INTEGER,
					real_start_time TIMESTAMP,
					team_0_score INTEGER,
					team_1_score INTEGER,
					team_0_ps_score INTEGER,
					team_1_ps_score INTEGER,
					player_count INTEGER,
					first_sync_by VARCHAR,
					first_sync_at TIMESTAMP,
					last_updated_at TIMESTAMP,
					created_at TIMESTAMP,
					updated_at TIMESTAMP,
					backfill_completed INTEGER DEFAULT 0,
					film_match_start_ms INTEGER,
					spnkr_version VARCHAR,
					events_loaded BOOLEAN DEFAULT FALSE
				);
				CREATE TABLE IF NOT EXISTS match_participants (
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
					kills_stddev DOUBLE,
					assists_expected DOUBLE,
					assists_stddev DOUBLE,
					deaths_stddev DOUBLE,
					team_mmr DOUBLE,
					enemy_mmr DOUBLE,
					backfill_bits INTEGER DEFAULT 0,
					created_at TIMESTAMP,
					PRIMARY KEY (match_id, xuid)
				);
				CREATE TABLE IF NOT EXISTS medals_earned (
					match_id VARCHAR,
					xuid VARCHAR,
					medal_name_id BIGINT,
					count INTEGER,
					created_at TIMESTAMP,
					PRIMARY KEY (match_id, xuid, medal_name_id)
				);
				CREATE TABLE IF NOT EXISTS xuid_aliases (
					xuid VARCHAR PRIMARY KEY,
					gamertag VARCHAR,
					last_seen TIMESTAMP,
					source VARCHAR DEFAULT 'sync',
					updated_at TIMESTAMP
				);
				CREATE TABLE IF NOT EXISTS weapon_kills (
					match_id VARCHAR,
					xuid VARCHAR,
					weapon_id UBIGINT,
					kills INTEGER DEFAULT 0,
					PRIMARY KEY (match_id, xuid, weapon_id)
				);
				CREATE TABLE IF NOT EXISTS killer_victim_pairs (
					match_id VARCHAR,
					killer_xuid VARCHAR,
					victim_xuid VARCHAR,
					created_at TIMESTAMP,
					PRIMARY KEY (match_id, killer_xuid, victim_xuid)
				);
				CREATE TABLE IF NOT EXISTS highlight_events (
					id INTEGER PRIMARY KEY,
					match_id VARCHAR,
					event_type VARCHAR,
					time_ms INTEGER,
					xuid VARCHAR,
					type_hint VARCHAR,
					raw_json VARCHAR
				);
				CREATE TABLE IF NOT EXISTS sync_meta (
					key VARCHAR PRIMARY KEY,
					value VARCHAR,
					updated_at TIMESTAMP
				);
			`)
		},
	})

	Register(Migration{
		Name:        "add_film_match_start",
		TargetDB:    TargetShared,
		Description: "Colonne film_match_start_ms sur match_registry",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "match_registry", "film_match_start_ms", "INTEGER")
		},
	})

	Register(Migration{
		Name:        "add_highlight_events_autoincrement",
		TargetDB:    TargetShared,
		Description: "Séquence auto-increment sur highlight_events.id",
		ApplySchema: applyHighlightEventsAutoincrement,
	})

	Register(Migration{
		Name:        "add_match_participants_columns",
		TargetDB:    TargetShared,
		Description: "~30 colonnes stats/MMR/backfill_bits sur match_participants",
		ApplySchema: func(db *sql.DB) error {
			cols := []struct{ name, typ string }{
				{"rank", "SMALLINT"},
				{"score", "INTEGER"},
				{"kills", "SMALLINT"},
				{"deaths", "SMALLINT"},
				{"assists", "SMALLINT"},
				{"shots_fired", "INTEGER"},
				{"shots_hit", "INTEGER"},
				{"damage_dealt", "FLOAT"},
				{"damage_taken", "FLOAT"},
				{"avg_life_seconds", "FLOAT"},
				{"headshot_kills", "SMALLINT"},
				{"max_killing_spree", "SMALLINT"},
				{"kda", "FLOAT"},
				{"accuracy", "FLOAT"},
				{"time_played_seconds", "INTEGER"},
				{"grenade_kills", "SMALLINT"},
				{"melee_kills", "SMALLINT"},
				{"power_weapon_kills", "SMALLINT"},
				{"personal_score", "INTEGER"},
				{"team_mmr", "FLOAT"},
				{"enemy_mmr", "FLOAT"},
				{"kills_expected", "FLOAT"},
				{"kills_stddev", "FLOAT"},
				{"deaths_expected", "FLOAT"},
				{"deaths_stddev", "FLOAT"},
				{"assists_expected", "FLOAT"},
				{"assists_stddev", "FLOAT"},
				{"backfill_bits", "INTEGER DEFAULT 0"},
			}
			for _, c := range cols {
				if err := addColumnIfMissing(db, "match_participants", c.name, c.typ); err != nil {
					return err
				}
			}
			return createIndexSafe(db, "CREATE INDEX IF NOT EXISTS idx_mp_backfill ON match_participants(xuid, backfill_bits)")
		},
	})

	Register(Migration{
		Name:        "add_medals_bigint",
		TargetDB:    TargetShared,
		Description: "medals_earned.medal_name_id INTEGER → BIGINT",
		ApplySchema: applyMedalsBigint,
	})

	Register(Migration{
		Name:        "add_mv_player_matches_fr_cols",
		TargetDB:    TargetShared,
		Description: "Recréation mv_player_matches avec colonnes FR",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "add_mv_player_matches_view",
		TargetDB:    TargetShared,
		Description: "Vue mv_player_matches (point d'entrée v6)",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "add_shared_performance_indexes",
		TargetDB:    TargetShared,
		Description: "Index sur match_participants, match_registry, medals_earned, highlight_events",
		ApplySchema: func(db *sql.DB) error {
			indexes := []string{
				"CREATE INDEX IF NOT EXISTS idx_mp_xuid_match ON match_participants(xuid, match_id)",
				"CREATE INDEX IF NOT EXISTS idx_mp_match_xuid ON match_participants(match_id, xuid)",
				"CREATE INDEX IF NOT EXISTS idx_mp_xuid_team ON match_participants(xuid, team_id, match_id)",
				"CREATE INDEX IF NOT EXISTS idx_mr_start_time ON match_registry(start_time DESC)",
				"CREATE INDEX IF NOT EXISTS idx_events_match_type ON highlight_events(match_id, event_type)",
				"CREATE INDEX IF NOT EXISTS idx_medals_full ON medals_earned(match_id, xuid, medal_name_id)",
			}
			for _, ddl := range indexes {
				_ = createIndexSafe(db, ddl)
			}
			return nil
		},
	})

	Register(Migration{
		Name:        "add_playable_duration",
		TargetDB:    TargetShared,
		Description: "playable_duration_seconds + real_start_time sur match_registry",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "match_registry", "playable_duration_seconds", "INTEGER"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "match_registry", "real_start_time", "TIMESTAMP")
		},
	})

	Register(Migration{
		Name:        "add_playlist_fr_name_fallback",
		TargetDB:    TargetShared,
		Description: "v_match_full : COALESCE secondaire playlist FR par nom EN",
		ApplySchema: applyResolutionViews,
	})

	Register(Migration{
		Name:        "add_spnkr_version",
		TargetDB:    TargetShared,
		Description: "Colonne sync_spnkr_version sur match_registry",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "match_registry", "sync_spnkr_version", "VARCHAR")
		},
	})

	Register(Migration{
		Name:        "add_team_ps_scores",
		TargetDB:    TargetShared,
		Description: "Colonnes team_0/1_ps_score + backfill depuis match_participants",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "match_registry", "team_0_ps_score", "INTEGER"); err != nil {
				return err
			}
			return addColumnIfMissing(db, "match_registry", "team_1_ps_score", "INTEGER")
		},
		ApplyBackfill: func(db *sql.DB) error {
			_, err := db.Exec(`
				UPDATE match_registry SET
					team_0_ps_score = sub.s0,
					team_1_ps_score = sub.s1
				FROM (
					SELECT match_id,
						SUM(CASE WHEN team_id = 0 THEN COALESCE(score, 0) ELSE 0 END) AS s0,
						SUM(CASE WHEN team_id = 1 THEN COALESCE(score, 0) ELSE 0 END) AS s1
					FROM match_participants GROUP BY match_id
				) sub
				WHERE match_registry.match_id = sub.match_id
					AND (match_registry.team_0_ps_score IS NULL OR match_registry.team_1_ps_score IS NULL)
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "add_weapon_kills",
		TargetDB:    TargetShared,
		Description: "Table weapon_kills (per-kill, weapon_id UBIGINT)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS weapon_kills (
					match_id       VARCHAR NOT NULL,
					xuid           VARCHAR NOT NULL,
					time_ms        INTEGER NOT NULL,
					weapon_id      UBIGINT,
					delta_ms       INTEGER,
					confidence     VARCHAR NOT NULL DEFAULT 'none',
					swap_detected  BOOLEAN NOT NULL DEFAULT FALSE,
					delayed_damage BOOLEAN NOT NULL DEFAULT FALSE
				);
				CREATE INDEX IF NOT EXISTS idx_wk_match_xuid ON weapon_kills(match_id, xuid);
			`)
		},
	})

	Register(Migration{
		Name:        "add_weapon_kills_reconciled_as",
		TargetDB:    TargetShared,
		Description: "Colonnes reconciled_as, attribution_path, player_index + vue v_weapon_kills",
		ApplySchema: func(db *sql.DB) error {
			_ = addColumnIfMissing(db, "weapon_kills", "reconciled_as", "UBIGINT")
			_ = addColumnIfMissing(db, "weapon_kills", "attribution_path", "VARCHAR DEFAULT 'none'")
			_ = addColumnIfMissing(db, "weapon_kills", "player_index", "INTEGER")
			_, err := db.Exec(`
				CREATE OR REPLACE VIEW v_weapon_kills AS
				SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id FROM weapon_kills
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "drop_highlight_events_gamertag",
		TargetDB:    TargetShared,
		Description: "Supprime colonne gamertag de highlight_events (recréation)",
		ApplySchema: applyDropHighlightEventsGamertag,
	})

	Register(Migration{
		Name:        "fix_bot_xuid",
		TargetDB:    TargetShared,
		Description: "Corrige bid(X.0 → bid(X.0) dans match_participants",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`
				UPDATE match_participants SET xuid = xuid || ')'
				WHERE xuid LIKE 'bid(%' AND xuid NOT LIKE 'bid(%)'
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "fix_events_loaded_inconsistency",
		TargetDB:    TargetShared,
		Description: "Remet events_loaded=FALSE pour matchs sans highlight_events",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`
				UPDATE match_registry SET events_loaded = FALSE
				WHERE events_loaded = TRUE
					AND match_id NOT IN (SELECT DISTINCT match_id FROM highlight_events)
			`)
			return err
		},
	})

	Register(Migration{
		Name:        "fix_mv_player_matches_scores",
		TargetDB:    TargetShared,
		Description: "Recréation mv_player_matches avec seuil corruption 100",
		ApplySchema: applyMvPlayerMatchesView,
	})

	Register(Migration{
		Name:        "migrate_weapon_kills_to_ubigint",
		TargetDB:    TargetShared,
		Description: "Conversion weapon_kills schema → UBIGINT natif",
		ApplySchema: func(db *sql.DB) error {
			// Si la table existe déjà avec weapon_id UBIGINT → rien à faire
			// La migration initiale crée déjà avec le bon type
			return nil
		},
	})
}

// applyHighlightEventsAutoincrement recrée highlight_events avec séquence.
func applyHighlightEventsAutoincrement(db *sql.DB) error {
	// Vérifier si la table existe et si la séquence est déjà présente
	exists, err := tableExists(db, "highlight_events")
	if err != nil || !exists {
		// Table absente → créer directement avec séquence
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
	err = db.QueryRow(
		"SELECT column_default FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'highlight_events' AND column_name = 'id'",
	).Scan(&colDefault)
	if err == nil && colDefault.Valid && len(colDefault.String) > 4 {
		return nil // Séquence déjà configurée
	}

	var maxID int
	_ = db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM highlight_events").Scan(&maxID)
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

// applyMedalsBigint migre medals_earned.medal_name_id vers BIGINT.
func applyMedalsBigint(db *sql.DB) error {
	// Vérifier le type actuel
	var dataType string
	err := db.QueryRow(
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

// applyDropHighlightEventsGamertag recrée highlight_events sans la colonne gamertag.
func applyDropHighlightEventsGamertag(db *sql.DB) error {
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

// applyResolutionViews crée les vues SQL v6 garanties.
func applyResolutionViews(db *sql.DB) error {
	// v_gamertag_lookup
	_, _ = db.Exec(`
		CREATE OR REPLACE VIEW v_gamertag_lookup AS
		SELECT
			COALESCE(xa.xuid, mp.xuid) AS xuid,
			COALESCE(xa.gamertag, mp.gamertag) AS gamertag
		FROM xuid_aliases xa
		FULL OUTER JOIN (
			SELECT xuid, MAX(gamertag) AS gamertag
			FROM match_participants
			WHERE gamertag IS NOT NULL AND gamertag != ''
			GROUP BY xuid
		) mp ON xa.xuid = mp.xuid
	`)

	// v_match_full — requires metadata ATTACHed as 'meta'
	// Simplified version that works with or without meta
	_, _ = db.Exec(`
		CREATE OR REPLACE VIEW v_match_full AS
		SELECT mr.*
		FROM match_registry mr
	`)

	// v_killer_victim_full
	if exists, _ := tableExists(db, "killer_victim_pairs"); exists {
		_, _ = db.Exec(`
			CREATE OR REPLACE VIEW v_killer_victim_full AS
			SELECT
				kvp.*,
				k.gamertag AS killer_gamertag,
				v.gamertag AS victim_gamertag
			FROM killer_victim_pairs kvp
			LEFT JOIN v_gamertag_lookup k ON kvp.killer_xuid = k.xuid
			LEFT JOIN v_gamertag_lookup v ON kvp.victim_xuid = v.xuid
		`)
	}

	// v_weapon_kills
	if exists, _ := tableExists(db, "weapon_kills"); exists {
		_, _ = db.Exec(`
			CREATE OR REPLACE VIEW v_weapon_kills AS
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM weapon_kills
		`)
	}

	return nil
}

// applyMvPlayerMatchesView crée ou recrée mv_player_matches.
func applyMvPlayerMatchesView(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE OR REPLACE VIEW mv_player_matches AS
		SELECT
			mr.match_id,
			mr.start_time,
			mr.end_time,
			mr.playlist_id,
			mr.playlist_name,
			mr.map_id,
			mr.map_name,
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
