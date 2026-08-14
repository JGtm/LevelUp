// Package migration — build_queue_schema.go : schéma de la FILE DURABLE de
// construction (piste F §2 du plan .ai/PLAN_MASTER_FILM_KILLFEED_REJEU.md).
//
// POURQUOI CETTE TABLE EXISTE. Le JobStore (internal/platform/jobs, jobs.json)
// n'a aucune sémantique de file : pas de prise (claim), pas de priorité, pas de
// reprise — au boot, tout job `running` devient `interrupted` et la rétention le
// jette après 1 h. C'est acceptable pour l'auto-sync (qui se relance tout seul),
// jamais pour un travail délégué à une machine tierce : un ouvrier qui meurt en
// plein décodage doit rendre son job à la file, pas l'emporter avec lui.
//
// OÙ ELLE VIT, ET POURQUOI ICI. Dans la base monitoring GLOBALE
// (data/global/monitoring.duckdb) : même fichier, même writer unique (le process
// serveur), même lease KindMonitoring, mêmes invariants. Zéro base nouvelle,
// zéro lease nouveau, zéro modèle d'écriture nouveau — « le chemin d'écriture
// normal » du plan.
//
// APPEND-ONLY (ADR 0026). Une prise, un battement, un résultat sont des
// ÉVÉNEMENTS : on n'UPDATE jamais une ligne de job, on append son nouvel état.
// L'état courant se lit EXCLUSIVEMENT par les vues `build_jobs_latest` /
// `build_workers_latest` — une lecture brute de la table servirait un état
// périmé (piège documenté ADR 0026).
//
// L'ÉTAT VIT CÔTÉ WEB, JAMAIS CÔTÉ OUVRIER (piste F §4bis) : l'ouvrier calcule et
// ne détient rien. C'est ce qui rend le travail distant observable depuis le
// dashboard admin sans jamais interroger l'ouvrier.
package migration

// buildQueueDDL : instructions idempotentes de la file de construction, posées
// par EnsureMonitoringSchema à la suite de monitoringDDL (même base, même
// discipline réentrante).
var buildQueueDDL = []string{
	// ─── Séquences backing les PK techniques ────────────────────────────────
	`CREATE SEQUENCE IF NOT EXISTS build_job_events_seq START 1`,
	`CREATE SEQUENCE IF NOT EXISTS build_worker_events_seq START 1`,

	// ─── build_job_events : un événement par transition d'état d'un job.
	//
	// job_id est la CLÉ LOGIQUE (partition des vues `_latest`), pas la PK : le
	// même job_id porte N lignes (queued → running → succeeded).
	//
	// payload_json porte le travail RÉSOLU PAR LE WEB — les URL CDN pré-signées
	// des morceaux du film. C'est la pièce de sécurité du découpage (piste F §1) :
	// le manifeste exige les tokens Halo, les morceaux non. Le web résout, met les
	// URL dans le job, et l'ouvrier n'a JAMAIS le moindre secret Halo ni le
	// moindre accès à la base.
	//
	// lease_expires_at est ce qui rend la file résistante à la mort d'un ouvrier :
	// un job pris dont le bail a expiré retourne en file (attempt+1), que
	// l'ouvrier soit mort, débranché, ou que ce soit le serveur qui ait redémarré.
	`CREATE TABLE IF NOT EXISTS build_job_events (
		id               BIGINT PRIMARY KEY DEFAULT nextval('build_job_events_seq'),
		job_id           VARCHAR NOT NULL,
		job_type         VARCHAR NOT NULL,
		title_slug       VARCHAR,
		match_id         VARCHAR,
		status           VARCHAR NOT NULL,
		priority         INTEGER DEFAULT 0,
		attempt          INTEGER DEFAULT 0,
		worker_id        VARCHAR,
		lease_expires_at TIMESTAMP,
		payload_json     VARCHAR,
		result_json      VARCHAR,
		error_code       VARCHAR,
		error_message    VARCHAR,
		enqueued_at      TIMESTAMP,
		written_at       TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,

	// ─── build_worker_events : un battement de cœur d'ouvrier (prise de job,
	// battement périodique, résultat rendu). Un ouvrier muet depuis N minutes
	// s'affiche hors ligne, et ses jobs retournent en file par expiration du bail.
	`CREATE TABLE IF NOT EXISTS build_worker_events (
		id             BIGINT PRIMARY KEY DEFAULT nextval('build_worker_events_seq'),
		worker_id      VARCHAR NOT NULL,
		hostname       VARCHAR,
		version        VARCHAR,
		current_job_id VARCHAR,
		jobs_done      BIGINT DEFAULT 0,
		jobs_failed    BIGINT DEFAULT 0,
		note           VARCHAR,
		beat_at        TIMESTAMP,
		written_at     TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,

	// ─── Vue : état courant de chaque job (ART règle n°2 — LA lecture).
	`CREATE OR REPLACE VIEW build_jobs_latest AS
		SELECT job_id, job_type, title_slug, match_id, status, priority, attempt,
			worker_id, lease_expires_at, payload_json, result_json,
			error_code, error_message, enqueued_at, written_at
		FROM (
			SELECT *, ROW_NUMBER() OVER (
				PARTITION BY job_id ORDER BY written_at DESC, id DESC
			) AS rn
			FROM build_job_events
		)
		WHERE rn = 1`,

	// ─── Vue : dernier battement de chaque ouvrier (+ son cumul de travail, qui
	// vient du dernier événement puisque l'ouvrier envoie des compteurs cumulés).
	`CREATE OR REPLACE VIEW build_workers_latest AS
		SELECT worker_id, hostname, version, current_job_id, jobs_done, jobs_failed,
			note, beat_at, written_at
		FROM (
			SELECT *, ROW_NUMBER() OVER (
				PARTITION BY worker_id ORDER BY beat_at DESC, id DESC
			) AS rn
			FROM build_worker_events
		)
		WHERE rn = 1`,
}
