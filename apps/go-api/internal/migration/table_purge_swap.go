package migration

// table_purge_swap.go — retirer des lignes d'une table SANS DELETE (bug DuckDB ART #23645,
// ADR 0026) : la table est reconstruite à côté avec son DDL exact (`duckdb_tables().sql` :
// colonnes, défauts, clé primaire, NOT NULL), remplie des seules lignes gardées, puis échangée
// dans une transaction (swapTableTx) ; ses index sont reposés après le RENAME. Avant le COMMIT :
// DDL et index identiques au caractère près, et, si la table a une vue `_latest`, la vue rend
// exactement ce qu'elle rendait avant sur les lignes gardées. Les valeurs gardées (ids compris)
// et les séquences ne bougent pas.
//
// Utilisé par les steps de purge ciblée (lignes à match_id composite de match_vehicle_takes,
// clés de credential héritées de sync_meta).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// purgeSuffix — suffixe de la table de construction d'une purge par swap.
const purgeSuffix = "__purge"

// swapKeepingRows reconstruit `table` réduite aux lignes de `keep` (clause WHERE complète) et
// l'échange. `view` vide : pas de vue à vérifier.
func swapKeepingRows(ctx context.Context, db *sql.DB, table, view, keep string) (swapCounts, error) {
	ddl, err := ddlDeTable(ctx, db, table)
	if err != nil {
		return swapCounts{}, err
	}
	creer, err := ddlDeConstructionSuffixe(ddl, table, purgeSuffix)
	if err != nil {
		return swapCounts{}, err
	}
	index, err := ddlDesIndex(ctx, db, table)
	if err != nil {
		return swapCounts{}, err
	}
	var attendue EmpreinteVue
	if view != "" {
		if attendue, err = empreinteRestreinte(ctx, db, view, keep); err != nil {
			return swapCounts{}, err
		}
	}
	n, err := swapTableTx(ctx, db, tableSwap{
		Table:    table,
		Suffix:   purgeSuffix,
		Expected: fmt.Sprintf(`SELECT COUNT(*) FROM %s %s`, table, keep),
		Build: []string{creer, fmt.Sprintf(`INSERT INTO %s%s SELECT * FROM %s %s`,
			table, purgeSuffix, table, keep)},
		PostRename: index,
		Verify: func(ctx context.Context, tx *sql.Tx) error {
			if err := verifierSchemaIdentique(ctx, tx, table, ddl, index); err != nil {
				return err
			}
			if view == "" {
				return nil
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
