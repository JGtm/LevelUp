package migration

// steps_metadata_texts_to_translations.go — les noms et descriptions de définitions, stockés en
// colonnes appariées `_fr` / `_en`, passent dans les tables PAR LANGUE (ADR 0040 D-5, plan
// i18n étape 8.3) : `medal_translations` pour les médailles, `asset_translations` pour le reste.
//
// Une langue de plus ne demande alors ni colonne ni changement de code : une ligne par
// (identifiant, langue), lue par ses étiquettes Halo (`config/locales.toml`).
//
// Tables traitées (colonne d'identifiant ; type d'actif de `asset_translations`) :
//   - medal_definitions         (medal_name_id)   -> medal_translations
//   - commendation_definitions  (commendation_id) -> « commendation »
//   - team_colors               (team_id)         -> « team_color »
//   - xbox_achievement_definitions (achievement_id) -> « achievement » (nom, description) et
//     « achievement_locked » (texte de succès verrouillé, rangé dans `name`)
//   - milestone_catalog         (id)              -> « milestone » (titre en `name`, condition
//     lisible en `description`)
//
// Garanties. Une ligne de traduction déjà présente pour (identifiant, langue) l'emporte sur la
// colonne (elle vient de l'API Halo ou d'une saisie d'administration : la chaîne de lecture
// d'avant la préférait déjà). Rien n'est perdu : avant le rebuild, chaque texte non vide d'une
// colonne doit avoir sa ligne de traduction, sinon la transaction est annulée. Les colonnes sont
// retirées par rebuild CTAS-swap (DuckDB refuse `DROP COLUMN` sous PK), schéma recopié depuis le
// catalogue (types, défauts, NOT NULL, clé primaire). Idempotente : une table sans colonne
// appariée ne bouge pas. Une transaction par table.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/i18n"
)

// DefinitionTextsStepName est le nom du step, réutilisé par le jeu de migrations de Halo 5
// (le registre global ne s'applique pas à son target metadata).
const DefinitionTextsStepName = "metadata_definition_texts_to_translations_v1"

func init() {
	Register(Migration{
		Name:        DefinitionTextsStepName,
		TargetDB:    TargetMetadata,
		Description: "Noms et descriptions de définitions (médailles, commendations, couleurs d'équipe, succès, jalons) : colonnes _fr/_en -> tables par langue (medal_translations, asset_translations), colonnes retirées (ADR 0040 D-5)",
		ApplySchema: applyDefinitionTextsToTranslations,
	})
}

// nameBase est la base de la colonne `<base>_fr` / `<base>_en` d'un nom ; sqlNull, l'expression SQL
// d'une colonne absente.
const (
	nameBase = "name"
	// descriptionBase est la base de la colonne de description (`description_fr`, `description_en`).
	descriptionBase = "description"
	sqlNull         = "NULL"
)

// textPair décrit les colonnes de nom et de description d'un texte par langue et sa destination dans
// asset_translations. Par défaut les colonnes sont `<base>_fr` / `<base>_en` ; nameCols et descCols
// nomment une colonne explicite par langue quand elle ne suit pas ce suffixe. descBase vide et
// descCols vide : pas de description.
type textPair struct {
	assetType string
	nameBase  string
	descBase  string
	nameCols  map[i18n.Locale]string
	descCols  map[i18n.Locale]string
}

// nameCol retourne la colonne du nom dans la langue l, "" quand le couple n'en a pas.
func (p textPair) nameCol(l i18n.Locale) string { return pairCol(p.nameCols, p.nameBase, l) }

// descCol retourne la colonne de la description dans la langue l, "" quand le couple n'en a pas.
func (p textPair) descCol(l i18n.Locale) string { return pairCol(p.descCols, p.descBase, l) }

func pairCol(explicit map[i18n.Locale]string, base string, l i18n.Locale) string {
	if c, ok := explicit[l]; ok {
		return c
	}
	if base == "" {
		return ""
	}
	return base + "_" + string(l)
}

type definitionTextTable struct {
	table string
	idCol string
	medal bool
	pairs []textPair
}

