// seed_citation_reconcile.go : alignement de citation_mappings et des textes de citation sur le
// seed embarqué (defaultCitationMappings, citation_texts.toml).
package ops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// citationRowColumns liste, dans l'ordre de citationArgs et de citationRow, les 13 colonnes de
// citation_mappings hors clé.
const citationRowColumns = `mapping_type, medal_id, medal_ids, stat_name, award_name, award_category,
	custom_function, composite_children, enabled, image_path, category, tier_targets, subcategory`

const (
	citationInsertQ = `INSERT INTO citation_mappings (
		citation_name_norm, mapping_type,
		medal_id, medal_ids, stat_name, award_name, award_category,
		custom_function, composite_children, enabled,
		image_path, category, tier_targets, subcategory
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	citationUpdateQ = `UPDATE citation_mappings SET
		mapping_type = ?, medal_id = ?, medal_ids = ?,
		stat_name = ?, award_name = ?, award_category = ?, custom_function = ?,
		composite_children = ?, enabled = ?, image_path = ?, category = ?,
		tier_targets = ?, subcategory = ?
	WHERE citation_name_norm = ?`
)

// citationRow est l'image d'une ligne de citation_mappings hors clé. Les NULL sont gardés
// distincts des chaînes vides : une ligne n'est « identique » que si chaque colonne l'est, NULL
// compris. Comparable avec ==.
type citationRow struct {
	mappingType       string
	medalID           sql.NullInt64
	medalIDs          sql.NullString
	statName          sql.NullString
	awardName         sql.NullString
	awardCategory     sql.NullString
	customFunction    sql.NullString
	compositeChildren sql.NullString
	enabled           bool
	imagePath         sql.NullString
	category          sql.NullString
	tierTargets       sql.NullString
	subcategory       sql.NullString
}

func nullString(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// citationRowOf rend la ligne que le seed pose pour m, avec les conventions d'écriture de
// citationArgs : chaîne vide et identifiant de médaille nul = NULL, sauf la catégorie, toujours
// écrite.
func citationRowOf(m CitationMapping) citationRow {
	return citationRow{
		mappingType:       m.MappingType,
		medalID:           sql.NullInt64{Int64: m.MedalID, Valid: m.MedalID > 0},
		medalIDs:          nullString(m.MedalIDs),
		statName:          nullString(m.StatName),
		awardName:         nullString(m.AwardName),
		awardCategory:     nullString(m.AwardCategory),
		customFunction:    nullString(m.CustomFunction),
		compositeChildren: nullString(m.CompositeChildren),
		enabled:           m.Enabled,
		imagePath:         nullString(m.ImagePath),
		category:          sql.NullString{String: m.Category, Valid: true},
		tierTargets:       nullString(m.TierTargets),
		subcategory:       nullString(m.Subcategory),
	}
}

// citationArgs rend les 13 valeurs de colonne de m, dans l'ordre de citationRowColumns.
func citationArgs(m CitationMapping) []any {
	var medal any
	if m.MedalID > 0 {
		medal = uint64(m.MedalID) //nolint:gosec // identifiant de médaille strictement positif
	}
	return []any{
		m.MappingType, medal, nullStr(m.MedalIDs),
		nullStr(m.StatName), nullStr(m.AwardName), nullStr(m.AwardCategory),
		nullStr(m.CustomFunction), nullStr(m.CompositeChildren), m.Enabled,
		nullStr(m.ImagePath), m.Category,
		nullStr(m.TierTargets), nullStr(m.Subcategory),
	}
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// loadCitationRows lit toutes les lignes de citation_mappings, par citation_name_norm.
func loadCitationRows(ctx context.Context, db *sql.DB) (map[string]citationRow, error) {
	rows, err := db.QueryContext(ctx, `SELECT citation_name_norm, `+citationRowColumns+` FROM citation_mappings`)
	if err != nil {
		return nil, fmt.Errorf("scan existing: %w", err)
	}
	defer rows.Close()
	out := make(map[string]citationRow)
	for rows.Next() {
		var norm string
		var r citationRow
		if err := rows.Scan(&norm, &r.mappingType, &r.medalID, &r.medalIDs, &r.statName, &r.awardName,
			&r.awardCategory, &r.customFunction, &r.compositeChildren, &r.enabled, &r.imagePath,
			&r.category, &r.tierTargets, &r.subcategory); err != nil {
			return nil, fmt.Errorf("scan existing: %w", err)
		}
		out[norm] = r
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan existing: %w", err)
	}
	return out, nil
}

// citationTextKey identifie une ligne de asset_translations de type citation.
type citationTextKey struct{ norm, lang string }

// citationTextRow est le nom et la description stockés pour une citation et une langue.
type citationTextRow struct{ name, description sql.NullString }

// loadCitationTexts lit toutes les lignes de asset_translations de type citation.
func loadCitationTexts(ctx context.Context, db *sql.DB) (map[citationTextKey]citationTextRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT asset_id, lang, name, description FROM asset_translations WHERE asset_type = ?`,
		migration.AssetTypeCitation)
	if err != nil {
		return nil, fmt.Errorf("lecture des textes de citations: %w", err)
	}
	defer rows.Close()
	out := make(map[citationTextKey]citationTextRow)
	for rows.Next() {
		var k citationTextKey
		var r citationTextRow
		if err := rows.Scan(&k.norm, &k.lang, &r.name, &r.description); err != nil {
			return nil, fmt.Errorf("lecture des textes de citations: %w", err)
		}
		out[k] = r
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lecture des textes de citations: %w", err)
	}
	return out, nil
}

