package migration

// steps_shared_recompute_mode_category.go - shared_recompute_mode_category_v1 : remet
// match_registry.mode_category d'accord avec la règle canonique (analysis/modelabel,
// InferCategory) appliquée au pair_name d'aujourd'hui, et retire l'index qui couvrait la colonne.
//
// POURQUOI. La colonne était écrite à l'insertion par un classifieur par sous-chaînes (disparu)
// qui divergeait de celui de l'interface, sur un nom de paire parfois encore égal à son
// identifiant : 3 111 lignes sur 9 230 fausses à la mesure du 2026-10-09. Le nom se résolvait
// plus tard, la catégorie jamais.
//
// REMÈDE SANS UPDATE (une colonne indexée mise à jour est un vecteur du bug DuckDB ART #23645) :
// reconstruction à côté avec le DDL exact de la table, la seule colonne mode_category remplacée
// par la valeur canonique des paires fausses, échange dans une transaction (swapTableTx), index
// reposés après le RENAME SAUF celui qui couvre mode_category. Une colonne sans index se réécrit
// ensuite avec le nom de la paire (persist.RegistryNamesPersister).
//
// GARDES, tous avant le COMMIT : cardinalité de la table, DDL identique, index identiques hors
// celui retiré, empreinte de la table et de chaque vue dépendante identique HORS mode_category,
// plus aucune paire fausse. Une ligne dont la catégorie est vide ou NULL (titre qui ne la
// renseigne pas) n'est jamais touchée. No-op sans écriture sur une base déjà juste et sans
// index ; sur une base juste mais indexée, seul l'index est retiré.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/modelabel"
)

const (
	registryTable      = "match_registry"
	modeCategoryColumn = "mode_category"
	modeCategorySuffix = "__modecat"
)

func init() {
	Register(Migration{
		Name:     "shared_recompute_mode_category_v1",
		TargetDB: TargetShared,
		Description: "Recalcule match_registry.mode_category depuis pair_name par la règle canonique " +
			"(swap sans UPDATE) et retire idx_registry_mode_category",
		ApplySchema: applyRecomputeModeCategory,
	})
}

// pairCategory : un couple (nom de paire, catégorie stockée) et le nombre de lignes qui le portent.
type pairCategory struct {
	pair, stored string
	rows         int64
}

func applyRecomputeModeCategory(db *sql.DB) error {
	ctx := bootCtx()
	if err := recoverOrphanTable(ctx, db, registryTable, modeCategorySuffix); err != nil {
		return err
	}
	if ok, err := tableExists(db, registryTable); err != nil || !ok {
		return err
	}
	if ok, err := columnExists(db, registryTable, modeCategoryColumn); err != nil || !ok {
		return err
	}
	pairs, err := readPairCategories(ctx, db)
	if err != nil {
		return err
	}
	wrong, wrongRows := wrongCategories(pairs)
	covering, err := coveringIndexNames(ctx, db)
	if err != nil {
		return err
	}
	switch {
	case wrongRows == 0 && len(covering) == 0:
		return nil
	case wrongRows == 0:
		return dropCoveringIndexes(ctx, db, covering)
	}
	n, err := swapRecomputedCategories(ctx, db, wrong)
	if err != nil {
		return err
	}
	slog.WarnContext(ctx, "migration: mode_category recalculée depuis pair_name (swap, sans UPDATE)",
		"table", registryTable, "paires_fausses", len(wrong), "lignes_corrigees", wrongRows,
		"lignes", n.Rebuilt, "index_retires", covering)
	return nil
}

