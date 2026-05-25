package migration

// steps_player_spartan_identity.go — table dédiée customisation Spartan
// (PLAN_SPARTAN_IDENTITY_REFACTOR §11, 2026-05-25).
//
// Pourquoi une table dédiée : `career_progression` mélangeait historique
// rank/XP (append-only) et customisation latest-only (banner, emblem,
// backdrop, spartan_id). Le hack `ARG_MAX FILTER WHERE NOT NULL` était
// fragile et ne permettait pas de tracer pourquoi un joueur n'avait pas
// sa bannière (échec API silencieux ? cache jamais chaud ? tokens absents ?).
//
// Cette table sépare proprement : 1 row par xuid, UPSERT, avec un champ
// `last_attempt_status` qui rend le diagnostic immédiat.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "create_spartan_identity_table",
		TargetDB:    TargetPlayer,
		Description: "Table dédiée customisation Spartan (1 row/xuid, UPSERT)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS spartan_identity (
					xuid                VARCHAR PRIMARY KEY,
					spartan_id          VARCHAR,
					banner_image_url    VARCHAR,
					emblem_image_url    VARCHAR,
					backdrop_image_url  VARCHAR,
					last_refreshed_at   TIMESTAMP,
					last_attempt_at     TIMESTAMP,
					last_attempt_status VARCHAR
				);
			`)
		},
		// Phase 4 PLAN_SPARTAN_IDENTITY_REFACTOR §11 : backfill depuis
		// career_progression pour ne perdre aucune bannière/emblem historique.
		// Idempotent : skip si une row existe déjà pour ce xuid. Best-effort :
		// si career_progression est vide ou la table absente, no-op silencieux.
		ApplyBackfill: backfillSpartanIdentityFromCareerProgression,
	})
}

// backfillSpartanIdentityFromCareerProgression copie la dernière customisation
// connue (ARG_MAX par recorded_at, filtrant les valeurs vides) de
// `career_progression` vers `spartan_identity` pour chaque xuid présent.
//
// Sécurité : idempotent (skip si row existe déjà pour ce xuid). No-op gracieux
// si career_progression est absente (joueur jamais sync'd).
func backfillSpartanIdentityFromCareerProgression(db *sql.DB) error {
	// 1. Vérifier que career_progression existe (no-op sinon).
	var hasTable int
	err := db.QueryRow(`SELECT 1 FROM information_schema.tables WHERE table_name = 'career_progression' LIMIT 1`).Scan(&hasTable)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("backfill spartan_identity: check table: %w", err)
	}

	// 2. Lister les xuid présents dans career_progression (en pratique 1 par
	//    player DB mais on supporte le cas multi-xuid au cas où).
	rows, err := db.Query(`SELECT DISTINCT xuid FROM career_progression WHERE NULLIF(TRIM(xuid), '') IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("backfill spartan_identity: list xuids: %w", err)
	}
	defer rows.Close()
	var xuids []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return fmt.Errorf("backfill spartan_identity: scan xuid: %w", err)
		}
		xuids = append(xuids, x)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("backfill spartan_identity: iter xuids: %w", err)
	}

	// 3. Pour chaque xuid, INSERT si row absente.
	const insertSQL = `
INSERT INTO spartan_identity (
	xuid, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url,
	last_refreshed_at, last_attempt_at, last_attempt_status
)
SELECT
	?,
	ARG_MAX(spartan_id,          recorded_at) FILTER (WHERE NULLIF(TRIM(spartan_id),          '') IS NOT NULL),
	ARG_MAX(banner_image_url,    recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url),    '') IS NOT NULL),
	ARG_MAX(emblem_image_url,    recorded_at) FILTER (WHERE NULLIF(TRIM(emblem_image_url),    '') IS NOT NULL),
	ARG_MAX(backdrop_image_url,  recorded_at) FILTER (WHERE NULLIF(TRIM(backdrop_image_url),  '') IS NOT NULL),
	MAX(recorded_at),
	CURRENT_TIMESTAMP,
	'ok'
FROM career_progression
WHERE xuid || '' = ?`

	for _, xuid := range xuids {
		// Skip si row spartan_identity déjà présente (idempotent).
		var exists int
		err := db.QueryRow(`SELECT 1 FROM spartan_identity WHERE xuid = ?`, xuid).Scan(&exists)
		if err == nil {
			continue // row existe déjà → skip
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("backfill spartan_identity: check exists xuid=%s: %w", xuid, err)
		}
		if _, err := db.Exec(insertSQL, xuid, xuid); err != nil {
			return fmt.Errorf("backfill spartan_identity: insert xuid=%s: %w", xuid, err)
		}
	}
	return nil
}