var definitionTextTables = []definitionTextTable{
	{table: "medal_definitions", idCol: "medal_name_id", medal: true, pairs: []textPair{{nameBase: nameBase, descBase: descriptionBase}}},
	{table: "commendation_definitions", idCol: "commendation_id", pairs: []textPair{{assetType: "commendation", nameBase: nameBase, descBase: descriptionBase}}},
	{table: "team_colors", idCol: "team_id", pairs: []textPair{{assetType: "team_color", nameBase: nameBase}}},
	{table: "xbox_achievement_definitions", idCol: "achievement_id", pairs: []textPair{
		{assetType: "achievement", nameBase: nameBase, descBase: descriptionBase},
		{assetType: "achievement_locked", nameBase: "locked_desc"},
	}},
	{table: "milestone_catalog", idCol: "id", pairs: []textPair{{assetType: "milestone", nameBase: "title", descBase: "condition"}}},
}

// legacyTextLocales : les deux langues dont les colonnes appariées existent (suffixes `_fr` et
// `_en`) ; la destination utilise l'étiquette Halo de chacune.
var legacyTextLocales = []i18n.Locale{i18n.LocaleFR, i18n.LocaleEN}

func applyDefinitionTextsToTranslations(db *sql.DB) error {
	return applyTextTables(db, definitionTextTables)
}

// applyTextTables migre chaque table de `tables` encore dotée de colonnes de texte par langue.
func applyTextTables(db *sql.DB, tables []definitionTextTable) error {
	ctx := bootCtx()
	for _, spec := range tables {
		has, err := tableExists(db, spec.table)
		if err != nil {
			return fmt.Errorf("definition texts: check table %s: %w", spec.table, err)
		}
		if !has {
			continue
		}
		textCols, err := presentTextColumns(db, spec)
		if err != nil {
			return err
		}
		if len(textCols) == 0 {
			continue
		}
		if err := migrateDefinitionTexts(ctx, db, spec, textCols); err != nil {
			return err
		}
	}
	return nil
}

// presentTextColumns retourne les colonnes appariées effectivement présentes dans la table.
func presentTextColumns(db *sql.DB, spec definitionTextTable) ([]string, error) {
	var out []string
	for _, p := range spec.pairs {
		for _, l := range legacyTextLocales {
			for _, col := range []string{p.nameCol(l), p.descCol(l)} {
				if col == "" {
					continue
				}
				ok, err := columnExists(db, spec.table, col)
				if err != nil {
					return nil, fmt.Errorf("definition texts: check column %s.%s: %w", spec.table, col, err)
				}
				if ok {
					out = append(out, col)
				}
			}
		}
	}
	return out, nil
}

func migrateDefinitionTexts(ctx context.Context, db *sql.DB, spec definitionTextTable, textCols []string) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("definition texts %s: begin tx: %w", spec.table, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	copied := 0
	for _, p := range spec.pairs {
		for _, l := range legacyTextLocales {
			n, cerr := copyTextPair(ctx, tx, spec, p, l)
			if cerr != nil {
				return cerr
			}
			copied += n
		}
	}
	if err = checkTextsCovered(ctx, tx, spec); err != nil {
		return err
	}
	var kept int
	if kept, err = rebuildWithoutColumns(ctx, tx, spec.table, textCols); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("definition texts %s: commit: %w", spec.table, err)
	}
	slog.InfoContext(ctx, "definition texts: colonnes appariées migrées vers les tables par langue",
		"table", spec.table, "rows", kept, "translations_inserted", copied, "columns_dropped", len(textCols))
	return nil
}

// textExpr retourne l'expression SQL d'une colonne appariée, ou NULL quand elle n'existe pas.
func textExpr(have map[string]bool, col string) string {
	if col == "" || !have[col] {
		return sqlNull
	}
	return "NULLIF(TRIM(CAST(" + col + " AS VARCHAR)), '')"
}

