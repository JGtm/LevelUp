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
//   3. DROP TABLE match_participants — les vues dépendantes RESTENT au catalogue
//      (cf. la note de swapMatchParticipantsTx : DuckDB ne les supprime pas, même
//      avec CASCADE)
//   4. RENAME __rebuilt → match_participants
//   5. ADD PRIMARY KEY (match_id, xuid)
//   6. Recrée les vues par CREATE OR REPLACE (applyResolutionViews +
//      applyMvPlayerMatchesView) — c'est ce OR REPLACE qui écrase celles restées
//      au catalogue et les rebinde sur la nouvelle table
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

	// Délègue à la fonction runtime (sans sentinel) — séparation propre entre
	// la mécanique de rebuild SQL et l'idempotence migration via sync_meta.
	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		return err
	}
	return markMatchParticipantsRebuildDone(db)
}

// RebuildMatchParticipantsART exécute le rebuild swap CTAS de la table
// match_participants pour défaire la corruption d'index ART (cf. Phase 4.1
// du plan stabilisation 2026-05-22, docs/adr/0018-concurrent-write-model.md
// et docs/INCIDENT_2026-05-20_match_participants_index.md).
//
// IDEMPOTENTE PAR DESIGN — peut être rappelée à chaque boot ou périodiquement
// quand BootARTGuard détecte une nouvelle divergence ART. Ne pose AUCUN
// sentinel (contrairement à applyRebuildMatchParticipants côté migration).
//
// Étapes :
//  1. PRAGMA table_info pour énumérer les colonnes actuelles (robuste aux
//     ALTER TABLE futurs).
//  2. CREATE TABLE __rebuilt AS SELECT <cols> FROM match_participants
//     (le SELECT * sans WHERE force un table-scan complet qui court-circuite
//     l'index ART corrompu).
//  3. DROP TABLE match_participants — les vues dépendantes restent au catalogue.
//  4. RENAME __rebuilt → match_participants.
//  5. ADD PRIMARY KEY (match_id, xuid).
//  6. Recrée les vues EN CREATE OR REPLACE (applyResolutionViews +
//     applyMvPlayerMatchesView) : le OR REPLACE est ce qui écrase les vues
//     survivantes de l'étape 3 et les rebinde sur la table neuve.
//  7. Recrée les 6 indexes idx_mp_*.
//
// No-op gracieux si la table match_participants est absente.
//
// Verrou : pas de lock interne. Le caller (cmd/server au boot, ou un trigger
// runtime périodique) doit garantir qu'aucun writer concurrent n'opère
// pendant l'exécution. Au boot, le serveur n'a pas encore démarré son
// scheduler de sync donc c'est trivialement safe.
func RebuildMatchParticipantsART(ctx context.Context, db *sql.DB) error {
	// Récupération d'un __rebuilt orphelin : table principale absente +
	// match_participants__rebuilt présent = crash au milieu d'un swap NON
	// transactionnel (versions antérieures à la revue P0 2026-06-02). On promeut
	// le rebuilt avant toute autre logique — sinon la table partagée de tous les
	// joueurs resterait définitivement absente, sans réparation automatique.
	if err := recoverOrphanMatchParticipants(ctx, db); err != nil {
		return err
	}

	hasTable, err := tableExists(db, "match_participants")
	if err != nil {
		return fmt.Errorf("rebuild_mp_runtime: check table: %w", err)
	}
	if !hasTable {
		// Pas de table → no-op silencieux (vs applyRebuildMatchParticipants
		// qui pose un sentinel ; la version runtime ne pose rien).
		return nil
	}

	cols, err := loadMatchParticipantsColumns(ctx, db)
	if err != nil {
		return fmt.Errorf("rebuild_mp_runtime: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		// Table existe mais sans colonnes — état impossible mais on sécurise.
		return nil
	}
	colList := strings.Join(cols, ", ")

	// Diag : compter rows pour log informatif (avant rebuild).
	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&before); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: count before: %w", err)
	}

	// Swap CTAS atomique en transaction. Le SELECT * (sans WHERE) force un
	// table-scan complet qui court-circuite l'index ART corrompu ; la nouvelle
	// table reçoit un ART vierge sur des données complètes. La transaction garantit
	// qu'un crash entre le DROP et le RENAME ne peut PAS laisser match_participants
	// (table partagée de tous les joueurs) absente — rollback intégral. Un garde
	// anti-perte vérifie en outre que le rebuild a bien toutes les rows avant de
	// détruire l'original. Cf. revue P0 2026-06-02. `cmd/rebuild_mp` DÉLÈGUE ICI
	// depuis le 2026-08-05 (dette H4) : il n'y a plus de second swap à la main.
	if err := swapMatchParticipantsTx(ctx, db, colList, before); err != nil {
		return err
	}

	// Recrée vues + indexes.
	if err := ApplyResolutionViews(db); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: recreate resolution views: %w", err)
	}
	if err := ApplyMvPlayerMatchesView(db); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: recreate mv_player_matches: %w", err)
	}
	for _, ddl := range matchParticipantsIndexes {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("rebuild_mp_runtime: recreate index: %w", err)
		}
	}

	// Validation post-rebuild : compter à nouveau pour log + sanity check.
	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants`).Scan(&after); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: count after: %w", err)
	}

	slog.InfoContext(ctx, "rebuild_match_participants_runtime: table rebuilt (ART corruption defeated)",
		"rows_before_rebuild", before,
		"rows_after_rebuild", after,
		"columns_preserved", len(cols),
	)
	return nil
}

// recoverOrphanMatchParticipants répare l'état laissé par un crash au milieu d'un
// swap non transactionnel : table principale absente + match_participants__rebuilt
// présent. Promeut le __rebuilt en table principale. No-op si la table principale
// existe ou si aucun __rebuilt orphelin n'est présent. La PK et les vues/indexes
// sont (re)posées par le rebuild normal qui suit.
func recoverOrphanMatchParticipants(ctx context.Context, db *sql.DB) error {
	hasMain, err := tableExists(db, "match_participants")
	if err != nil {
		return fmt.Errorf("rebuild_mp_runtime: check main table: %w", err)
	}
	if hasMain {
		return nil
	}
	hasRebuilt, err := tableExists(db, "match_participants__rebuilt")
	if err != nil {
		return fmt.Errorf("rebuild_mp_runtime: check __rebuilt table: %w", err)
	}
	if !hasRebuilt {
		return nil
	}
	slog.WarnContext(ctx, "rebuild_match_participants_runtime: __rebuilt orphelin détecté (crash mid-swap antérieur) — récupération",
		"action", "RENAME match_participants__rebuilt -> match_participants")
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE match_participants__rebuilt RENAME TO match_participants`); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: recover orphan __rebuilt: %w", err)
	}
	return nil
}

