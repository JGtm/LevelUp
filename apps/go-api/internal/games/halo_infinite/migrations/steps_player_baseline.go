package migrations

// steps_player_baseline.go — BASELINE SQUASHÉE du bloc contigu player title-owned
// (chantier squash N4, plan PLAN_MIGRATION_SQUASH_BASELINE_2026-07, M3).
//
// `create_baseline_player_v1` REMPLACE, sur une DB VIERGE, les 33 steps title-owned
// CONTIGUS allant de create_base_player_schema à player_append_only_csr_snapshots_v1
// (bornes figées M3a, sur pièces : le 1er step GLOBAL suivant est
// player_append_only_match_citations_v1 — la baseline NE traverse PAS la frontière
// global→title, DM-4). Elle émet le schéma « à plat » : EXACTEMENT l'état CUMULÉ des
// 33 steps (schéma seul — les steps de données/drop du bloc sont sans objet sur une
// DB vierge : media_files était créée puis droppée → absente ici).
//
// GARANTIE ZÉRO-PERTE : squash_invariant_test.go prouve, à chaque CI intégration, que
// SchemaSnapshot(baseline) == SchemaSnapshot(33 steps historiques) octet pour octet
// (golden testdata/squash/player_block_golden.snapshot, capturé avant le squash).
//
// DM-5 (équivalence ledger) : SupersededByAll liste les 33 steps ; une DB EXISTANTE
// qui les a déjà appliqués (sentinelle = le dernier, player_append_only_csr_snapshots_v1,
// présent dans schema_migrations) est réputée porter la baseline → le runner
// l'enregistre SANS rejouer le DDL (aucune régression sur les player DB en prod).
//
// Les 33 sources squashées sont archivées dans .ai/migrations/squashed/player_v1/
// (DM-3) ; git conserve l'historique complet.

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// BaselinePlayerV1Name est le nom du step baseline (exporté : référencé par le test
// d'invariant et la règle d'équivalence ledger).
const BaselinePlayerV1Name = "create_baseline_player_v1"

// squashedPlayerV1Steps est la liste ORDONNÉE des 33 steps squashés par la baseline
// v1 (bornes M3a). Sert de sentinelle DM-5 (le dernier = présence ⇒ baseline
// satisfaite) et de spec d'archivage. Ordre = canonicalOrder (order.go).
var squashedPlayerV1Steps = []string{
	"create_base_player_schema",
	"add_engagement_score_columns_to_player_match_enrichment",
	"create_engagement_coefficients_table",
	"repair_engagement_coefficients_primary_key",
	"add_engagement_pace_columns_to_player_match_enrichment",
	"create_engagement_response_bins_table",
	"add_bot_teammate_column",
	"add_career_progression_sequence",
	"add_career_identity_assets",
	"add_career_banner_image",
	"add_career_last_fetch_status",
	"add_challenge_snapshots",
	"add_challenge_snapshots_render_columns",
	"add_challenge_snapshots_display_path",
	"add_battlepass_snapshots",
	"add_dominance_flag_column",
	"add_media_like_columns",
	"add_media_capture_start_utc",
	"add_performance_score",
	"add_player_performance_indexes",
	"add_pme_session_label",
	"add_pme_session_index",
	"add_skill_rating_table",
	"fix_mv_session_stats_varchar",
	"add_match_exclusion_flag",
	"add_player_privacy_state",
	"drop_media_from_player_db",
	"add_player_achievements",
	"fix_match_citations_schema",
	"cleanup_spartan_customization_garbage_urls",
	"add_msr_measurement_matches_remaining",
	"player_add_expected_win_prob",
	"player_append_only_csr_snapshots_v1",
}

// playerBaselineSteps retourne le step baseline v1 (racine squashée du tier player).
// Positionné en tête du bloc player dans canonicalOrder (à la place de
// create_base_player_schema).
func playerBaselineSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:            BaselinePlayerV1Name,
			TargetDB:        migration.TargetPlayer,
			Description:     "Baseline squashée v1 du bloc player title-owned (33 steps create_base_player_schema..player_append_only_csr_snapshots_v1)",
			SupersededByAll: squashedPlayerV1Steps,
			ApplySchema:     applyBaselinePlayerV1,
			// DM-5 heal (M3/M4) : rejoue le même ensure additif idempotent sur une DB
			// porteuse de la sentinelle mais au schéma partiel (expected_win_prob,
			// colonnes render challenge_snapshots, engagement_response_bins).
			EnsureAdditive: ensureBaselinePlayerV1AdditiveColumns,
		},
	}
}

