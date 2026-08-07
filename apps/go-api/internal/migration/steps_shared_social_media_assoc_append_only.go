package migration

// steps_shared_social_media_assoc_append_only.go — bascule media_match_associations
// en APPEND-ONLY (soft-delete is_active).
//
// AVANT : SetMediaMatchAssociation (replace manuel) = DELETE WHERE media_file_id +
// INSERT manual ; ReassociateMedia = DELETE WHERE NOT is_manual + re-INSERT. Deux
// surfaces ART (PK media_file_id,match_id + idx_mma_match) sur shared_social (handle
// RW partagé concurrent).
//
// APRÈS : table sœur media_match_associations_history (PK technique id BIGINT) avec
//   - is_manual : priorité dans la vue (une correction manuelle masque l'auto).
//   - is_active : soft-delete (purge CLI sans DELETE).
//   - associated_at IMMUABLE : horodatage de l'association (récence notif préservée),
//     distinct de written_at (horodatage de l'event).
// L'association EFFECTIVE d'un média = dernier event actif (priorité manuel) via la
// vue media_match_associations_latest. Le replace manuel = 1 INSERT (la vue masque
// l'ancienne). Le reindex auto = INSERT additif (loadUnassociatedMedia forward-only
// → plus de DELETE+re-INSERT). Zéro DELETE/UPDATE-indexé.
//
// media_file_id = INTEGER en schéma LIVE (ensureMediaTables) → stocké BIGINT ici
// (compatible int64 Go). La table legacy media_match_associations n'est PAS droppée
// (vestige vide, plus jamais écrite par le hot path). Pattern gabarit favorites/likes.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_media_assoc_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "media_match_associations append-only : table _history (is_active/is_manual/associated_at) + vue _latest + backfill — élimine DELETE (surface ART)",
		ApplySchema: applyMediaAssocAppendOnly,
	})
}

// mediaAssocHistoryDDL est partagé entre la migration et ensureMediaTables (ops/media_store.go)
// pour garantir un substrat convergent même sur DB fraîche (ensureMediaTables gagne parfois
// la course aux migrations au 1er IndexMedia).
const mediaAssocHistoryDDL = `
	CREATE SEQUENCE IF NOT EXISTS media_match_associations_history_id_seq START 1;
	CREATE TABLE IF NOT EXISTS media_match_associations_history (
		id            BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
		media_file_id BIGINT NOT NULL,
		match_id      VARCHAR NOT NULL,
		delta_seconds INTEGER,
		is_manual     BOOLEAN NOT NULL DEFAULT FALSE,
		is_active     BOOLEAN NOT NULL DEFAULT TRUE,
		associated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		written_at    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE INDEX IF NOT EXISTS idx_mmah_lookup ON media_match_associations_history(media_file_id, written_at DESC);
	CREATE INDEX IF NOT EXISTS idx_mmah_match  ON media_match_associations_history(match_id);
	CREATE OR REPLACE VIEW media_match_associations_latest AS
		WITH lpp AS (
			SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
				ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
			FROM media_match_associations_history
		),
		act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
		hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
		SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
		FROM act a JOIN hm ON hm.media_file_id = a.media_file_id
		WHERE a.is_manual = hm.has_manual;
`

func applyMediaAssocAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "media_match_associations_history")
	if err != nil {
		return fmt.Errorf("media_assoc append-only: check history: %w", err)
	}
	if hasHistory {
		return nil
	}

	if _, err := db.ExecContext(ctx, mediaAssocHistoryDDL); err != nil {
		return fmt.Errorf("media_assoc append-only: schema: %w", err)
	}

	// Backfill idempotent depuis la table legacy (live INTEGER) : chaque ligne = event
	// actif ; associated_at = created_at legacy si présent (préserve la récence).
	hasLegacy, _ := tableExists(db, "media_match_associations")
	if hasLegacy {
		hasManual, _ := columnExists(db, "media_match_associations", "is_manual")
		hasCreated, _ := columnExists(db, "media_match_associations", "created_at")
		manualExpr := "FALSE"
		if hasManual {
			manualExpr = "COALESCE(is_manual, FALSE)"
		}
		// atExpr alimente written_at : horloge UTC canonique OBLIGATOIRE (lot S5).
		// `CURRENT_TIMESTAMP` y rendait un TIMESTAMPTZ coercé par le fuseau de
		// session, donc un written_at daté dans le futur qui gagne l'arbitrage de
		// media_match_associations_latest contre les events UTC écrits ensuite.
		atExpr := WrittenAtDefaultUTC
		if hasCreated {
			atExpr = "COALESCE(created_at, " + WrittenAtDefaultUTC + ")"
		}
		q := fmt.Sprintf(`
			INSERT INTO media_match_associations_history
				(media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at)
			SELECT CAST(media_file_id AS BIGINT), match_id, delta_seconds, %s, TRUE, %s, %s
			FROM media_match_associations`, manualExpr, atExpr, atExpr)
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("media_assoc append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_match_associations_history`).Scan(&n)
	slog.InfoContext(ctx, "media_match_associations append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
