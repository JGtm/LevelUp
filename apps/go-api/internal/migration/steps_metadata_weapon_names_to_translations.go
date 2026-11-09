package migration

// steps_metadata_weapon_names_to_translations.go — les noms d'armes, stockés en colonnes
// appariées `name_en` / `name_fr`, passent dans des tables PAR LANGUE (ADR 0040 D-5, plan i18n
// étape 8.4). Une langue de plus ne demande alors ni colonne ni changement de code : une ligne
// par (arme, langue), la clé de langue étant le CODE de langue de `config/locales.toml` (`fr`,
// `en` : ce sont des noms curés à la main ou posés par l'API par Accept-Language, pas des
// étiquettes Halo d'asset).
//
// Deux tables, deux formes finales :
//   - weapon_name_labels (copie du TOML weapon_names.toml, clé = weapon_key) :
//     `(title_slug, weapon_key, name_en, name_fr)` devient
//     `(title_slug, weapon_key, lang, name)`, clé primaire `(title_slug, weapon_key, lang)` ;
//   - weapon_labels (clé = weapon_id du film) GARDE ses colonnes non textuelles (icon_url,
//     weapon_type...) et perd `name_en` / `name_fr` ; les noms passent dans la table neuve
//     weapon_label_names `(weapon_id, lang, name)`, clé primaire `(weapon_id, lang)`.
//
// Garanties. Une ligne déjà présente pour (clé, langue) l'emporte sur la colonne. Rien n'est
// perdu : avant tout swap, chaque nom non vide d'une colonne doit avoir sa ligne par langue,
// sinon la transaction est annulée. Les colonnes de weapon_labels sont retirées par rebuild
// CTAS-swap (DuckDB refuse `DROP COLUMN` sous PK), schéma recopié depuis le catalogue. Une
// table absente ou déjà migrée ne bouge pas : idempotente. Une transaction par table.
//
// Les DDL des deux tables par langue vivent ICI (source unique, exportée : WeaponTablesDDL) ; le
// paquet games/weapons, qui les crée sur une base neuve, les reprend.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/i18n"
)

// WeaponNamesStepName est le nom du step, réutilisé par le jeu de migrations de Halo 5 (le
// registre global ne s'applique pas à son target metadata).
const WeaponNamesStepName = "metadata_weapon_names_to_translations_v1"

// Noms des tables d'armes touchées par le step.
const (
	tableWeaponLabels     = "weapon_labels"
	tableWeaponLabelNames = "weapon_label_names"
	tableWeaponNameLabels = "weapon_name_labels"
)

func init() {
	Register(Migration{
		Name:        WeaponNamesStepName,
		TargetDB:    TargetMetadata,
		Description: "Noms d'armes : weapon_name_labels (title_slug, weapon_key, lang, name) et weapon_label_names (weapon_id, lang, name) par langue, colonnes name_en/name_fr retirées de weapon_labels (ADR 0040 D-5)",
		ApplySchema: applyWeaponNamesToTranslations,
	})
}

// WeaponTablesDDL rend le DDL (idempotent) des deux tables de noms d'armes par langue. Source
// unique : le step ci-dessus reconstruit vers cette forme, games/weapons la crée sur une base neuve.
func WeaponTablesDDL() string {
	return weaponNameLabelsDDL(tableWeaponNameLabels, "IF NOT EXISTS") + ";\n" +
		"CREATE TABLE IF NOT EXISTS " + tableWeaponLabelNames + " (" + weaponLabelNamesDefs + ")"
}

const weaponLabelNamesDefs = `weapon_id UBIGINT NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL,
	PRIMARY KEY (weapon_id, lang)`

// weaponNameLabelsDDL rend le CREATE de la forme finale de weapon_name_labels sous le nom donné
// (la table définitive, ou la table de reconstruction). `ifNot` : "" ou "IF NOT EXISTS".
func weaponNameLabelsDDL(name, ifNot string) string {
	if ifNot != "" {
		ifNot += " "
	}
	return "CREATE TABLE " + ifNot + name + ` (
		title_slug VARCHAR NOT NULL, weapon_key VARCHAR NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL,
		PRIMARY KEY (title_slug, weapon_key, lang))`
}

// weaponNameColumn associe une colonne appariée héritée à sa langue.
type weaponNameColumn struct {
	col  string
	lang i18n.Locale
}

