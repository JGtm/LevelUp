// Package migration — monitoring_schema.go : schéma de la base monitoring
// globale (data/global/monitoring.duckdb, DC-1 du plan monitoring 2026-07).
//
// Cette base est GLOBALE (pas per-titre) et n'entre donc pas dans le registre
// title-scopé (registry.go / order.go). Son schéma est posé idempotemment par
// EnsureMonitoringSchema, appelé à l'ouverture du store (internal/ops) — même
// discipline que les migrations : CREATE ... IF NOT EXISTS + CREATE OR REPLACE
// VIEW, réentrant sans état de tracking.
//
// Toutes les tables sont APPEND-ONLY (ADR 0026) : PK technique `id` alimentée
// par une séquence, colonne d'horloge `written_at`, écriture = INSERT pur, aucun
// UPDATE/DELETE. La lecture passe TOUJOURS par les vues `_latest` (une lecture
// brute des status/cron servirait des lignes périmées). Un seul writer = le
// process serveur (aucune CLI concurrente) — pas de pression ART.
package migration

import (
	"context"
	"database/sql"
	"fmt"
)

// monitoringDDL regroupe les instructions idempotentes de création du schéma
// monitoring, dans l'ordre (séquences → tables → vues). Chaque instruction est
// réentrante (IF NOT EXISTS / OR REPLACE).
var monitoringDDL = []string{
	// ─── Séquences backing les PK techniques ────────────────────────────────
	`CREATE SEQUENCE IF NOT EXISTS monitoring_detection_events_seq START 1`,
	`CREATE SEQUENCE IF NOT EXISTS monitoring_detection_status_seq START 1`,
	`CREATE SEQUENCE IF NOT EXISTS monitoring_cron_runs_seq START 1`,
	`CREATE SEQUENCE IF NOT EXISTS monitoring_data_health_seq START 1`,

	// ─── detection_events : une ligne par flush avec des occurrences nouvelles
	// pour un fingerprint. count_delta = occurrences depuis le dernier flush.
	`CREATE TABLE IF NOT EXISTS detection_events (
		id            BIGINT PRIMARY KEY DEFAULT nextval('monitoring_detection_events_seq'),
		fingerprint   VARCHAR NOT NULL,
		level         VARCHAR,
		module        VARCHAR,
		message       VARCHAR,
		title_slug    VARCHAR,
		occurred_at   TIMESTAMP,
		count_delta   BIGINT,
		sample_detail VARCHAR,
		written_at    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,

	// ─── detection_status_events : cycle de vie (open/acked/muted/resolved).
	// Une occurrence après resolved ré-ouvre = append d'un event 'open' par le
	// writer (DC-2). La vue _latest reflète le dernier statut.
	`CREATE TABLE IF NOT EXISTS detection_status_events (
		id          BIGINT PRIMARY KEY DEFAULT nextval('monitoring_detection_status_seq'),
		fingerprint VARCHAR NOT NULL,
		status      VARCHAR NOT NULL,
		note        VARCHAR,
		written_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,

	// ─── cron_runs : historique par exécution de cron (persiste au restart).
	`CREATE TABLE IF NOT EXISTS cron_runs (
		id          BIGINT PRIMARY KEY DEFAULT nextval('monitoring_cron_runs_seq'),
		cron_name   VARCHAR NOT NULL,
		started_at  TIMESTAMP,
		ok          BOOLEAN,
		err         VARCHAR,
		duration_ms BIGINT,
		written_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,

	// ─── data_health_runs : résultat sérialisé (JSON) du dernier audit data
	// health, pour survivre au restart (l'overview lisait un état mémoire).
	`CREATE TABLE IF NOT EXISTS data_health_runs (
		id          BIGINT PRIMARY KEY DEFAULT nextval('monitoring_data_health_seq'),
		result_json VARCHAR,
		written_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,

	// ─── Vue : dernier statut par fingerprint (ART règle n°2 : lecture via _latest).
	`CREATE OR REPLACE VIEW detection_status_latest AS
		SELECT fingerprint, status, note, written_at
		FROM (
			SELECT *, ROW_NUMBER() OVER (
				PARTITION BY fingerprint ORDER BY written_at DESC, id DESC
			) AS rn
			FROM detection_status_events
		)
		WHERE rn = 1`,

	// ─── Vue : détection agrégée par fingerprint (count cumulé, first/last_seen,
	// dernier level/module/message/title/sample) + statut effectif (défaut 'open').
	`CREATE OR REPLACE VIEW detections_latest AS
		SELECT
			a.fingerprint,
			a.total_count               AS count,
			a.first_seen,
			a.last_seen,
			a.level,
			a.module,
			a.message,
			a.title_slug,
			a.sample_detail,
			COALESCE(s.status, 'open')  AS status,
			s.note                      AS note,
			s.written_at                AS status_at
		FROM (
			SELECT
				fingerprint,
				SUM(count_delta)              AS total_count,
				MIN(occurred_at)              AS first_seen,
				MAX(occurred_at)              AS last_seen,
				ARG_MAX(level, occurred_at)         AS level,
				ARG_MAX(module, occurred_at)        AS module,
				ARG_MAX(message, occurred_at)       AS message,
				ARG_MAX(title_slug, occurred_at)    AS title_slug,
				ARG_MAX(sample_detail, occurred_at) AS sample_detail
			FROM detection_events
			GROUP BY fingerprint
		) a
		LEFT JOIN detection_status_latest s USING (fingerprint)`,

	// ─── Vue : dernier run par cron (santé courante, A6).
	`CREATE OR REPLACE VIEW cron_runs_latest AS
		SELECT cron_name, started_at, ok, err, duration_ms, written_at
		FROM (
			SELECT *, ROW_NUMBER() OVER (
				PARTITION BY cron_name ORDER BY started_at DESC, id DESC
			) AS rn
			FROM cron_runs
		)
		WHERE rn = 1`,
}

// EnsureMonitoringSchema pose (idempotemment) le schéma de la base monitoring
// globale. Réentrant : sûr à chaque ouverture du store.
func EnsureMonitoringSchema(ctx context.Context, db *sql.DB) error {
	for _, stmt := range monitoringDDL {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration monitoring: %w", err)
		}
	}
	return nil
}
