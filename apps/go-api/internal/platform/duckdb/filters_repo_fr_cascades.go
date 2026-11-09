// Package duckdb — filters_repo_fr_cascades.go : cascades de traduction des noms de carte et de
// sélection du pipeline de filtres (applyMapFRTranslations, applyPlaylistFRTranslations) et leurs
// deux lectures (match_registry puis asset_translations). Les noms sont complétés dans TOUTES les
// langues connues : une langue de plus est un jeu de lignes de asset_translations, sans changer ce
// code.
//
// Extrait de filters_repo_asset_names.go sans changement de comportement (lot perf L9-go,
// 2026-09-23) : les échecs de ces lectures, jusque-là avalés en silence, sont désormais
// journalisés et consignés (bestEffortFailed) — les lignes d'une résolution dégradée ne
// sont jamais mises en cache (player_read_cache.go) — et le fichier d'origine aurait
// dépassé 500 lignes.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// nameCascade décrit la complétion des noms d'UN type d'actif des lignes de filtres : le type
// d'asset_translations, les colonnes de nom et d'id de match_registry qui relient un nom à son id, et
// le champ de nom à compléter.
type nameCascade struct {
	assetType string
	nameCol   string
	idCol     string
	field     func(*domain.FilterMatchRow) *i18n.Text
}

var (
	mapCascade = nameCascade{
		assetType: assetTypeMap, nameCol: "map_name", idCol: "map_id",
		field: func(r *domain.FilterMatchRow) *i18n.Text { return &r.MapName },
	}
	playlistCascade = nameCascade{
		assetType: assetTypePlaylist, nameCol: "playlist_name", idCol: "playlist_id",
		field: func(r *domain.FilterMatchRow) *i18n.Text { return &r.PlaylistName },
	}
)

// applyMapFRTranslations complète MapName, dans chaque langue autre que la langue de repli, quand
// le nom de cette langue est absent de match_registry (nom absent ou identique au nom de repli =
// COALESCE SQL). Interroge match_registry pour les map_id puis asset_translations pour les noms.
// Best-effort : échec via bestEffortFailed.
func (r *FiltersRepo) applyMapFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	r.applyNameCascade(ctx, rows, mapCascade)
}

// applyPlaylistFRTranslations complète PlaylistName comme applyMapFRTranslations.
func (r *FiltersRepo) applyPlaylistFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	r.applyNameCascade(ctx, rows, playlistCascade)
}

func (r *FiltersRepo) applyNameCascade(ctx context.Context, rows []domain.FilterMatchRow, c nameCascade) {
	if r.pdb.Metadata == nil {
		return
	}
	uniqueEN := collectUntranslatedNames(rows, c.field)
	if len(uniqueEN) == 0 {
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nameToID, ok := r.resolveSharedAssetNameToID(ctx2, uniqueEN, c.nameCol, c.idCol)
	if !ok || len(nameToID) == 0 {
		return
	}
	ids := make([]string, 0, len(nameToID))
	for _, id := range nameToID {
		ids = append(ids, id)
	}
	idToTexts := assetNameTexts(ctx2, r.pdb.Metadata, c.assetType, ids)
	if len(idToTexts) == 0 {
		return
	}

	for i := range rows {
		field := c.field(&rows[i])
		en := (*field)[i18n.Fallback]
		if en == "" {
			continue
		}
		if id, ok := nameToID[en]; ok {
			fillUntranslated(field, idToTexts[id])
		}
	}
}

// collectUntranslatedNames retourne les noms de la langue de repli distincts pour lesquels une
// traduction manque, dans au moins une langue (nom absent ou identique : fallback COALESCE).
func collectUntranslatedNames(rows []domain.FilterMatchRow, field func(*domain.FilterMatchRow) *i18n.Text) map[string]struct{} {
	unique := make(map[string]struct{}, 16)
	for i := range rows {
		t := *field(&rows[i])
		en := t[i18n.Fallback]
		if en == "" {
			continue
		}
		for _, l := range i18n.Locales() {
			if l != i18n.Fallback && namePlaceholder(t[l], en) {
				unique[en] = struct{}{}
				break
			}
		}
	}
	return unique
}

// resolveSharedAssetNameToID résout en map (name → id) depuis shared.match_registry pour
// un set de noms EN (traductions des cartes et des sélections). Échec : bestEffortFailed, ok=false.
func (r *FiltersRepo) resolveSharedAssetNameToID(
	ctx context.Context,
	uniqueEN map[string]struct{},
	nameCol, idCol string,
) (map[string]string, bool) {
	names := make([]string, 0, len(uniqueEN))
	for n := range uniqueEN {
		names = append(names, n)
	}
	ph := Placeholders(len(names))
	q := fmt.Sprintf(
		`SELECT DISTINCT %s, %s FROM match_registry WHERE %s IN (%s) AND %s IS NOT NULL`,
		nameCol, idCol, nameCol, ph, idCol,
	)
	args := make([]any, len(names))
	for i, n := range names {
		args[i] = n
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		bestEffortFailed(ctx, "asset_name_ids", err)
		return nil, false
	}
	defer release()

	idRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		bestEffortFailed(ctx, "asset_name_ids", err)
		return nil, false
	}
	defer idRows.Close()

	nameToID := make(map[string]string, len(names))
	for idRows.Next() {
		var name, id string
		if idRows.Scan(&name, &id) == nil && name != "" && id != "" {
			nameToID[name] = id
		}
	}
	if err := idRows.Err(); err != nil {
		bestEffortFailed(ctx, "asset_name_ids", err)
		return nil, false
	}
	return nameToID, true
}