// applyBaselinePlayerV1 émet le schéma cumulé « à plat » du bloc squashé, PUIS garantit
// (AddColumnIfMissing) les colonnes que les steps historiques du bloc ajoutaient par
// ALTER — indispensable car certaines tables (match_skill_rank, player_match_enrichment,
// career_progression) peuvent PRÉ-EXISTER avec un schéma partiel quand sync EnsureSchema
// (ou un bootstrap de test) les a créées AVANT le cycle de migrations. Le `CREATE TABLE
// IF NOT EXISTS` seul no-operait alors et laissait ces colonnes manquantes (contrat
// idempotent-additif du système de migrations, préservé). Sur DB vierge, le CREATE crée
// tout et les AddColumnIfMissing sont des no-op → schéma == golden (invariant préservé).
func applyBaselinePlayerV1(db *sql.DB) error {
	if err := applyBaselinePlayerV1Tables(db); err != nil {
		return err
	}
	return ensureBaselinePlayerV1AdditiveColumns(db)
}

// Fragments DDL et noms de tables réutilisés par la liste d'ensure ci-dessous.
// Factorisés en constantes pour éviter la répétition de littéraux (mots-clés SQL courts
// + noms de tables fréquents dans le package) — sans quoi goconst bruite sur des
// littéraux triviaux. Exemption ciblée.
//
//nolint:goconst
const (
	tblMSR                = "match_skill_rank"
	tblPME                = "player_match_enrichment"
	tblCareer             = "career_progression"
	tblChallenge          = "challenge_snapshots"
	tyFloat               = "FLOAT"
	tyVarchar             = "VARCHAR"
	tyDouble              = "DOUBLE"
	tyInteger             = "INTEGER"
	tyVarcharEmptyDefault = "VARCHAR DEFAULT ''"
)

// ddlEngagementResponseBins — DDL UNIQUE de engagement_response_bins (create_engagement_
// response_bins_table). Source unique partagée par la baseline (fresh DB) ET le heal
// DM-5 (ensureBaselinePlayerV1AdditiveColumns) : CREATE IF NOT EXISTS idempotent, donc
// no-op sur une DB qui la porte déjà. Centralisé pour qu'un seul schéma fasse foi (pas
// de divergence entre le chemin création et le chemin heal).
const ddlEngagementResponseBins = `
	CREATE TABLE IF NOT EXISTS engagement_response_bins (
		xuid VARCHAR NOT NULL,
		mode_category VARCHAR NOT NULL,
		intensity_bin VARCHAR NOT NULL,
		lower_bound DOUBLE NOT NULL,
		upper_bound DOUBLE NOT NULL,
		coef_lobby DOUBLE NOT NULL,
		n_matches INTEGER NOT NULL,
		last_updated TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		PRIMARY KEY (xuid, mode_category, intensity_bin)
	);`