// staleCitationTexts rend, pour la citation norm, les noms et descriptions des seules langues
// dont la ligne de asset_translations manque ou diffère du texte attendu. La comparaison suit
// UpsertAssetTexts : textes rognés, langue sans texte ignorée, description vide = NULL. Une
// langue sans étiquette Halo est rendue telle quelle : l'écriture refuse alors avec son erreur.
func staleCitationTexts(stored map[citationTextKey]citationTextRow, norm string, names, descriptions i18n.Text) (i18n.Text, i18n.Text) {
	staleNames, staleDescs := i18n.Text{}, i18n.Text{}
	for _, l := range i18n.Locales() {
		name, desc := strings.TrimSpace(names[l]), strings.TrimSpace(descriptions[l])
		if name == "" && desc == "" {
			continue
		}
		if tags := i18n.HaloTags(l); len(tags) > 0 {
			cur, ok := stored[citationTextKey{norm: norm, lang: tags[0]}]
			if ok && cur.name == (sql.NullString{String: name, Valid: true}) && cur.description == nullString(desc) {
				continue
			}
		}
		staleNames[l], staleDescs[l] = name, desc
	}
	return staleNames, staleDescs
}

// validateCitationTexts valide les textes embarqués AVANT toute écriture : un fichier incomplet
// ne laisse pas une base à moitié seedée.
func validateCitationTexts() error {
	if _, err := loadedCitationTexts(); err != nil {
		return fmt.Errorf("textes des citations: %w", err)
	}
	return nil
}

// ensureCitationSchema crée citation_mappings à son schéma complet si elle manque, compatible avec
// les migrations add_citation_mappings, add_citation_mappings_pk et
// add_citation_mappings_v2_fields, puis porte à la nouvelle forme une base restée à l'ancienne
// (colonnes de texte revenues, base préconstruite) : sinon l'INSERT échouerait sur une colonne
// `NOT NULL`.
func ensureCitationSchema(ctx context.Context, db *sql.DB) error {
	var hasTranslations int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'asset_translations'`).Scan(&hasTranslations); err != nil {
		return fmt.Errorf("vérification asset_translations: %w", err)
	}
	if hasTranslations == 0 {
		return errors.New("asset_translations absente : lancer d'abord les migrations (les textes des citations y sont écrits)")
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS citation_mappings (
			citation_name_norm    VARCHAR PRIMARY KEY,
			mapping_type          VARCHAR NOT NULL DEFAULT 'medal',
			medal_id              UBIGINT,
			medal_ids             VARCHAR,
			stat_name             VARCHAR,
			award_name            VARCHAR,
			award_category        VARCHAR,
			custom_function       VARCHAR,
			composite_children    VARCHAR,
			enabled               BOOLEAN NOT NULL DEFAULT TRUE,
			image_path            VARCHAR,
			category              VARCHAR,
			tier_targets          VARCHAR,
			subcategory           VARCHAR
		);
		-- PAS d'index sur medal_id/mapping_type : mutés par le seed (SELECT-then-write) → surface ART.
		-- Drop DBs existantes : drop_metadata_art_surface_indexes_v4.
		CREATE INDEX IF NOT EXISTS idx_citation_mappings_norm ON citation_mappings(citation_name_norm);
	`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if err := migration.MigrateCuratedTexts(db); err != nil {
		return fmt.Errorf("textes de citations: %w", err)
	}
	return nil
}

