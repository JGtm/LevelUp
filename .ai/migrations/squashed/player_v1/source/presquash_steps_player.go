package migrations

// steps_player.go — migrations DDL ciblant stats.duckdb (par joueur), CONSOMMATRICES
// du schéma de base, déplacées depuis internal/migration/steps_player_*.go
// (Phase 1.5 b15, voie B).
//
// IMPORTANT — direction de relocation : seuls des CONSOMMATEURS sont déplacés ici.
// Le SCHÉMA DE BASE player (create_base_player_schema, add_skill_rating_table,
// create_prestige_player_schema, create_progression_player_schema, …) reste dans le
// package migration (global) tant que TOUS ses consommateurs + leurs tests ne sont
// pas title-owned : c'est la RACINE du tier player (comme add_asset_translations pour
// metadata). Déplacer un créateur en laissant un consommateur global casse les runs
// global-only (RunForDB combine global+title, mais les tests du package migration
// tournent sans provider). Les 3 steps ci-dessous ALTERnent des tables créées par le
// schéma de base (player_match_enrichment, career_progression) qui restent globales →
// safe dans les 2 contextes.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/migration"
)

// playerSteps retourne les migrations player title-owned (consommateurs, b15).
func playerSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:     "player_match_enrichment_performance_chain_v1",
			TargetDB: migration.TargetPlayer,
			Description: "Ajout de la colonne performance_chain à player_match_enrichment" +
				" (découplage du score de performance par chaîne de playlist, 6 chaînes).",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(),
					`ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS performance_chain VARCHAR`,
				)
				return err
			},
		},
		{
			Name:     "player_match_enrichment_psa_checked_v1",
			TargetDB: migration.TargetPlayer,
			Description: "Ajout de la colonne psa_checked_at à player_match_enrichment" +
				" (marqueur terminal de la convergence des personal_score_awards).",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(),
					`ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS psa_checked_at TIMESTAMP`,
				)
				return err
			},
		},
		{
			Name:        "fix_career_xp_total_default_zero",
			TargetDB:    migration.TargetPlayer,
			Description: "Supprime DEFAULT 0 sur xp_total et corrige toutes les rows corrompues (xp_total=0 jamais légitime).",
			ApplySchema: applyFixCareerXPTotalDefault,
		},
		// ─── consommateurs des tables progression (record_history, streak) — b16.
		// record_history/streak sont créées par create_progression_player_schema
		// (RACINE, reste global) ; ces steps les dédoublonnent / migrent en append-only.
		{
			Name:        "dedup_record_history_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "Dédoublonne record_history (doublons de transition append-only records). Rebuild CTAS conditionnel, no-op si aucun doublon.",
			ApplySchema: func(db *sql.DB) error {
				has, err := migration.TableExists(db, "record_history")
				if err != nil {
					return fmt.Errorf("dedup_record_history: check table: %w", err)
				}
				if !has {
					return nil
				}

				var total, distinctKeys int
				if err := db.QueryRowContext(migration.BootCtx(), `SELECT COUNT(*) FROM record_history`).Scan(&total); err != nil {
					return fmt.Errorf("dedup_record_history: count total: %w", err)
				}
				if err := db.QueryRowContext(migration.BootCtx(), `
					SELECT COUNT(*) FROM (
						SELECT 1 FROM record_history
						GROUP BY user_id, title_slug, metric, period, value, achieved_at
					)
				`).Scan(&distinctKeys); err != nil {
					return fmt.Errorf("dedup_record_history: count distinct: %w", err)
				}
				if total == distinctKeys {
					return nil
				}

				if _, err := db.ExecContext(migration.BootCtx(), `
					CREATE TABLE record_history__dedup AS
					SELECT * FROM record_history
					QUALIFY ROW_NUMBER() OVER (
						PARTITION BY user_id, title_slug, metric, period, value, achieved_at
						ORDER BY id
					) = 1;
					DROP TABLE record_history;
					ALTER TABLE record_history__dedup RENAME TO record_history;
					ALTER TABLE record_history ADD PRIMARY KEY (id);
					CREATE INDEX IF NOT EXISTS idx_rec_hist_user_title_metric
						ON record_history(user_id, title_slug, metric);
					CREATE INDEX IF NOT EXISTS idx_rec_hist_achieved_desc
						ON record_history(achieved_at DESC);
				`); err != nil {
					return fmt.Errorf("dedup_record_history: rebuild: %w", err)
				}
				return nil
			},
		},
		{
			Name:        "create_streak_history_append_only",
			TargetDB:    migration.TargetPlayer,
			Description: "Pattern append-only pour streak : nouvelle table streak_history (INSERT pur) + vue streak_latest. Backfill depuis l'ancienne table streak.",
			ApplySchema: func(db *sql.DB) error {
				has, err := migration.TableExists(db, "streak_history")
				if err != nil {
					return fmt.Errorf("create_streak_history: check table: %w", err)
				}
				if has {
					return nil
				}

				if _, err := db.ExecContext(migration.BootCtx(), `
					CREATE SEQUENCE IF NOT EXISTS streak_history_tech_id_seq START 1;
					CREATE TABLE streak_history (
						tech_id           BIGINT PRIMARY KEY DEFAULT nextval('streak_history_tech_id_seq'),
						id                VARCHAR NOT NULL,
						user_id           VARCHAR NOT NULL,
						title_slug        VARCHAR NOT NULL,
						type              VARCHAR NOT NULL,
						started_at        TIMESTAMP NOT NULL,
						current_length    INTEGER NOT NULL DEFAULT 0,
						best_length       INTEGER NOT NULL DEFAULT 0,
						last_increment_at TIMESTAMP,
						threshold         DOUBLE,
						shields_used      INTEGER NOT NULL DEFAULT 0,
						shields_available INTEGER NOT NULL DEFAULT 1,
						status            VARCHAR NOT NULL DEFAULT 'active',
						broken_at         TIMESTAMP,
						written_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX idx_streak_history_lookup
						ON streak_history(id, written_at DESC);
					CREATE INDEX idx_streak_history_user
						ON streak_history(user_id, title_slug, type, written_at DESC);
				`); err != nil {
					return fmt.Errorf("create_streak_history: schema: %w", err)
				}

				if _, err := db.ExecContext(migration.BootCtx(), `
					CREATE OR REPLACE VIEW streak_latest AS
					SELECT DISTINCT ON (id)
						id, user_id, title_slug, type, started_at, current_length,
						best_length, last_increment_at, threshold, shields_used,
						shields_available, status, broken_at, written_at
					FROM streak_history
					ORDER BY id, written_at DESC, tech_id DESC;
				`); err != nil {
					return fmt.Errorf("create_streak_history: view: %w", err)
				}

				hasSource, err := migration.TableExists(db, "streak")
				if err != nil {
					return fmt.Errorf("create_streak_history: check source: %w", err)
				}
				if hasSource {
					if _, err := db.ExecContext(migration.BootCtx(), `
						INSERT INTO streak_history
							(id, user_id, title_slug, type, started_at, current_length,
							 best_length, last_increment_at, threshold, shields_used,
							 shields_available, status, broken_at, written_at)
						SELECT
							id, user_id, title_slug, type, started_at, current_length,
							best_length, last_increment_at, threshold, shields_used,
							shields_available, status, broken_at,
							COALESCE(last_increment_at, started_at, CURRENT_TIMESTAMP)
						FROM streak;
					`); err != nil {
						return fmt.Errorf("create_streak_history: backfill: %w", err)
					}
				}
				return nil
			},
		},
		// ─── engagement (player) — b17. ALTER player_match_enrichment (créée par le
		// schéma de base global) + table engagement_coefficients (create+repair PK,
		// paire atomique self-contained). Le step shared add_match_intensity_to_match_
		// registry est dans la section shared de Steps() (steps.go).
		{
			Name:        "add_engagement_score_columns_to_player_match_enrichment",
			TargetDB:    migration.TargetPlayer,
			Description: "Ajoute engagement_score, engagement_score_brut, engagement_score_confidence et mode_category a player_match_enrichment (idempotent)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "player_match_enrichment")
				if err != nil {
					return fmt.Errorf("engagement migration: check table: %w", err)
				}
				if !exists {
					return nil
				}
				for _, col := range []struct{ name, typ string }{
					{"engagement_score", "DOUBLE"},
					{"engagement_score_brut", "DOUBLE"},
					{"engagement_score_confidence", "VARCHAR"},
					{"mode_category", "VARCHAR"},
				} {
					if err := migration.AddColumnIfMissing(db, "player_match_enrichment", col.name, col.typ); err != nil {
						return err
					}
				}
				// Append-only #23046 (2026-06-21) : PLUS d'index ART idx_pme_engagement_history
				// sur (mode_category, engagement_score_brut) — colonnes mutées par l'étage
				// engagement (INSERT pur taggé). Lecture via player_match_enrichment_latest.
				// player_append_only_match_enrichment_v1 le supprime sur les DB existantes.
				return nil
			},
		},
		{
			Name:        "create_engagement_coefficients_table",
			TargetDB:    migration.TargetPlayer,
			Description: "Cree la table engagement_coefficients pour stocker le coef lobby global par (xuid, mode_category)",
			ApplySchema: func(db *sql.DB) error {
				// coef_team_share : colonne conservee NOT NULL mais INERTE depuis le
				// modele lobby-anchored v2 (D5, PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07) :
				// plus calculee ni lue (on y ecrit 1.0). Pas de DROP COLUMN sur les
				// player DB existantes. Seul coef_lobby_share alimente l'attendu (fallback
				// global) ; les bins d'intensite vivent dans engagement_response_bins.
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS engagement_coefficients (
						xuid             VARCHAR NOT NULL,
						mode_category    VARCHAR NOT NULL,
						coef_team_share  DOUBLE NOT NULL,
						coef_lobby_share DOUBLE NOT NULL,
						n_matches        INTEGER NOT NULL,
						last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (xuid, mode_category)
					);
					-- PAS d'index idx_engagement_coefficients_xuid : la PK (xuid, mode_category)
					-- couvre déjà le préfixe xuid ; un index redondant sur xuid est une surface
					-- ART superflue (#23046). Drop DB existantes : drop_engagement_coefficients_xuid_art_index_v1.
				`)
			},
		},
		{
			Name:        "repair_engagement_coefficients_primary_key",
			TargetDB:    migration.TargetPlayer,
			Description: "Reconstruit engagement_coefficients avec PRIMARY KEY (xuid, mode_category) quand elle manque (CREATE TABLE IF NOT EXISTS historique)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "engagement_coefficients")
				if err != nil {
					return fmt.Errorf("repair eng coefs PK: check table: %w", err)
				}
				if !exists {
					return nil
				}
				hasPK, err := migration.HasPrimaryKey(db, "engagement_coefficients")
				if err != nil {
					return fmt.Errorf("repair eng coefs PK: check PK: %w", err)
				}
				if hasPK {
					return nil
				}
				return migration.ExecScript(db, `
					CREATE TABLE engagement_coefficients__pkfix (
						xuid             VARCHAR NOT NULL,
						mode_category    VARCHAR NOT NULL,
						coef_team_share  DOUBLE NOT NULL,
						coef_lobby_share DOUBLE NOT NULL,
						n_matches        INTEGER NOT NULL,
						last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (xuid, mode_category)
					);
					INSERT INTO engagement_coefficients__pkfix
						SELECT xuid, mode_category, coef_team_share, coef_lobby_share, n_matches, last_updated
						FROM (
							SELECT *, ROW_NUMBER() OVER (
								PARTITION BY xuid, mode_category ORDER BY last_updated DESC
							) AS rn
							FROM engagement_coefficients
						) WHERE rn = 1;
					DROP TABLE engagement_coefficients;
					ALTER TABLE engagement_coefficients__pkfix RENAME TO engagement_coefficients;
					-- PAS d'index idx_engagement_coefficients_xuid (PK couvre xuid ; surface ART
					-- redondante #23046). Drop DB existantes : drop_engagement_coefficients_xuid_art_index_v1.
				`)
			},
		},
		{
			Name:        "add_engagement_pace_columns_to_player_match_enrichment",
			TargetDB:    migration.TargetPlayer,
			Description: "Ajoute engagement_pace_player, engagement_pace_team, engagement_pace_lobby et engagement_player_activity a player_match_enrichment (Phase recompute coefs)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "player_match_enrichment")
				if err != nil {
					return fmt.Errorf("engagement paces migration: check table: %w", err)
				}
				if !exists {
					return nil
				}
				for _, col := range []struct{ name, typ string }{
					{"engagement_pace_player", "DOUBLE"},
					{"engagement_pace_team", "DOUBLE"},
					{"engagement_pace_lobby", "DOUBLE"},
					{"engagement_player_activity", "INTEGER"},
				} {
					if err := migration.AddColumnIfMissing(db, "player_match_enrichment", col.name, col.typ); err != nil {
						return err
					}
				}
				// Append-only #23046 (2026-06-21) : PLUS d'index ART idx_pme_engagement_paces
				// sur (mode_category) — colonne mutée par l'étage engagement. LoadRatioSamples
				// lit player_match_enrichment_latest. player_append_only_match_enrichment_v1
				// le supprime sur les DB existantes.
				return nil
			},
		},
		{
			Name:     "create_engagement_response_bins_table",
			TargetDB: migration.TargetPlayer,
			Description: "Cree engagement_response_bins (modele lobby-anchored v2) : coef de reponse par" +
				" bin d'intensite (tercile de pace_lobby), par (xuid, mode_category, intensity_bin).",
			ApplySchema: func(db *sql.DB) error {
				// Table NEUVE : CREATE TABLE IF NOT EXISTS avec PK inline est sûr (elle
				// ne pré-existe jamais sans PK, donc pas de piège CREATE IF NOT EXISTS + PK).
				// Ecritures basse fréquence via SELECT-then-UPDATE-or-INSERT sous lease
				// (cf. engagement_coefficients) — pas d'ON CONFLICT, pas d'index secondaire
				// muté : aucune surface ART supplémentaire (#23046).
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS engagement_response_bins (
						xuid          VARCHAR NOT NULL,
						mode_category VARCHAR NOT NULL,
						intensity_bin VARCHAR NOT NULL,
						lower_bound   DOUBLE NOT NULL,
						upper_bound   DOUBLE NOT NULL,
						coef_lobby    DOUBLE NOT NULL,
						n_matches     INTEGER NOT NULL,
						last_updated  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (xuid, mode_category, intensity_bin)
					);
				`)
			},
		},
	}
}

