package migration

// steps_metadata_texts_to_translations_curated.go — les textes curés par langue de trois
// référentiels de metadata passent dans `asset_translations` (ADR 0040 D-15, plan i18n 8.8.a),
// même recette que metadata_definition_texts_to_translations_v1 (copie sans perte, garde de
// couverture, rebuild CTAS-swap, idempotente, une transaction par table).
//
// Tables traitées (colonne d'identifiant ; type d'actif de `asset_translations`) :
//   - citation_mappings (citation_name_norm) -> « citation » : nom = `citation_name_display` (fr)
//     et `citation_name_display_en` (en), description = `description` (fr) et `description_en` (en) ;
//   - challenge_template (id) -> « challenge_template » : `label_*` et `description_*` ;
//   - preset_arc (id) -> « preset_arc » : `title_*` et `description_*`.
//
// `lang` = première étiquette Halo de la langue (`fr-FR`, `en-US`), comme les autres types
// curés (jalons). Une ligne de traduction déjà présente l'emporte sur la colonne. Une table
// absente ou déjà sans colonne appariée ne bouge pas.

import (
	"database/sql"

	"levelup/go-api/internal/i18n"
)

// CuratedTextsStepName est le nom du step du registre global.
const CuratedTextsStepName = "metadata_curated_texts_to_translations_v1"

// Types d'actifs de asset_translations qui portent les textes curés migrés par ce step.
const (
	AssetTypeCitation          = "citation"
	AssetTypeChallengeTemplate = "challenge_template"
	AssetTypePresetArc         = "preset_arc"
)

func init() {
	Register(Migration{
		Name:        CuratedTextsStepName,
		TargetDB:    TargetMetadata,
		Description: "Textes curés (citations, modèles de défi, arcs préréglés) : colonnes appariées -> asset_translations, colonnes retirées (ADR 0040 D-15)",
		ApplySchema: applyCuratedTextsToTranslations,
	})
}

var curatedTextTables = []definitionTextTable{
	{table: "citation_mappings", idCol: "citation_name_norm", pairs: []textPair{{
		assetType: AssetTypeCitation,
		nameCols:  map[i18n.Locale]string{i18n.LocaleFR: "citation_name_display", i18n.LocaleEN: "citation_name_display_en"},
		descCols:  map[i18n.Locale]string{i18n.LocaleFR: descriptionBase, i18n.LocaleEN: descriptionBase + "_en"},
	}}},
	{table: "challenge_template", idCol: "id", pairs: []textPair{{assetType: AssetTypeChallengeTemplate, nameBase: "label", descBase: descriptionBase}}},
	{table: "preset_arc", idCol: "id", pairs: []textPair{{assetType: AssetTypePresetArc, nameBase: "title", descBase: descriptionBase}}},
}

func applyCuratedTextsToTranslations(db *sql.DB) error {
	return applyTextTables(db, curatedTextTables)
}

// MigrateCuratedTexts applique le step sur `db`. Exportée pour ops, qui l'appelle avant d'écrire
// les citations : une base restée à l'ancienne forme (colonnes de texte revenues, base
// préconstruite) est portée à la nouvelle au lieu d'échouer sur une colonne `NOT NULL`.
func MigrateCuratedTexts(db *sql.DB) error { return applyCuratedTextsToTranslations(db) }