// legacyWeaponNameColumns : les deux langues dont les colonnes appariées existaient.
var legacyWeaponNameColumns = []weaponNameColumn{
	{"name_en", i18n.LocaleEN},
	{"name_fr", i18n.LocaleFR},
}

// MigrateWeaponNames porte weapon_name_labels et weapon_labels à la forme par langue (no-op sur
// une table absente ou déjà migrée). Exportée pour games/weapons, qui l'appelle avant d'écrire
// dans ces tables : une base restée à l'ancienne forme (step "done" mais colonnes revenues, ou
// base préconstruite) est portée à la nouvelle au lieu d'échouer sur une colonne `NOT NULL`.
func MigrateWeaponNames(db *sql.DB) error { return applyWeaponNamesToTranslations(db) }

func applyWeaponNamesToTranslations(db *sql.DB) error {
	ctx := bootCtx()
	if err := migrateWeaponNameLabels(ctx, db); err != nil {
		return err
	}
	return migrateWeaponLabels(ctx, db)
}

// presentWeaponNameColumns rend celles des colonnes appariées qui existent encore dans la table
// (vide aussi quand la table est absente).
func presentWeaponNameColumns(db *sql.DB, table string) ([]weaponNameColumn, error) {
	has, err := tableExists(db, table)
	if err != nil {
		return nil, fmt.Errorf("weapon names: check table %s: %w", table, err)
	}
	if !has {
		return nil, nil
	}
	var out []weaponNameColumn
	for _, c := range legacyWeaponNameColumns {
		ok, err := columnExists(db, table, c.col)
		if err != nil {
			return nil, fmt.Errorf("weapon names: check column %s.%s: %w", table, c.col, err)
		}
		if ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// nonEmptyName est la condition SQL « cette colonne porte un nom ».
func nonEmptyName(col string) string {
	return "NULLIF(TRIM(CAST(" + col + " AS VARCHAR)), '') IS NOT NULL"
}

// migrateWeaponNameLabels reconstruit weapon_name_labels vers (title_slug, weapon_key, lang, name).
func migrateWeaponNameLabels(ctx context.Context, db *sql.DB) (err error) {
	cols, err := presentWeaponNameColumns(db, tableWeaponNameLabels)
	if err != nil || len(cols) == 0 {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("weapon names %s: begin tx: %w", tableWeaponNameLabels, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	tmp := tableWeaponNameLabels + "__lang_new"
	for _, s := range []string{`DROP TABLE IF EXISTS ` + tmp, weaponNameLabelsDDL(tmp, "")} {
		if _, err = tx.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("weapon names %s: création de %s: %w", tableWeaponNameLabels, tmp, err)
		}
	}
	total := 0
	for _, c := range cols {
		var n int
		if n, err = copyWeaponNameLabelsColumn(ctx, tx, tmp, c); err != nil {
			return err
		}
		total += n
	}
	var rebuilt int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+tmp).Scan(&rebuilt); err != nil {
		return fmt.Errorf("weapon names %s: count: %w", tableWeaponNameLabels, err)
	}
	if rebuilt != total {
		err = fmt.Errorf("weapon names %s: %d lignes écrites au lieu de %d, migration annulée", tableWeaponNameLabels, rebuilt, total)
		return err
	}
	for _, s := range []string{`DROP TABLE ` + tableWeaponNameLabels, `ALTER TABLE ` + tmp + ` RENAME TO ` + tableWeaponNameLabels} {
		if _, err = tx.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("weapon names %s: swap (%s): %w", tableWeaponNameLabels, firstWords(s, 3), err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("weapon names %s: commit: %w", tableWeaponNameLabels, err)
	}
	slog.InfoContext(ctx, "weapon names: weapon_name_labels reconstruite par langue",
		"rows", rebuilt, "columns_dropped", len(cols))
	return nil
}

// copyWeaponNameLabelsColumn copie une colonne appariée vers la table de reconstruction, une
// ligne par nom non vide, et vérifie la couverture. Rend le nombre de lignes de la langue.
func copyWeaponNameLabelsColumn(ctx context.Context, tx *sql.Tx, tmp string, c weaponNameColumn) (int, error) {
	lang := string(c.lang)
	res, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %[1]s (title_slug, weapon_key, lang, name)
		SELECT title_slug, weapon_key, ?, %[2]s FROM %[3]s WHERE %[4]s`,
		tmp, c.col, tableWeaponNameLabels, nonEmptyName(c.col)), lang)
	if err != nil {
		return 0, fmt.Errorf("weapon names %s (%s): copie: %w", tableWeaponNameLabels, c.lang, err)
	}
	n, _ := res.RowsAffected()
	var missing int
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %[1]s o WHERE %[2]s
		AND NOT EXISTS (SELECT 1 FROM %[3]s n WHERE n.title_slug = o.title_slug AND n.weapon_key = o.weapon_key AND n.lang = ?)`,
		tableWeaponNameLabels, nonEmptyName(c.col), tmp), lang).Scan(&missing)
	if err != nil {
		return 0, fmt.Errorf("weapon names %s (%s): contrôle de couverture: %w", tableWeaponNameLabels, c.lang, err)
	}
	if missing > 0 {
		return 0, fmt.Errorf("weapon names %s (%s): %d noms sans ligne par langue, migration annulée", tableWeaponNameLabels, c.lang, missing)
	}
	return int(n), nil
}

// migrateWeaponLabels copie les noms de weapon_labels dans weapon_label_names puis retire les
// colonnes appariées de weapon_labels (rebuild CTAS-swap).
func migrateWeaponLabels(ctx context.Context, db *sql.DB) (err error) {
	cols, err := presentWeaponNameColumns(db, tableWeaponLabels)
	if err != nil || len(cols) == 0 {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("weapon names %s: begin tx: %w", tableWeaponLabels, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS `+tableWeaponLabelNames+` (`+weaponLabelNamesDefs+`)`); err != nil {
		return fmt.Errorf("weapon names %s: création: %w", tableWeaponLabelNames, err)
	}
	copied, drop := 0, make([]string, 0, len(cols))
	for _, c := range cols {
		var n int
		if n, err = copyWeaponLabelsColumn(ctx, tx, c); err != nil {
			return err
		}
		copied += n
		drop = append(drop, c.col)
	}
	var kept int
	if kept, err = rebuildWithoutColumns(ctx, tx, tableWeaponLabels, drop); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("weapon names %s: commit: %w", tableWeaponLabels, err)
	}
	slog.InfoContext(ctx, "weapon names: noms de weapon_labels migrés vers weapon_label_names",
		"rows", kept, "names_inserted", copied, "columns_dropped", len(drop))
	return nil
}

// copyWeaponLabelsColumn insère, pour la langue de la colonne, les lignes de weapon_label_names
// manquantes (une ligne déjà présente pour (arme, langue) n'est jamais modifiée) et vérifie la
// couverture. Rend le nombre de lignes insérées.
func copyWeaponLabelsColumn(ctx context.Context, tx *sql.Tx, c weaponNameColumn) (int, error) {
	lang := string(c.lang)
	res, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %[1]s (weapon_id, lang, name)
		SELECT t.weapon_id, ?, MIN(t.%[2]s) FROM %[3]s t WHERE %[4]s
		  AND NOT EXISTS (SELECT 1 FROM %[1]s n WHERE n.weapon_id = t.weapon_id AND n.lang = ?)
		GROUP BY t.weapon_id`,
		tableWeaponLabelNames, c.col, tableWeaponLabels, nonEmptyName("t."+c.col)), lang, lang)
	if err != nil {
		return 0, fmt.Errorf("weapon names %s (%s): copie: %w", tableWeaponLabels, c.lang, err)
	}
	n, _ := res.RowsAffected()
	var missing int
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %[1]s t WHERE %[2]s
		AND NOT EXISTS (SELECT 1 FROM %[3]s n WHERE n.weapon_id = t.weapon_id AND n.lang = ?)`,
		tableWeaponLabels, nonEmptyName("t."+c.col), tableWeaponLabelNames), lang).Scan(&missing)
	if err != nil {
		return 0, fmt.Errorf("weapon names %s (%s): contrôle de couverture: %w", tableWeaponLabels, c.lang, err)
	}
	if missing > 0 {
		return 0, fmt.Errorf("weapon names %s (%s): %d noms sans ligne par langue, migration annulée", tableWeaponLabels, c.lang, missing)
	}
	return int(n), nil
}
