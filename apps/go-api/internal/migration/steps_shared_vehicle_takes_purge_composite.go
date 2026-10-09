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

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

const (
	vehicleTakesTable      = "match_vehicle_takes"
	vehicleTakesView       = "match_vehicle_takes_latest"
	vehicleTakesPurgeSufx  = "__purge"
	vehicleTakesKeepClause = "WHERE strpos(match_id, ',') = 0"
)

func applyPurgeCompositeVehicleTakes(db *sql.DB) error {
	ctx := bootCtx()
	if err := recoverOrphanTable(ctx, db, vehicleTakesTable, vehicleTakesPurgeSufx); err != nil {
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

// swapKeepingRows reconstruit `table` avec son DDL exact, réduite aux lignes de `keep`, et
// l'échange. La vue doit rendre après l'échange exactement ce qu'elle rendait, avant, sur les
// lignes gardées.
func swapKeepingRows(ctx context.Context, db *sql.DB, table, view, keep string) (swapCounts, error) {
	ddl, err := ddlDeTable(ctx, db, table)
	if err != nil {
		return swapCounts{}, err
	}
	creer, err := ddlDeConstructionSuffixe(ddl, table, vehicleTakesPurgeSufx)
	if err != nil {
		return swapCounts{}, err
	}
	index, err := ddlDesIndex(ctx, db, table)
	if err != nil {
		return swapCounts{}, err
	}
	attendue, err := empreinteRestreinte(ctx, db, view, keep)
	if err != nil {
		return swapCounts{}, err
	}
	n, err := swapTableTx(ctx, db, tableSwap{
		Table:    table,
		Suffix:   vehicleTakesPurgeSufx,
		Expected: fmt.Sprintf(`SELECT COUNT(*) FROM %s %s`, table, keep),
		Build: []string{creer, fmt.Sprintf(`INSERT INTO %s%s SELECT * FROM %s %s`,
			table, vehicleTakesPurgeSufx, table, keep)},
		PostRename: index,
		Verify: func(ctx context.Context, tx *sql.Tx) error {
			if err := verifierSchemaIdentique(ctx, tx, table, ddl, index); err != nil {
				return err
			}
			apres, err := empreinteDeVue(ctx, tx, view)
			if err != nil {
				return err
			}
			if apres != attendue {
				return fmt.Errorf("la vue %s ne rend plus les lignes gardées : attendu %+v, après %+v",
					view, attendue, apres)
			}
			return nil
		},
	})
	if err != nil {
		return n, fmt.Errorf("purge de %s: %w", table, err)
	}
	return n, nil
}

// empreinteRestreinte : empreinte de la vue restreinte à `keep` (même calcul qu'empreinteDeVue).
func empreinteRestreinte(ctx context.Context, q lecteurSQL, view, keep string) (EmpreinteVue, error) {
	var e EmpreinteVue
	err := q.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*),
		CAST(COALESCE(SUM(CAST(hash(v) AS HUGEINT)), 0) AS VARCHAR) FROM (SELECT * FROM %s %s) v`, view, keep)).
		Scan(&e.Lignes, &e.Somme)
	if err != nil {
		return e, fmt.Errorf("empreinte de %s: %w", view, err)
	}
	return e, nil
}

// verifierSchemaIdentique : DDL de la table et DDL de ses index identiques au caractère près.
func verifierSchemaIdentique(ctx context.Context, q lecteurSQL, table, ddl string, index []string) error {
	apres, err := ddlDeTable(ctx, q, table)
	if err != nil {
		return err
	}
	if apres != ddl {
		return fmt.Errorf("DDL de %s changé :\navant %s\naprès %s", table, ddl, apres)
	}
	idx, err := ddlDesIndex(ctx, q, table)
	if err != nil {
		return err
	}
	if strings.Join(idx, "\n") != strings.Join(index, "\n") {
		return fmt.Errorf("index de %s changés : avant %v, après %v", table, index, idx)
	}
	return nil
}