// swapMatchParticipantsTx effectue le swap CTAS dans une transaction unique.
// Garde anti-perte : refuse de détruire l'original si le rebuild n'a pas
// exactement le même nombre de rows que l'original (le SELECT * sans WHERE doit
// faire un table-scan complet). Un échec ou un crash provoque un rollback intégral
// — match_participants reste intacte.
func swapMatchParticipantsTx(ctx context.Context, db *sql.DB, colList string, before int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rebuild_mp_runtime: begin swap tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS match_participants__rebuilt`); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: drop stale __rebuilt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE match_participants__rebuilt AS SELECT %s FROM match_participants`, colList)); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: create __rebuilt: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants__rebuilt`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: count __rebuilt: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("rebuild_mp_runtime: swap abandonné, rebuilt=%d != before=%d (rollback, aucune perte de rows)", rebuilt, before)
	}
	// ⚠ CE QUE `DROP TABLE` NE FAIT PAS, ET QUI A ÉTÉ MESURÉ (2026-08-02, sonde DuckDB,
	// constat J4R-7) : il NE supprime PAS les vues dépendantes, même avec CASCADE — elles
	// restent au catalogue, liées à une table qui n'existe plus. C'est pour cela que la
	// recréation par l'appelant (ApplyResolutionViews / ApplyMvPlayerMatchesView) doit être
	// en `CREATE OR REPLACE` : un `CREATE VIEW` nu échouerait en « View with name "v"
	// already exists! » et ferait avorter la réparation (c'était le défaut de l'ancien
	// cmd/rebuild_mp). Ne pas « simplifier » ces OR REPLACE.
	for _, sqlStmt := range []string{
		`DROP TABLE match_participants`,
		`ALTER TABLE match_participants__rebuilt RENAME TO match_participants`,
		`ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid)`,
	} {
		if _, err := tx.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("rebuild_mp_runtime: swap step (%s): %w", firstWords(sqlStmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rebuild_mp_runtime: commit swap: %w", err)
	}
	committed = true
	return nil
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
