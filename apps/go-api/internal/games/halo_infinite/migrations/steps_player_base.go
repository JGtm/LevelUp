package migrations

// steps_player_base.go — schémas prestige / campaign / progression du tier player
// (stats.duckdb), title-owned (Phase 1.5 b25, voie B).
//
// SQUASH v1 (chantier N4, 2026-07-12) : les 25 steps du god-file de base
// (create_base_player_schema … add_msr_measurement_matches_remaining) qui vivaient ici
// sont désormais SQUASHÉS dans create_baseline_player_v1 (steps_player_baseline.go) —
// leur schéma cumulé « à plat » est émis par la baseline sur une DB vierge. Restent ici
// les racines POSTÉRIEURES à la borne du squash : create_prestige_player_schema,
// create_arc_titles_join + drop_arc_titles, create_improvement_campaign_schema,
// create_progression_player_schema, drop_notifications_from_player_db.
// Les 3 helpers alors dédiés au bloc squashé
// (applyCareerProgressionSequence, *IdentityAssets, applyFixMvSessionStats) sont retirés
// avec lui (archivés dans .ai/migrations/squashed/player_v1/).

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// playerBaseSteps retourne la racine player title-owned (b25).
func playerBaseSteps() []migration.Migration {
	return []migration.Migration{
		// ─── challenge_snapshots.locale : ALTER post-baseline additif ───
		// Mirror du (squashé) add_challenge_snapshots_display_path : nouvelle colonne
		// additive sur challenge_snapshots (table créée par la baseline). NON squashée —
		// posée en step vivant post-baseline pour ne pas muter le golden de la baseline
		// (squash_invariant_test.go : SchemaSnapshot(baseline) == historique 33 steps).
		// La colonne porte la LANGUE des titres/descriptions résolus persistés : le
		// state_hash étant langue-indépendant, sans cette dimension le 2e insert (autre
		// locale) est dédupliqué et seule la dernière langue fetchée survit.
		{
			Name:        "add_challenge_snapshots_locale",
			TargetDB:    migration.TargetPlayer,
			Description: "Ajoute challenge_snapshots.locale (langue des libellés résolus) — sépare les snapshots par locale (dédup state_hash langue-indépendant sinon).",
			ApplySchema: func(db *sql.DB) error {
				return migration.AddColumnIfMissing(db, tblChallenge, "locale", tyVarcharEmptyDefault)
			},
		},
		// ─── schémas prestige / campaign / progression (créateurs des tables ALTERées par b15-b22) ───
		{
			Name:        "create_prestige_player_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Tables Prestige côté joueur (arc, challenge, moment_card, telemetry, baseline_state)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS arc (
						id            VARCHAR PRIMARY KEY,
						user_id       VARCHAR NOT NULL,
						title_slug    VARCHAR NOT NULL,
						title         VARCHAR NOT NULL,
						description   VARCHAR,
						is_preset     BOOLEAN NOT NULL DEFAULT FALSE,
						preset_id     VARCHAR,
						created_at    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
						completed_at  TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_arc_user_title ON arc(user_id, title_slug);

					CREATE TABLE IF NOT EXISTS challenge (
						id                          VARCHAR PRIMARY KEY,
						user_id                     VARCHAR NOT NULL,
						title_slug                  VARCHAR NOT NULL,
						arc_id                      VARCHAR,
						position                    INTEGER,
						template_id                 VARCHAR,
						metric                      VARCHAR NOT NULL,
						target                      DOUBLE NOT NULL,
						target_per_member           DOUBLE,
						window_type                 VARCHAR NOT NULL,
						window_value                VARCHAR,
						cadence                     VARCHAR NOT NULL DEFAULT 'free',
						eval_type                   VARCHAR NOT NULL DEFAULT 'threshold',
						mode                        VARCHAR NOT NULL DEFAULT 'libre',
						tier                        VARCHAR,
						data_tier                   VARCHAR NOT NULL DEFAULT 'full',
						label                       VARCHAR,
						status                      VARCHAR NOT NULL DEFAULT 'draft',
						created_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
						committed_at                TIMESTAMP,
						completed_at                TIMESTAMP,
						expired_at                  TIMESTAMP,
						abandoned_at                TIMESTAMP,
						last_palier_recompute_at    TIMESTAMP,
						is_private                  BOOLEAN DEFAULT FALSE
					);
					-- PAS d'index sur status ni arc_id : UpdateStatus mute status,
					-- DetachFromArc mute arc_id → un index ART sur une colonne mutée corrompt
					-- la player DB (cf. crash match_skill_rank mono-writer). Drop sur DB
					-- existantes : drop_challenge_mutated_art_indexes_v1. metric jamais muté → OK.
					CREATE INDEX IF NOT EXISTS idx_ch_metric ON challenge(metric);

					CREATE TABLE IF NOT EXISTS moment_card (
						id            VARCHAR PRIMARY KEY,
						challenge_id  VARCHAR NOT NULL,
						blob_path     VARCHAR,
						created_at    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
					);
					CREATE INDEX IF NOT EXISTS idx_mc_challenge ON moment_card(challenge_id);

					CREATE TABLE IF NOT EXISTS prestige_telemetry (
						id                          VARCHAR PRIMARY KEY,
						user_id                     VARCHAR NOT NULL,
						challenge_id                VARCHAR,
						event_type                  VARCHAR NOT NULL,
						palier                      VARCHAR,
						stretch_ratio               DOUBLE,
						baseline_value              DOUBLE,
						mode                        VARCHAR,
						cadence                     VARCHAR,
						eval_type                   VARCHAR,
						time_since_create_seconds   INTEGER,
						created_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
					);
					CREATE INDEX IF NOT EXISTS idx_pt_user_event ON prestige_telemetry(user_id, event_type);
					CREATE INDEX IF NOT EXISTS idx_pt_challenge ON prestige_telemetry(challenge_id);
					CREATE INDEX IF NOT EXISTS idx_pt_created ON prestige_telemetry(created_at);

					CREATE TABLE IF NOT EXISTS baseline_state (
						user_id                     VARCHAR NOT NULL,
						title_slug                  VARCHAR NOT NULL,
						metric                      VARCHAR NOT NULL,
						last_match_at               TIMESTAMP,
						is_stale                    BOOLEAN NOT NULL DEFAULT FALSE,
						recovery_matches_remaining  INTEGER NOT NULL DEFAULT 0,
						updated_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
						PRIMARY KEY (user_id, title_slug, metric)
					);
				`)
			},
		},
		// ─── arc_titles : structure RETIRÉE (décision produit 2026-07-18) ───
		// Le couple create_arc_titles_join + drop_arc_titles ci-dessous est
		// VOLONTAIREMENT conservé en l'état : `create_arc_titles_join` est un
		// enregistrement présent dans schema_migrations sur toutes les bases joueur
		// existantes ET une entrée de canonicalOrder — le supprimer désynchroniserait
		// le registre (order_audit_test.go, audit bidirectionnel « pas d'entrée morte »).
		// Sur une base VIERGE la table est créée puis immédiatement droppée dans le même
		// cycle : effet net nul. NE PAS « nettoyer » cette paire.
		{
			Name:     "create_arc_titles_join",
			TargetDB: migration.TargetPlayer,
			Description: "HISTORIQUE (2026-06-18) — table de jointure arc_titles(arc_id, title_slug). " +
				"Structure retirée le 2026-07-25 par drop_arc_titles (décision produit 2026-07-18 : " +
				"arcs, défis et points de progression strictement indépendants par titre). Entrée " +
				"conservée pour la cohérence du ledger schema_migrations des bases existantes.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS arc_titles (
						arc_id      VARCHAR NOT NULL,
						title_slug  VARCHAR NOT NULL,
						PRIMARY KEY (arc_id, title_slug)
					);
					CREATE INDEX IF NOT EXISTS idx_arc_titles_slug ON arc_titles(title_slug);
				`)
			},
			// ApplyBackfill RETIRÉ le 2026-07-25 : il remplissait arc_titles depuis arc
			// (miroir 1:1). La table étant droppée dans le même cycle, un rejeu du backfill
			// (cas backfill_done = FALSE) taperait une table absente et logguerait un WARN à
			// chaque boot. ApplyBackfill == nil ⇒ applyStepBackfill no-ope (registry.go).
		},
		{
			Name:     "drop_arc_titles",
			TargetDB: migration.TargetPlayer,
			Description: "Supprime arc_titles de stats.duckdb — retrait de la capacité multi-titres d'arc " +
				"(décision produit 2026-07-18 : arcs, défis et points de progression strictement " +
				"indépendants par titre ; cf. .ai/archive/PLAN_CROSS_TITLE_ARCS_2026-07.md).",
			ApplySchema: func(db *sql.DB) error {
				// RAISON (décision produit du 2026-07-18, Option A confirmée par l'utilisateur) :
				// arc_titles était une jointure many-to-many arc↔titre livrée le 2026-06-18
				// (PLAN_CROSS_TITLE_ARCS_BACKEND) qui rendait REPRÉSENTABLE un arc couvrant N
				// titres. La décision produit interdit cette notion : un arc porte UN titre et un
				// seul — `arc.title_slug` est la source unique.
				// ZÉRO PERTE DE DONNÉE : la table était un miroir 1:1 de arc.title_slug (backfill
				// d'origine = `SELECT id, title_slug FROM arc` ; unique writer applicatif =
				// PrestigeArcRepo.Create, qui insérait (a.ID, a.TitleSlug)). Aucune ligne n'a
				// jamais pu porter un titre absent de l'arc correspondant.
				// IRRÉVERSIBLE côté données, mais reconstructible à l'identique depuis `arc`.
				return migration.ExecScript(db, `
					DROP TABLE IF EXISTS arc_titles;
				`)
			},
		},
		{
			Name:        "create_improvement_campaign_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Table improvement_campaign + challenge.campaign_id (V1 PlayerProfile §4.5)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS improvement_campaign (
						id                       VARCHAR PRIMARY KEY,
						user_id                  VARCHAR NOT NULL,
						title_slug               VARCHAR NOT NULL,
						axis                     VARCHAR NOT NULL,
						axis_kind                VARCHAR NOT NULL,
						started_at               TIMESTAMP NOT NULL,
						ended_at                 TIMESTAMP,
						status                   VARCHAR NOT NULL DEFAULT 'active',
						playlist_group           VARCHAR NOT NULL DEFAULT 'all',
						snapshot_value           DOUBLE NOT NULL,
						snapshot_sample          INTEGER NOT NULL DEFAULT 0,
						current_value_raw        DOUBLE,
						current_value_lowess     DOUBLE,
						matches_since_start      INTEGER NOT NULL DEFAULT 0,
						last_evaluated_at        TIMESTAMP,
						mann_whitney_p           DOUBLE,
						progression_confirmed    BOOLEAN NOT NULL DEFAULT FALSE,
						auto_closure_suggested   BOOLEAN NOT NULL DEFAULT FALSE,
						auto_closure_reason      VARCHAR
					);
					CREATE INDEX IF NOT EXISTS idx_campaign_user_title ON improvement_campaign(user_id, title_slug, status);

					-- PAS d'index sur campaign_id : campaign_repo UPDATE challenge SET
					-- campaign_id = … → index ART sur colonne mutée = corrupteur (#23046).
					-- Drop sur DB existantes : drop_challenge_mutated_art_indexes_v1.
					ALTER TABLE challenge ADD COLUMN IF NOT EXISTS campaign_id VARCHAR;
				`)
			},
		},
		{
			Name:        "create_progression_player_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Tables Progression Tracking côté joueur (streak, record_history, milestone_earned)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS streak (
						id                  VARCHAR PRIMARY KEY,
						user_id             VARCHAR NOT NULL,
						title_slug          VARCHAR NOT NULL,
						type                VARCHAR NOT NULL,
						started_at          TIMESTAMP NOT NULL,
						current_length      INTEGER NOT NULL DEFAULT 0,
						best_length         INTEGER NOT NULL DEFAULT 0,
						last_increment_at   TIMESTAMP,
						threshold           DOUBLE,
						shields_used        INTEGER NOT NULL DEFAULT 0,
						shields_available   INTEGER NOT NULL DEFAULT 1,
						status              VARCHAR NOT NULL DEFAULT 'active',
						broken_at           TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_streak_user_title_type ON streak(user_id, title_slug, type);
					CREATE INDEX IF NOT EXISTS idx_streak_user_status     ON streak(user_id, status);

					CREATE TABLE IF NOT EXISTS record_history (
						id           VARCHAR PRIMARY KEY,
						user_id      VARCHAR NOT NULL,
						title_slug   VARCHAR NOT NULL,
						metric       VARCHAR NOT NULL,
						period       VARCHAR NOT NULL,
						value        DOUBLE NOT NULL,
						achieved_at  TIMESTAMP NOT NULL
					);
					CREATE INDEX IF NOT EXISTS idx_rec_hist_user_title_metric ON record_history(user_id, title_slug, metric);
					CREATE INDEX IF NOT EXISTS idx_rec_hist_achieved_desc     ON record_history(user_id, achieved_at DESC);

					CREATE TABLE IF NOT EXISTS milestone_earned (
						user_id      VARCHAR NOT NULL,
						title_slug   VARCHAR NOT NULL,
						milestone_id VARCHAR NOT NULL,
						earned_at    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
						PRIMARY KEY (user_id, title_slug, milestone_id)
					);
					CREATE INDEX IF NOT EXISTS idx_ms_earned_user_title ON milestone_earned(user_id, title_slug);
				`)
			},
		},
		{
			Name:        "drop_notifications_from_player_db",
			TargetDB:    migration.TargetPlayer,
			Description: "Supprime player_notifications, notification_preferences, player_records de stats.duckdb (déplacés dans la base sociale).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					DROP TABLE IF EXISTS player_notifications;
					DROP TABLE IF EXISTS notification_preferences;
					DROP TABLE IF EXISTS player_records;
				`)
			},
		},
	}
}
