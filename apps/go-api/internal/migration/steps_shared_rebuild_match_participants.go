package migration

// steps_shared_rebuild_match_participants.go — rebuild match_participants
// via swap pour défaire la corruption d'index ART qui faussait les requêtes
// `WHERE match_id = ?` et `WHERE match_id IN (...)`.
//
// Contexte : bug DuckDB documenté dans
// `docs/INCIDENT_2026-05-20_match_participants_index.md`. Les requêtes avec
// filter pushdown sur la PK `(match_id, xuid)` retournaient un sous-ensemble
// strict des rows (ex: 1 row au lieu de 10 pour le match
// `50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e`). Le workaround `WHERE col || '' = ?`
// force un table-scan et retourne les rows correctes — preuve que les rows
// existent mais que l'arbre ART ne pointe pas dessus.
//
// Conséquence métier : `loadLUSRParticipants` chargeait 1-2 participants au
// lieu de 8-16 pour les matchs concernés. Pour Madina97294 (1079 match_ids
// LUSR-éligibles), presque tous ses matchs étaient impactés. Résultat : LUSR
// figé en Argent IV alors qu'attendu fin Platine / début Diamant.
//
// Cf. également `steps_player_rebuild_career_progression.go` — même classe de
// bug sur `career_progression` (player DB), même pattern de fix appliqué le
// 2026-05-21 (commits 2e0f0247 + 651b9de6).
//
// Stratégie : rebuild `match_participants` via swap CTAS (Create Table As
// Select) — le SELECT * sans WHERE force un table-scan complet qui lit les
// pages physiques sans consulter l'index ART corrompu. La nouvelle table est
// créée avec un index ART vierge.
//
//   1. PRAGMA table_info pour énumérer les colonnes actuelles (robuste aux
//      additions futures de colonnes via ALTER TABLE)
//   2. CREATE TABLE __rebuilt AS SELECT <cols> FROM match_participants
//   3. DROP TABLE match_participants (cascade sur vues)
//   4. RENAME __rebuilt → match_participants
//   5. ADD PRIMARY KEY (match_id, xuid)
//   6. Recrée les vues (applyResolutionViews + applyMvPlayerMatchesView)
//   7. Recrée les indexes (5 indexes idx_mp_*)
//
// Idempotente : sentinel `match_participants_rebuilt_v1` dans `sync_meta`.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

const matchParticipantsRebuildMetaKey = "match_participants_rebuilt_v1"

func init() {
	Register(Migration{
		Name:        "rebuild_match_participants_defeat_art_corruption",
		TargetDB:    TargetShared,
		Description: "Rebuild match_participants via swap CTAS pour défaire la corruption ART (filter pushdown sur PK match_id ramenait sous-ensemble strict — incident 2026-05-20).",
		ApplySchema: applyRebuildMatchParticipants,
	})
}