const careerXPTotalFixMetaKey = "career_xp_total_default_fixed_v1"

// applyFixCareerXPTotalDefault supprime le DEFAULT 0 sur xp_total et corrige les rows à 0.
func applyFixCareerXPTotalDefault(db *sql.DB) error {
	hasMeta, err := migration.TableExists(db, "sync_meta")
	if err != nil {
		return fmt.Errorf("fix_xp_default: check sync_meta: %w", err)
	}
	if hasMeta {
		var marker sql.NullString
		if scanErr := db.QueryRowContext(migration.BootCtx(),
			`SELECT value FROM sync_meta WHERE key = ?`,
			careerXPTotalFixMetaKey,
		).Scan(&marker); scanErr == nil && marker.Valid {
			return nil
		}
	}

	hasTable, err := migration.TableExists(db, "career_progression")
	if err != nil {
		return fmt.Errorf("fix_xp_default: check table: %w", err)
	}
	if !hasTable {
		return markXPTotalFixDone(db)
	}

	var corrupt int
	if err := db.QueryRowContext(migration.BootCtx(),
		`SELECT COUNT(*) FROM career_progression WHERE xp_total = 0`,
	).Scan(&corrupt); err != nil {
		return fmt.Errorf("fix_xp_default: count corrupt: %w", err)
	}

	res, err := db.ExecContext(migration.BootCtx(),
		`UPDATE career_progression SET xp_total = NULL WHERE xp_total = 0`)
	if err != nil {
		return fmt.Errorf("fix_xp_default: update null: %w", err)
	}
	fixed, _ := res.RowsAffected()

	if _, err := db.ExecContext(migration.BootCtx(),
		`ALTER TABLE career_progression ALTER COLUMN xp_total DROP DEFAULT`); err != nil {
		return fmt.Errorf("fix_xp_default: drop default xp_total: %w", err)
	}

	slog.Info("migration fix_career_xp_total_default: done",
		"corrupt_rows_found", corrupt,
		"rows_set_to_null", fixed,
	)
	return markXPTotalFixDone(db)
}

func markXPTotalFixDone(db *sql.DB) error {
	hasMeta, err := migration.TableExists(db, "sync_meta")
	if err != nil || !hasMeta {
		return nil
	}
	if err := migration.AddColumnIfMissing(db, "sync_meta", "updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return fmt.Errorf("fix_xp_default: ensure updated_at: %w", err)
	}
	var exists bool
	_ = db.QueryRowContext(migration.BootCtx(), `SELECT EXISTS(SELECT 1 FROM sync_meta WHERE key = ?)`, careerXPTotalFixMetaKey).Scan(&exists)
	if exists {
		_, err = db.ExecContext(migration.BootCtx(), `UPDATE sync_meta SET value = 'true', updated_at = NOW() WHERE key = ?`, careerXPTotalFixMetaKey)
	} else {
		_, err = db.ExecContext(migration.BootCtx(), `INSERT INTO sync_meta (key, value, updated_at) VALUES (?, 'true', NOW())`, careerXPTotalFixMetaKey)
	}
	if err != nil {
		return fmt.Errorf("fix_xp_default: mark done: %w", err)
	}
	return nil
}
