package migration

// steps_shared_vehicle_takes_purge_composite.go — shared_purge_composite_vehicle_takes_v1 :
// retire de match_vehicle_takes les lignes dont le match_id est une LISTE de préfixes
// (« fccc61cd,879a4dba,… »), écrites quand `levelup backfill-vehicle-takes --match a,b,…`
// prenait encore la valeur brute de l'option pour un identifiant de match (corrigé depuis :
// l'option se résout contre le registre, registreBorne, test
// cmd_backfill_vehicle_takes_match_integration_test.go).
//
// Ces lignes forment leur propre « match » dans la vue match_vehicle_takes_latest : toute
// lecture qui agrège la vue sans joindre le registre (totaux par joueur) les compte. Aucun
// match réel ne porte de virgule dans son identifiant.
//
// Remède SANS DELETE (table append-only, ADR 0026) : reconstruction à côté avec le DDL exact
// de la table, remplie des lignes gardées, index reposés après le RENAME, échange dans une
// transaction (swapTableTx) ; avant le COMMIT, DDL et index identiques au caractère près et
// vue identique à ce qu'elle rendait hors des lignes retirées. Les id gardés et la séquence
// ne bougent pas. No-op (sans écriture) sur une base qui n'a pas de telles lignes.
// Mécanique : swapKeepingRows (table_purge_swap.go).

import (
	"database/sql"
	"fmt"
	"log/slog"
)

const (
	vehicleTakesTable      = "match_vehicle_takes"
	vehicleTakesView       = "match_vehicle_takes_latest"
	vehicleTakesKeepClause = "WHERE strpos(match_id, ',') = 0"
)

func applyPurgeCompositeVehicleTakes(db *sql.DB) error {
	ctx := bootCtx()
	if err := recoverOrphanTable(ctx, db, vehicleTakesTable, purgeSuffix); err != nil {
		return err
	}
	if ok, err := tableExists(db, vehicleTakesTable); err != nil || !ok {
		return err
	}
	var composite int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM %s WHERE strpos(match_id, ',') > 0`, vehicleTakesTable)).Scan(&composite); err != nil {
		return fmt.Errorf("%s: compte des match_id composites: %w", vehicleTakesTable, err)
	}
	if composite == 0 {
		return nil
	}
	n, err := swapKeepingRows(ctx, db, vehicleTakesTable, vehicleTakesView, vehicleTakesKeepClause)
	if err != nil {
		return err
	}
	slog.WarnContext(ctx, "migration: lignes à match_id composite retirées (swap, sans DELETE)",
		"table", vehicleTakesTable, "removed", composite, "kept", n.Rebuilt)
	return nil
}
