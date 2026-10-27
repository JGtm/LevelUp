package migration

// definition_texts_write.go — écriture des textes de définitions dans les tables PAR LANGUE
// (asset_translations), pour les écrivains qui ne peuvent pas passer par `ops` ni par
// `platform/duckdb` : les seeds de migration (jeux Halo Infinite et Halo 5), la synchro des succès
// et le dépôt du catalogue de jalons. La convention de rangement (type d'actif, `name`,
// `description`) est celle de metadata_definition_texts_to_translations_v1 ; la lecture est
// `platform/duckdb.LoadAssetTexts`.
//
// Écriture SELECT-puis-UPDATE-ou-INSERT (duckdbkeys.Upsert), jamais `ON CONFLICT` (ADR 0019).
// Deux écritures concurrentes sur la même clé doivent être sérialisées par l'appelant (bail
// d'écrivain metadata, transaction de migration).

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/pkg/duckdbkeys"
)

// AssetTypeMilestone est le type d'actif des jalons dans asset_translations : le titre est rangé
// dans `name`, la condition lisible dans `description`.
const AssetTypeMilestone = "milestone"

// UpsertAssetTexts écrit le nom et la description d'une définition dans asset_translations, une
// ligne par langue qui porte un texte non vide (étiquette Halo principale de la langue). Une langue
// sans texte n'écrit rien et n'efface rien. Idempotente. ex est une base ou une transaction.
func UpsertAssetTexts(ctx context.Context, ex duckdbkeys.Querier, assetType, assetID string, names, descriptions i18n.Text) error {
	assetType, assetID = strings.TrimSpace(assetType), strings.TrimSpace(assetID)
	if assetType == "" || assetID == "" {
		return errors.New("asset texts: type ou identifiant vide")
	}
	for _, l := range i18n.Locales() {
		name, desc := strings.TrimSpace(names[l]), strings.TrimSpace(descriptions[l])
		if name == "" && desc == "" {
			continue
		}
		tags := i18n.HaloTags(l)
		if len(tags) == 0 {
			return fmt.Errorf("asset texts %s/%s: langue %s sans étiquette Halo", assetType, assetID, l)
		}
		var descArg any
		if desc != "" {
			descArg = desc
		}
		err := duckdbkeys.Upsert{
			ExistsSQL:  `SELECT 1 FROM asset_translations WHERE asset_id = ? AND asset_type = ? AND lang = ?`,
			ExistsArgs: []any{assetID, assetType, tags[0]},
			UpdateSQL:  `UPDATE asset_translations SET name = ?, description = ? WHERE asset_id = ? AND asset_type = ? AND lang = ?`,
			UpdateArgs: []any{name, descArg, assetID, assetType, tags[0]},
			InsertSQL:  `INSERT INTO asset_translations (asset_id, asset_type, lang, name, description) VALUES (?, ?, ?, ?, ?)`,
			InsertArgs: []any{assetID, assetType, tags[0], name, descArg},
		}.Exec(ctx, ex)
		if err != nil {
			return fmt.Errorf("asset texts %s/%s (%s): %w", assetType, assetID, l, err)
		}
	}
	return nil
}

// UpsertMilestoneCatalogEntry écrit un jalon du catalogue : la ligne de milestone_catalog (champs
// non localisés, dont la formule technique `condition`) puis son titre et sa condition lisible par
// langue dans asset_translations. À appeler sur une base dont les colonnes appariées
// `title_en` / `title_fr` / `condition_*` ont été retirées (après
// metadata_definition_texts_to_translations_v1) ; dans une transaction pour que la ligne et ses
// traductions se valident ensemble.
func UpsertMilestoneCatalogEntry(ctx context.Context, ex duckdbkeys.Querier, e milestones.CatalogEntry) error {
	if e.ID == "" {
		return errors.New("milestone catalog: identifiant vide")
	}
	now := time.Now().UTC()
	err := duckdbkeys.Upsert{
		ExistsSQL:  `SELECT 1 FROM milestone_catalog WHERE id = ?`,
		ExistsArgs: []any{e.ID},
		UpdateSQL: `UPDATE milestone_catalog SET title_slug = ?, metric = ?, threshold = ?, icon = ?,
			condition = ?, updated_at = ? WHERE id = ?`,
		UpdateArgs: []any{e.TitleSlug, e.Metric, e.Threshold, nullIfEmptyText(e.Icon), nullIfEmptyText(e.Condition), now, e.ID},
		InsertSQL: `INSERT INTO milestone_catalog (id, title_slug, metric, threshold, icon, condition, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
		InsertArgs: []any{e.ID, e.TitleSlug, e.Metric, e.Threshold, nullIfEmptyText(e.Icon), nullIfEmptyText(e.Condition), now},
	}.Exec(ctx, ex)
	if err != nil {
		return fmt.Errorf("milestone catalog %s: %w", e.ID, err)
	}
	return UpsertAssetTexts(ctx, ex, AssetTypeMilestone, e.ID, e.Title, e.ConditionText)
}

// nullIfEmptyText rend nil pour une chaîne vide (NULL en base plutôt que ”).
func nullIfEmptyText(s string) any {
	if s == "" {
		return nil
	}
	return s
}
