package migration

// steps_player_append_only_streak.go — migration vers le pattern append-only
// pour les streaks (fix 2026-05-30, Phase B du fix progression ART).
//
// Contexte : la table `streak` (PK VARCHAR id, player DB stats.duckdb) était
// écrite par StreaksRepo.Upsert en SELECT-then-UPDATE-or-INSERT. L'UPDATE in
// place reste une mutation qui, sur une DB dont l'index ART a été corrompu par
// un crash antérieur, peut ressusciter le bug "Failed to delete all rows from
// index" / "Binder Error: conflict target...". On bascule en INSERT pur.
//
// Solution (modèle player_records_history) :
//   - Nouvelle table `streak_history` : PK technique tech_id BIGINT (séquence)
//     + colonne métier `id` VARCHAR + toutes les colonnes streak + written_at.
//   - INSERT pur, jamais UPDATE → zéro pression ART.
//   - Vue `streak_latest` : dernière version par `id` métier (DISTINCT ON id,
//     ORDER BY written_at DESC, tech_id DESC). Préserve la sémantique "une
//     streak logique = un id ; sa version courante = la plus récente".
//
// Backfill : INSERT INTO streak_history SELECT * FROM streak (one-shot,
// conditionnel sur l'existence de `streak`).
//
// Idempotente : si `streak_history` existe déjà, no-op. Défensive vs ordre
// d'init Go (ce fichier "append_only_streak" précède "progression"
// alphabétiquement → `streak` peut ne pas exister encore : on crée la table +
// la vue dans tous les cas, le backfill est conditionnel).

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "create_streak_history_append_only",
		TargetDB:    TargetPlayer,
		Description: "Pattern append-only pour streak : nouvelle table streak_history (INSERT pur) + vue streak_latest. Backfill depuis l'ancienne table streak.",
		ApplySchema: func(db *sql.DB) error {
			has, err := tableExists(db, "streak_history")
			if err != nil {
				return fmt.Errorf("create_streak_history: check table: %w", err)
			}
			if has {
				return nil
			}

			// 1. CREATE SEQUENCE + TABLE. tech_id = PK technique (≠ id métier
			//    VARCHAR existant). written_at départage les versions.
			if _, err := db.ExecContext(bootCtx(), `
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

			// 2. CREATE VIEW streak_latest : dernière version par id métier.
			if _, err := db.ExecContext(bootCtx(), `
				CREATE OR REPLACE VIEW streak_latest AS
				SELECT DISTINCT ON (id)
					id,
					user_id,
					title_slug,
					type,
					started_at,
					current_length,
					best_length,
					last_increment_at,
					threshold,
					shields_used,
					shields_available,
					status,
					broken_at,
					written_at
				FROM streak_history
				ORDER BY id, written_at DESC, tech_id DESC;
			`); err != nil {
				return fmt.Errorf("create_streak_history: view: %w", err)
			}

			// 3. Backfill one-shot depuis l'ancienne table streak (si présente).
			hasSource, err := tableExists(db, "streak")
			if err != nil {
				return fmt.Errorf("create_streak_history: check source: %w", err)
			}
			if hasSource {
				if _, err := db.ExecContext(bootCtx(), `
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
	})
}