// reconcileCitation aligne la ligne et les textes de la citation m. Rend present (la citation
// existait) et updated (elle existait et une colonne ou un texte a été réécrit). Une ligne et des
// textes identiques ne déclenchent aucune écriture. Écriture ART-safe : SELECT-then-INSERT-or-UPDATE
// décidé sur les lectures de l'appelant, jamais d'ON CONFLICT.
func reconcileCitation(ctx context.Context, db *sql.DB, m CitationMapping,
	rows map[string]citationRow, texts map[citationTextKey]citationTextRow,
) (present, updated bool, err error) {
	cur, present := rows[m.Norm]
	switch {
	case !present:
		if _, err := db.ExecContext(ctx, citationInsertQ, append([]any{m.Norm}, citationArgs(m)...)...); err != nil {
			return false, false, fmt.Errorf("upsert %s: %w", m.Norm, err)
		}
	case cur != citationRowOf(m):
		if _, err := db.ExecContext(ctx, citationUpdateQ, append(citationArgs(m), m.Norm)...); err != nil {
			return true, false, fmt.Errorf("upsert %s: %w", m.Norm, err)
		}
		updated = true
	}
	names, descriptions, err := citationTextsFor(m.Norm)
	if err != nil {
		return present, updated, err
	}
	staleNames, staleDescs := staleCitationTexts(texts, m.Norm, names, descriptions)
	if len(staleNames) == 0 && len(staleDescs) == 0 {
		return present, updated, nil
	}
	if err := UpsertAssetTexts(ctx, db, migration.AssetTypeCitation, m.Norm, staleNames, staleDescs); err != nil {
		return present, updated, fmt.Errorf("textes de %s: %w", m.Norm, err)
	}
	// Des textes réécrits sur une citation existante la comptent comme mise à jour.
	return present, updated || present, nil
}

// seedCitationMappingsOn est le cœur du seed des citations : il aligne citation_mappings et les
// textes par langue (asset_translations, type citation, ADR 0040 D-15) sur le seed embarqué, sur
// le handle db reçu, sans l'ouvrir ni le fermer. Chaque ligne et chaque texte sont comparés à
// l'existant avant écriture : une base déjà alignée ne reçoit aucune écriture (index ART de la
// base de référentiels épargnés).
//
// Compte Inserted (citations nouvelles), Skipped (citations déjà présentes) et Updated (parmi
// celles-ci, les citations dont une colonne ou un texte a été réécrit).
func seedCitationMappingsOn(ctx context.Context, db *sql.DB) (SeedResult, error) {
	failed := SeedResult{Component: componentCitationMappings}
	if err := validateCitationTexts(); err != nil {
		return failed, err
	}
	if err := ensureCitationSchema(ctx, db); err != nil {
		return failed, err
	}
	rows, err := loadCitationRows(ctx, db)
	if err != nil {
		return failed, err
	}
	texts, err := loadCitationTexts(ctx, db)
	if err != nil {
		return failed, err
	}
	res := SeedResult{Component: componentCitationMappings}
	for _, m := range defaultCitationMappings() {
		present, updated, err := reconcileCitation(ctx, db, m, rows, texts)
		if err != nil {
			return failed, err
		}
		switch {
		case !present:
			res.Inserted++
		case updated:
			res.Updated++
			res.Skipped++
		default:
			res.Skipped++
		}
	}
	res.Message = fmt.Sprintf("%d insérées, %d déjà présentes dont %d mises à jour", res.Inserted, res.Skipped, res.Updated)
	return res, nil
}

// ReconcileCitationMappings aligne les citations de la base de référentiels db sur le seed
// embarqué, à chaque démarrage : une citation ajoutée par une mise à jour atteint ainsi une base
// déjà installée. db est le handle de l'écrivain unique de metadata.duckdb ; la fonction ne
// l'ouvre pas et ne le ferme pas. Idempotente : sur une base alignée, Inserted et Updated valent 0
// et rien n'est écrit.
func ReconcileCitationMappings(ctx context.Context, db *sql.DB) (SeedResult, error) {
	return seedCitationMappingsOn(ctx, db)
}