// ensureBaselinePlayerV1AdditiveColumns reproduit les ALTER idempotents du bloc squashé
// sur les tables susceptibles de pré-exister partiellement (E7) et garantit la table
// engagement_response_bins. No-op sur DB vierge (colonnes déjà émises par le CREATE, table
// créée ici). Rejoué sur le chemin DM-5 (heal M3/M4) pour une DB porteuse de la sentinelle
// au schéma partiel — les tables ciblées existent alors déjà (steps squashés appliqués).
func ensureBaselinePlayerV1AdditiveColumns(db *sql.DB) error {
	cols := []struct{ table, name, typ string }{
		// match_skill_rank : ALTER de add_skill_rating_table / add_msr_measurement_matches_remaining
		// / player_add_expected_win_prob (+ created_at/updated_at absents d'un bootstrap minimal).
		{tblMSR, "rating_deviation", tyFloat},
		{tblMSR, "playlist_group", tyVarchar},
		{tblMSR, "start_time", "TIMESTAMP"},
		{tblMSR, "created_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"},
		{tblMSR, "updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"},
		{tblMSR, "measurement_matches_remaining", "INTEGER DEFAULT 0"},
		{tblMSR, "expected_win_prob", tyFloat},
		// player_match_enrichment : ALTER engagement / bot / dominance / exclusion / session.
		{tblPME, "session_label", tyVarchar},
		{tblPME, "engagement_score", tyDouble},
		{tblPME, "engagement_score_brut", tyDouble},
		{tblPME, "engagement_score_confidence", tyVarchar},
		{tblPME, "mode_category", tyVarchar},
		{tblPME, "engagement_pace_player", tyDouble},
		{tblPME, "engagement_pace_team", tyDouble},
		{tblPME, "engagement_pace_lobby", tyDouble},
		{tblPME, "engagement_player_activity", tyInteger},
		{tblPME, "had_bot_teammate", "BOOLEAN"},
		{tblPME, "dominance_flag", "TINYINT"},
		{tblPME, "is_excluded", "BOOLEAN DEFAULT FALSE"},
		// career_progression : ALTER identity assets / banner / last_fetch_status.
		{tblCareer, "adornment_path", tyVarcharEmptyDefault},
		{tblCareer, "spartan_id", tyVarcharEmptyDefault},
		{tblCareer, "banner_image_url", tyVarcharEmptyDefault},
		{tblCareer, "emblem_image_url", tyVarcharEmptyDefault},
		{tblCareer, "backdrop_image_url", tyVarcharEmptyDefault},
		{tblCareer, "last_fetch_status", tyVarchar},
		// challenge_snapshots : ALTER de add_challenge_snapshots_render_columns (title /
		// description / image_url, 06-22) + add_challenge_snapshots_display_path — absents
		// des DBs dont la sentinelle précède ces steps (skew M4). Sans eux, INSERT challenges
		// casse (colonnes render manquantes).
		{tblChallenge, "title", tyVarchar},
		{tblChallenge, "description", tyVarchar},
		{tblChallenge, "image_url", tyVarchar},
		{tblChallenge, "display_path", tyVarchar},
	}
	for _, c := range cols {
		if err := migration.AddColumnIfMissing(db, c.table, c.name, c.typ); err != nil {
			return fmt.Errorf("baseline player v1: ensure %s.%s: %w", c.table, c.name, err)
		}
	}
	// engagement_response_bins : create_engagement_response_bins_table. CREATE IF NOT
	// EXISTS (source unique ddlEngagementResponseBins) → couvre à la fois la création sur
	// DB vierge et le heal DM-5 d'une DB porteuse de la sentinelle mais sans la table.
	if err := migration.ExecScript(db, ddlEngagementResponseBins); err != nil {
		return fmt.Errorf("baseline player v1: ensure engagement_response_bins: %w", err)
	}
	return nil
}