func applyRebuildMatchParticipants(db *sql.DB) error {
	ctx := bootCtx()

	// Idempotence : si sentinel présent dans sync_meta, no-op.
	hasMeta, err := tableExists(db, "sync_meta")
	if err != nil {
		return fmt.Errorf("rebuild_mp: check sync_meta: %w", err)
	}
	if hasMeta {
		var marker sql.NullString
		if err := db.QueryRowContext(ctx,
			`SELECT value FROM sync_meta WHERE key = ?`,
			matchParticipantsRebuildMetaKey).Scan(&marker); err == nil && marker.Valid {
			return nil
		}
	}

	hasTable, err := tableExists(db, "match_participants")
	if err != nil {
		return fmt.Errorf("rebuild_mp: check table: %w", err)
	}
	if !hasTable {
		// Pas de table → créée par la migration initiale plus tard, on marque
		// le sentinel pour éviter de re-check à chaque boot.
		return markMatchParticipantsRebuildDone(db)
	}

	cols, err := loadMatchParticipantsColumns(ctx, db)
	if err != nil {
		return fmt.Errorf("rebuild_mp: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		// Table existe mais sans colonnes — état impossible mais on sécurise.
		return markMatchParticipantsRebuildDone(db)
	}
	colList := strings.Join(cols, ", ")

	// Diag : compter rows pour log informatif (avant rebuild).
	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&before); err != nil {
		return fmt.Errorf("rebuild_mp: count before: %w", err)
	}

	// Swap CTAS. Le SELECT * (sans WHERE) force un table-scan complet qui
	// court-circuite l'index ART corrompu et lit les pages physiques. La
	// nouvelle table reçoit un ART vierge construit sur des données complètes.
	stmts := []string{
		`DROP TABLE IF EXISTS match_participants__rebuilt`,
		fmt.Sprintf(`CREATE TABLE match_participants__rebuilt AS SELECT %s FROM match_participants`, colList),
		// DROP TABLE supprime aussi les vues dépendantes (cascade interne DuckDB).
		`DROP TABLE match_participants`,
		`ALTER TABLE match_participants__rebuilt RENAME TO match_participants`,
		`ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid)`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("rebuild_mp: swap step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	// Recrée vues + indexes (cf. dropAssistsExpectedShared lignes 718-731).
	if err := applyResolutionViews(db); err != nil {
		return fmt.Errorf("rebuild_mp: recreate resolution views: %w", err)
	}
	if err := applyMvPlayerMatchesView(db); err != nil {
		return fmt.Errorf("rebuild_mp: recreate mv_player_matches: %w", err)
	}
	for _, ddl := range matchParticipantsIndexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("rebuild_mp: recreate index: %w", err)
		}
	}

	// Validation post-rebuild : compter à nouveau pour log + sanity check.
	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&after); err != nil {
		return fmt.Errorf("rebuild_mp: count after: %w", err)
	}

	slog.Info("migration rebuild_match_participants: table rebuilt (ART corruption defeated)",
		"rows_before_rebuild", before,
		"rows_after_rebuild", after,
		"columns_preserved", len(cols),
	)
	return markMatchParticipantsRebuildDone(db)
}

// loadMatchParticipantsColumns énumère les colonnes via PRAGMA table_info.
// Ordre préservé via ordinal_position.
func loadMatchParticipantsColumns(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info('match_participants')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid int
		var name, typ string
		var nn bool
		var dflt *string
		var pk bool
		if err := rows.Scan(&cid, &name, &typ, &nn, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}

// matchParticipantsIndexes : indexes à recréer post-rebuild.
// Liste alignée sur dropAssistsExpectedShared (steps_shared.go:724-731).
var matchParticipantsIndexes = []string{
	"CREATE INDEX IF NOT EXISTS idx_mp_backfill   ON match_participants(xuid, backfill_bits)",
	"CREATE INDEX IF NOT EXISTS idx_mp_xuid_match ON match_participants(xuid, match_id)",
	"CREATE INDEX IF NOT EXISTS idx_mp_match_xuid ON match_participants(match_id, xuid)",
	"CREATE INDEX IF NOT EXISTS idx_mp_xuid_team  ON match_participants(xuid, team_id, match_id)",
	"CREATE INDEX IF NOT EXISTS idx_mp_xuid       ON match_participants(xuid)",
	"CREATE INDEX IF NOT EXISTS idx_mp_match_id   ON match_participants(match_id)",
}

// markMatchParticipantsRebuildDone écrit le sentinel d'idempotence.
// Robuste aux schémas legacy sans updated_at (ajout colonne si absente).
func markMatchParticipantsRebuildDone(db *sql.DB) error {
	hasMeta, err := tableExists(db, "sync_meta")
	if err != nil || !hasMeta {
		return nil
	}
	if err := addColumnIfMissing(db, "sync_meta", "updated_at",
		"TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return fmt.Errorf("rebuild_mp: ensure updated_at: %w", err)
	}
	_, err = db.ExecContext(bootCtx(), `
		INSERT INTO sync_meta (key, value, updated_at)
		VALUES (?, 'true', NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = 'true',
			updated_at = NOW()
	`, matchParticipantsRebuildMetaKey)
	if err != nil {
		return fmt.Errorf("rebuild_mp: mark done: %w", err)
	}
	return nil
}