// readPairCategories lit les couples (pair_name, mode_category) des lignes dont la catégorie est
// renseignée : celles qu'un titre qui la renseigne a écrites.
func readPairCategories(ctx context.Context, q lecteurSQL) ([]pairCategory, error) {
	rows, err := q.QueryContext(ctx, `SELECT pair_name, mode_category, COUNT(*) FROM match_registry
		WHERE pair_name IS NOT NULL AND NULLIF(mode_category, '') IS NOT NULL GROUP BY 1, 2`)
	if err != nil {
		return nil, fmt.Errorf("%s: paires et catégories: %w", registryTable, err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []pairCategory
	for rows.Next() {
		var p pairCategory
		if err := rows.Scan(&p.pair, &p.stored, &p.rows); err != nil {
			return nil, fmt.Errorf("%s: paires et catégories: %w", registryTable, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: paires et catégories: %w", registryTable, err)
	}
	return out, nil
}

// wrongCategories rend, par nom de paire, la catégorie canonique des paires dont une ligne porte
// autre chose, et le nombre de lignes concernées.
func wrongCategories(pairs []pairCategory) (map[string]string, int64) {
	wrong := map[string]string{}
	var lines int64
	for _, p := range pairs {
		if want := modelabel.InferCategory(p.pair); want != p.stored {
			wrong[p.pair] = want
			lines += p.rows
		}
	}
	return wrong, lines
}

// categoryCaseExpr : l'expression qui remplace mode_category par la valeur canonique des paires
// fausses et laisse toute autre ligne telle quelle. Littéraux échappés, ordre déterministe.
func categoryCaseExpr(wrong map[string]string) string {
	pairs := make([]string, 0, len(wrong))
	for p := range wrong {
		pairs = append(pairs, p)
	}
	sort.Strings(pairs)
	var b strings.Builder
	b.WriteString("CASE pair_name")
	for _, p := range pairs {
		fmt.Fprintf(&b, " WHEN %s THEN %s", sqlLiteral(p), sqlLiteral(wrong[p]))
	}
	b.WriteString(" ELSE mode_category END")
	return b.String()
}

func sqlLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// coveringIndexNames : les index de match_registry dont le DDL cite mode_category.
func coveringIndexNames(ctx context.Context, q lecteurSQL) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT index_name FROM duckdb_indexes()
		WHERE schema_name = 'main' AND table_name = ? AND sql ILIKE '%mode_category%' ORDER BY index_name`,
		registryTable)
	if err != nil {
		return nil, fmt.Errorf("%s: index de mode_category: %w", registryTable, err)
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%s: index de mode_category: %w", registryTable, err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func dropCoveringIndexes(ctx context.Context, db *sql.DB, names []string) error {
	for _, name := range names {
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS "`+strings.ReplaceAll(name, `"`, `""`)+`"`); err != nil {
			return fmt.Errorf("%s: retrait de l'index %s: %w", registryTable, name, err)
		}
	}
	slog.WarnContext(ctx, "migration: index de mode_category retirés (catégorie déjà juste)",
		"table", registryTable, "index", names)
	return nil
}

// keptIndexDDL : les CREATE INDEX de la table qui ne citent pas mode_category.
func keptIndexDDL(index []string) []string {
	var kept []string
	for _, ddl := range index {
		if !strings.Contains(strings.ToLower(ddl), modeCategoryColumn) {
			kept = append(kept, ddl)
		}
	}
	return kept
}

func swapRecomputedCategories(ctx context.Context, db *sql.DB, wrong map[string]string) (swapCounts, error) {
	ddl, err := ddlDeTable(ctx, db, registryTable)
	if err != nil {
		return swapCounts{}, err
	}
	creer, err := ddlDeConstructionSuffixe(ddl, registryTable, modeCategorySuffix)
	if err != nil {
		return swapCounts{}, err
	}
	index, err := ddlDesIndex(ctx, db, registryTable)
	if err != nil {
		return swapCounts{}, err
	}
	kept := keptIndexDDL(index)
	avant, err := fingerprintsHorsCategorie(ctx, db)
	if err != nil {
		return swapCounts{}, err
	}
	n, err := swapTableTx(ctx, db, tableSwap{
		Table:    registryTable,
		Suffix:   modeCategorySuffix,
		Expected: `SELECT COUNT(*) FROM match_registry`,
		Build: []string{creer, fmt.Sprintf(`INSERT INTO %s%s SELECT * REPLACE (%s AS mode_category) FROM %s`,
			registryTable, modeCategorySuffix, categoryCaseExpr(wrong), registryTable)},
		PostRename: kept,
		Verify: func(ctx context.Context, tx *sql.Tx) error {
			return verifierRecalcul(ctx, tx, ddl, kept, avant)
		},
	})
	if err != nil {
		return n, fmt.Errorf("recalcul de %s.%s: %w", registryTable, modeCategoryColumn, err)
	}
	return n, nil
}

// verifierRecalcul : DDL et index (hors celui retiré) identiques, empreintes de la table et des
// vues identiques hors mode_category, plus aucune paire fausse.
func verifierRecalcul(ctx context.Context, tx *sql.Tx, ddl string, kept []string,
	avant map[string]EmpreinteVue) error {
	if err := verifierSchemaIdentique(ctx, tx, registryTable, ddl, kept); err != nil {
		return err
	}
	apres, err := fingerprintsHorsCategorie(ctx, tx)
	if err != nil {
		return err
	}
	for name, want := range avant {
		if got, ok := apres[name]; !ok || got != want {
			return fmt.Errorf("%s changé hors mode_category : avant %+v, après %+v", name, want, got)
		}
	}
	pairs, err := readPairCategories(ctx, tx)
	if err != nil {
		return err
	}
	if _, rows := wrongCategories(pairs); rows != 0 {
		return fmt.Errorf("%d ligne(s) encore fausses après recalcul", rows)
	}
	return nil
}

// fingerprintsHorsCategorie : empreinte de match_registry et de chaque vue qui la lit, calculée
// sans la colonne mode_category (la seule qui doit changer).
func fingerprintsHorsCategorie(ctx context.Context, q lecteurSQL) (map[string]EmpreinteVue, error) {
	names := []string{registryTable}
	rows, err := q.QueryContext(ctx, `SELECT view_name FROM duckdb_views()
		WHERE NOT internal AND sql ILIKE '%match_registry%' ORDER BY view_name`)
	if err != nil {
		return nil, fmt.Errorf("vues de %s: %w", registryTable, err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("vues de %s: %w", registryTable, err)
		}
		names = append(names, v)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("vues de %s: %w", registryTable, err)
	}
	_ = rows.Close()
	out := make(map[string]EmpreinteVue, len(names))
	for _, name := range names {
		e, err := empreinteHorsColonne(ctx, q, name, modeCategoryColumn)
		if err != nil {
			return nil, err
		}
		out[name] = e
	}
	return out, nil
}

// empreinteHorsColonne : empreinte d'une table ou d'une vue sans `col` (toutes les colonnes si
// elle n'en porte pas).
func empreinteHorsColonne(ctx context.Context, q lecteurSQL, relation, col string) (EmpreinteVue, error) {
	var has int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM duckdb_columns()
		WHERE table_name = ? AND column_name = ?`, relation, col).Scan(&has); err != nil {
		return EmpreinteVue{}, fmt.Errorf("colonnes de %s: %w", relation, err)
	}
	cols := "*"
	if has > 0 {
		cols = "* EXCLUDE (" + col + ")"
	}
	var e EmpreinteVue
	err := q.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*),
		CAST(COALESCE(SUM(CAST(hash(v) AS HUGEINT)), 0) AS VARCHAR) FROM (SELECT %s FROM %s) v`, cols, relation)).
		Scan(&e.Lignes, &e.Somme)
	if err != nil {
		return e, fmt.Errorf("empreinte de %s: %w", relation, err)
	}
	return e, nil
}