func tableColumnSet(ctx context.Context, tx *sql.Tx, table string) (map[string]bool, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT column_name FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ?`, table)
	if err != nil {
		return nil, fmt.Errorf("definition texts %s: list columns: %w", table, err)
	}
	defer rows.Close()
	have := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("definition texts %s: scan column: %w", table, err)
		}
		have[c] = true
	}
	return have, rows.Err()
}

// pairExprs rend les expressions (nom, description) SQL d'un couple pour une langue ; le texte de
// succès verrouillé n'a pas de description : son texte va dans `name`.
func pairExprs(have map[string]bool, p textPair, l i18n.Locale) (name, desc string) {
	name, desc = textExpr(have, p.nameCol(l)), textExpr(have, p.descCol(l))
	if p.assetType == "achievement_locked" {
		desc = sqlNull
	}
	return name, desc
}

// copyTextPair insère, pour la langue l, les lignes de traduction manquantes d'un couple de
// colonnes. Une ligne déjà présente pour (identifiant, étiquette) n'est jamais modifiée.
func copyTextPair(ctx context.Context, tx *sql.Tx, spec definitionTextTable, p textPair, l i18n.Locale) (int, error) {
	have, err := tableColumnSet(ctx, tx, spec.table)
	if err != nil {
		return 0, err
	}
	name, desc := pairExprs(have, p, l)
	if name == sqlNull && desc == sqlNull {
		return 0, nil
	}
	tags := i18n.HaloTags(l)
	if len(tags) == 0 {
		return 0, fmt.Errorf("definition texts %s: langue %s sans étiquette Halo", spec.table, l)
	}
	lang := tags[0]
	var q string
	var args []any
	if spec.medal {
		q = fmt.Sprintf(`INSERT INTO medal_translations (medal_name_id, lang, name, description)
			SELECT t.%[1]s, ?, COALESCE(%[2]s, ''), %[3]s FROM %[4]s t
			WHERE (%[2]s IS NOT NULL OR %[3]s IS NOT NULL)
			  AND NOT EXISTS (SELECT 1 FROM medal_translations m WHERE m.medal_name_id = t.%[1]s AND m.lang = ?)`,
			spec.idCol, name, desc, spec.table)
		args = []any{lang, lang}
	} else {
		q = fmt.Sprintf(`INSERT INTO asset_translations (asset_id, asset_type, lang, name, description)
			SELECT CAST(t.%[1]s AS VARCHAR), ?, ?, COALESCE(%[2]s, ''), %[3]s FROM %[4]s t
			WHERE (%[2]s IS NOT NULL OR %[3]s IS NOT NULL)
			  AND NOT EXISTS (SELECT 1 FROM asset_translations a
			                  WHERE a.asset_id = CAST(t.%[1]s AS VARCHAR) AND a.asset_type = ? AND a.lang = ?)`,
			spec.idCol, name, desc, spec.table)
		args = []any{p.assetType, lang, p.assetType, lang}
	}
	res, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("definition texts %s (%s, %s): copie: %w", spec.table, p.nameBase, l, err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// checkTextsCovered refuse de retirer les colonnes tant qu'un texte non vide n'a pas sa ligne de
// traduction dans la langue de la colonne.
func checkTextsCovered(ctx context.Context, tx *sql.Tx, spec definitionTextTable) error {
	have, err := tableColumnSet(ctx, tx, spec.table)
	if err != nil {
		return err
	}
	for _, p := range spec.pairs {
		for _, l := range legacyTextLocales {
			name, desc := pairExprs(have, p, l)
			if name == sqlNull && desc == sqlNull {
				continue
			}
			lang := i18n.HaloTags(l)[0]
			var q string
			var args []any
			if spec.medal {
				q = fmt.Sprintf(`SELECT COUNT(*) FROM %[1]s t WHERE (%[2]s IS NOT NULL OR %[3]s IS NOT NULL)
					AND NOT EXISTS (SELECT 1 FROM medal_translations m WHERE m.medal_name_id = t.%[4]s AND m.lang = ?)`,
					spec.table, name, desc, spec.idCol)
				args = []any{lang}
			} else {
				q = fmt.Sprintf(`SELECT COUNT(*) FROM %[1]s t WHERE (%[2]s IS NOT NULL OR %[3]s IS NOT NULL)
					AND NOT EXISTS (SELECT 1 FROM asset_translations a WHERE a.asset_id = CAST(t.%[4]s AS VARCHAR)
					                AND a.asset_type = ? AND a.lang = ?)`,
					spec.table, name, desc, spec.idCol)
				args = []any{p.assetType, lang}
			}
			var missing int
			if err := tx.QueryRowContext(ctx, q, args...).Scan(&missing); err != nil {
				return fmt.Errorf("definition texts %s: contrôle de couverture: %w", spec.table, err)
			}
			if missing > 0 {
				return fmt.Errorf("definition texts %s (%s): %d textes sans ligne de traduction, migration annulée", spec.table, l, missing)
			}
		}
	}
	return nil
}

// rebuildWithoutColumns recrée `table` sans les colonnes `drop`, schéma lu du catalogue (type,
// défaut, NOT NULL, clé primaire), garde anti-perte sur le nombre de lignes. Retourne ce nombre.
func rebuildWithoutColumns(ctx context.Context, tx *sql.Tx, table string, drop []string) (int, error) {
	defs, keep, err := schemaWithout(ctx, tx, table, drop)
	if err != nil {
		return 0, err
	}
	pk, err := primaryKeyColumns(ctx, tx, table)
	if err != nil {
		return 0, err
	}
	if len(pk) > 0 {
		defs = append(defs, "PRIMARY KEY ("+strings.Join(pk, ", ")+")")
	}
	var before int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&before); err != nil {
		return 0, fmt.Errorf("definition texts %s: count: %w", table, err)
	}
	tmp := table + "__texts_new"
	keepList := strings.Join(keep, ", ")
	for _, s := range []string{
		`DROP TABLE IF EXISTS ` + tmp,
		`CREATE TABLE ` + tmp + ` (` + strings.Join(defs, ", ") + `)`,
		`INSERT INTO ` + tmp + ` (` + keepList + `) SELECT ` + keepList + ` FROM ` + table,
	} {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return 0, fmt.Errorf("definition texts %s: rebuild (%s): %w", table, firstWords(s, 3), err)
		}
	}
	var after int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+tmp).Scan(&after); err != nil {
		return 0, fmt.Errorf("definition texts %s: count rebuild: %w", table, err)
	}
	if after != before {
		return 0, fmt.Errorf("definition texts %s: rebuild abandonné, %d lignes au lieu de %d", table, after, before)
	}
	for _, s := range []string{`DROP TABLE ` + table, `ALTER TABLE ` + tmp + ` RENAME TO ` + table} {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return 0, fmt.Errorf("definition texts %s: swap (%s): %w", table, firstWords(s, 3), err)
		}
	}
	return before, nil
}

// schemaWithout lit le schéma de `table` et rend les définitions de colonne (type, défaut, NOT
// NULL) et les noms de colonne conservés, hors `drop`.
func schemaWithout(ctx context.Context, tx *sql.Tx, table string, drop []string) (defs, keep []string, err error) {
	skip := map[string]bool{}
	for _, c := range drop {
		skip[c] = true
	}
	rows, err := tx.QueryContext(ctx, `SELECT column_name, data_type, column_default, is_nullable
		FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, nil, fmt.Errorf("definition texts %s: lire le schéma: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, typ, nullable string
		var dflt sql.NullString
		if err := rows.Scan(&name, &typ, &dflt, &nullable); err != nil {
			return nil, nil, fmt.Errorf("definition texts %s: scan schéma: %w", table, err)
		}
		if skip[name] {
			continue
		}
		def := name + " " + typ
		if dflt.Valid && dflt.String != "" {
			def += " DEFAULT " + dflt.String
		}
		if nullable == "NO" {
			def += " NOT NULL"
		}
		defs = append(defs, def)
		keep = append(keep, name)
	}
	return defs, keep, rows.Err()
}

func primaryKeyColumns(ctx context.Context, tx *sql.Tx, table string) ([]string, error) {
	var cols any
	err := tx.QueryRowContext(ctx, `SELECT constraint_column_names FROM duckdb_constraints()
		WHERE table_name = ? AND constraint_type = 'PRIMARY KEY' LIMIT 1`, table).Scan(&cols)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("definition texts %s: clé primaire: %w", table, err)
	}
	list, ok := cols.([]any)
	if !ok {
		return nil, fmt.Errorf("definition texts %s: clé primaire illisible (%T)", table, cols)
	}
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, fmt.Sprint(c))
	}
	return out, nil
}