// applyBaselinePlayerV1Tables émet les CREATE « à plat » (schéma cumulé du bloc). L'ordre
// des colonnes est POSITIONNEL (observable) et reproduit l'ordre d'ajout historique.
func applyBaselinePlayerV1Tables(db *sql.DB) error {
	return migration.ExecScript(db, `
		-- player_match_enrichment (PK match_id) : base + engagement + bot + dominance + exclusion.
		CREATE TABLE IF NOT EXISTS player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			session_id VARCHAR,
			session_label VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			teammates_signature VARCHAR,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			engagement_score DOUBLE,
			engagement_score_brut DOUBLE,
			engagement_score_confidence VARCHAR,
			mode_category VARCHAR,
			engagement_pace_player DOUBLE,
			engagement_pace_team DOUBLE,
			engagement_pace_lobby DOUBLE,
			engagement_player_activity INTEGER,
			had_bot_teammate BOOLEAN,
			dominance_flag TINYINT,
			is_excluded BOOLEAN DEFAULT FALSE
		);

		-- sync_meta (PK key).
		CREATE TABLE IF NOT EXISTS sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR,
			updated_at TIMESTAMP
		);

		-- career_progression : PAS de PK/id sur DB vierge (add_career_progression_sequence
		-- crée la SÉQUENCE mais son CREATE TABLE IF NOT EXISTS no-ope sur la table existante).
		CREATE SEQUENCE IF NOT EXISTS career_progression_id_seq START 1;
		CREATE TABLE IF NOT EXISTS career_progression (
			xuid VARCHAR,
			rank INTEGER,
			rank_name VARCHAR,
			rank_tier VARCHAR,
			current_xp INTEGER,
			xp_for_next_rank INTEGER,
			xp_total INTEGER,
			is_max_rank BOOLEAN DEFAULT FALSE,
			adornment_path VARCHAR DEFAULT '',
			spartan_id VARCHAR DEFAULT '',
			banner_image_url VARCHAR DEFAULT '',
			emblem_image_url VARCHAR DEFAULT '',
			backdrop_image_url VARCHAR DEFAULT '',
			recorded_at TIMESTAMP,
			last_fetch_status VARCHAR
		);
		CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid);

		-- sessions (PK session_id).
		CREATE TABLE IF NOT EXISTS sessions (
			session_id INTEGER PRIMARY KEY,
			label VARCHAR,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			match_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);

		-- match_citations (PK match_id, citation_name_norm).
		CREATE TABLE IF NOT EXISTS match_citations (
			match_id VARCHAR NOT NULL,
			citation_name_norm VARCHAR NOT NULL,
			value INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (match_id, citation_name_norm)
		);

		-- skill_history (sans PK).
		CREATE TABLE IF NOT EXISTS skill_history (
			playlist_id VARCHAR,
			recorded_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			csr INTEGER,
			tier VARCHAR,
			division INTEGER,
			matches_played INTEGER
		);

		-- match_skill_rank (PK match_id) : base + measurement_matches_remaining + expected_win_prob.
		CREATE TABLE IF NOT EXISTS match_skill_rank (
			match_id VARCHAR PRIMARY KEY,
			rating_type VARCHAR NOT NULL,
			rating_value FLOAT,
			rating_deviation FLOAT,
			tier VARCHAR,
			tier_fr VARCHAR,
			sub_tier SMALLINT DEFAULT 0,
			tier_label VARCHAR,
			rating_delta FLOAT,
			playlist_group VARCHAR,
			start_time TIMESTAMP,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			measurement_matches_remaining INTEGER DEFAULT 0,
			expected_win_prob FLOAT
		);
		CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type);
		CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group);

		-- challenge_snapshots (sans PK) : base + render columns + display_path.
		CREATE TABLE IF NOT EXISTS challenge_snapshots (
			snapshot_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			xuid VARCHAR NOT NULL,
			challenge_path VARCHAR NOT NULL,
			challenge_id VARCHAR,
			content_hash VARCHAR,
			status VARCHAR NOT NULL,
			progress_current INTEGER,
			progress_target INTEGER,
			xp_reward INTEGER DEFAULT 0,
			can_reroll BOOLEAN,
			expires_at TIMESTAMP,
			deck_index INTEGER,
			state_hash VARCHAR NOT NULL,
			title VARCHAR,
			description VARCHAR,
			image_url VARCHAR,
			display_path VARCHAR
		);
		CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_xuid_time ON challenge_snapshots(xuid, snapshot_at DESC);
		CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_path_time ON challenge_snapshots(challenge_path, snapshot_at DESC);

		-- battlepass_snapshots (sans PK).
		CREATE TABLE IF NOT EXISTS battlepass_snapshots (
			snapshot_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			xuid VARCHAR NOT NULL,
			reward_track_path VARCHAR NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			current_rank INTEGER NOT NULL DEFAULT 0,
			partial_progress INTEGER NOT NULL DEFAULT 0,
			is_owned BOOLEAN NOT NULL DEFAULT FALSE,
			has_reached_max_rank BOOLEAN NOT NULL DEFAULT FALSE,
			base_xp INTEGER NOT NULL DEFAULT 0,
			boost_xp INTEGER NOT NULL DEFAULT 0,
			state_hash VARCHAR NOT NULL,
			raw_payload_json VARCHAR NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_battlepass_snapshots_xuid_time ON battlepass_snapshots(xuid, snapshot_at DESC);
		CREATE INDEX IF NOT EXISTS idx_battlepass_snapshots_track_time ON battlepass_snapshots(reward_track_path, snapshot_at DESC);

		-- player_privacy_state (PK xuid).
		CREATE TABLE IF NOT EXISTS player_privacy_state (
			xuid VARCHAR PRIMARY KEY,
			is_private BOOLEAN NOT NULL DEFAULT FALSE,
			observed_at TIMESTAMP NOT NULL,
			source VARCHAR NOT NULL DEFAULT 'waypoint'
		);

		-- player_achievements (PK achievement_id).
		CREATE TABLE IF NOT EXISTS player_achievements (
			achievement_id VARCHAR PRIMARY KEY,
			unlocked BOOLEAN NOT NULL DEFAULT FALSE,
			unlocked_at TIMESTAMP,
			current_progress INTEGER,
			target_progress INTEGER,
			fetched_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);

		-- engagement_coefficients (PK xuid, mode_category).
		CREATE TABLE IF NOT EXISTS engagement_coefficients (
			xuid VARCHAR NOT NULL,
			mode_category VARCHAR NOT NULL,
			coef_team_share DOUBLE NOT NULL,
			coef_lobby_share DOUBLE NOT NULL,
			n_matches INTEGER NOT NULL,
			last_updated TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (xuid, mode_category)
		);
		-- engagement_response_bins : émise par ensureBaselinePlayerV1AdditiveColumns
		-- (source unique ddlEngagementResponseBins, partagée avec le heal DM-5).
	`)
}
