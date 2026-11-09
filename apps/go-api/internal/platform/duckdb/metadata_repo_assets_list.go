// Package duckdb — metadata_repo_assets_list.go : listing maps & armes pour l'Asset Drawer.
package duckdb

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// Compile-time check : MetadataRepo implémente port.AssetMetaRepository.
var _ port.AssetMetaRepository = (*MetadataRepo)(nil)

// ListMapsByTitle retourne les maps d'un titre avec leurs noms dans chaque langue qui en porte un.
// Source primaire : maps_catalog (toujours peuplé via populate-playlists-catalog), dont le nom canonique
// tient la langue de repli quand asset_translations n'en porte pas. Enrichissement : asset_translations
// (peuplé via populate-assets), lu par LoadAssetTexts pour toutes les langues de config/locales.toml.
// Si asset_translations est vide, name_canonical de maps_catalog sert de nom dans la langue de repli —
// le drawer affiche les noms même sans populate-assets.
// search filtre sur le nom dans N'IMPORTE QUELLE langue (case-insensitive, sous-chaîne). Vide = tout.
//
// LA DÉDUPLICATION SE FAIT PAR `map_asset_id`, PAS PAR NOM (corrigé le 2026-09-13, D15).
// Le `DISTINCT ON (m.name_canonical)` d'origine ne gardait qu'UNE ligne par nom canonique :
// deux cartes homonymes mais d'asset_id distincts (une variante Forge republiée, une carte
// et son portage classé) s'effondraient en une seule, et c'est l'asset_id — la clé sur
// laquelle le web identifie une carte — qui était choisi arbitrairement par l'ordre de tri.
// Le grain de sortie voulu est UNE LIGNE PAR ASSET : c'est donc sur `map_asset_id` que le
// DISTINCT ON doit porter. Sur les schémas courants il n'y déduplique rien — `maps_catalog`
// a pour PK (title_slug, map_asset_id) — et c'est bien ainsi : il exprime le grain au lieu de
// le laisser dépendre de contraintes qu'une base legacy peut ne pas porter.
//
// Le tri d'affichage reste celui d'avant : nom canonique, puis nom dans la langue de repli ; les
// ex aequo se rangent par identifiant (l'ancienne requête ne les ordonnait pas).
func (r *MetadataRepo) ListMapsByTitle(
	ctx context.Context,
	titleID string,
	search string,
) ([]canonical.AssetMeta, error) {
	rows, err := r.meta.Query(ctx, `
		SELECT DISTINCT ON (m.map_asset_id) m.map_asset_id, COALESCE(m.name_canonical, '') AS name_canonical
		FROM maps_catalog m
		WHERE m.title_slug = ?
		  AND COALESCE(m.name_canonical, '') NOT LIKE '% - %'
		ORDER BY m.map_asset_id`, titleID)
	if err != nil {
		return nil, fmt.Errorf("ListMapsByTitle: %w", err)
	}
	defer rows.Close()

	var ids []string
	canonicalByID := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("ListMapsByTitle scan: %w", err)
		}
		ids = append(ids, id)
		canonicalByID[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	texts, err := LoadAssetNames(ctx, r.meta, assetTypeMap, ids)
	if err != nil {
		return nil, fmt.Errorf("ListMapsByTitle: %w", err)
	}
	out := make([]canonical.AssetMeta, 0, len(ids))
	for _, id := range ids {
		names := texts[id].Name.Clone()
		if names == nil {
			names = i18n.Text{}
		}
		if !names.Has(i18n.Fallback) && canonicalByID[id] != "" {
			names[i18n.Fallback] = canonicalByID[id]
		}
		if !mapNameMatches(names, search) {
			continue
		}
		out = append(out, canonical.AssetMeta{ID: id, Names: names})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ci, cj := canonicalByID[out[i].ID], canonicalByID[out[j].ID]
		if ci != cj {
			return ci < cj
		}
		if ni, nj := out[i].Names[i18n.Fallback], out[j].Names[i18n.Fallback]; ni != nj {
			return ni < nj
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// mapNameMatches indique si `search` (vide = tout) est contenu, sans tenir compte de la casse, dans
// le nom de la carte dans l'une quelconque de ses langues.
func mapNameMatches(names i18n.Text, search string) bool {
	if search == "" {
		return true
	}
	needle := strings.ToLower(search)
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), needle) {
			return true
		}
	}
	return false
}

// ListWeaponsByTitle retourne les armes avec leurs noms dans chaque langue qui en porte un
// (weapon_label_names : une ligne par arme et par langue). search filtre sur le nom dans
// N'IMPORTE QUELLE langue (case-insensitive, LIKE %search%). Vide = tout. Tri : nom dans la langue
// de repli, puis identifiant ; le choix de la langue servie est fait par AssetService.
// titleID est accepté pour respecter l'interface — weapon_labels n'est pas segmenté par titre en V1.
func (r *MetadataRepo) ListWeaponsByTitle(
	ctx context.Context,
	_ string,
	search string,
) ([]canonical.AssetMeta, error) {
	query := `
		SELECT n.weapon_id::VARCHAR, n.lang, n.name
		FROM weapon_label_names n
		WHERE ? = ''
		   OR n.weapon_id IN (SELECT m.weapon_id FROM weapon_label_names m
		                      WHERE lower(m.name) LIKE lower('%' || ? || '%'))
		ORDER BY n.weapon_id, n.lang
	`
	rows, err := r.meta.Query(ctx, query, search, search)
	if err != nil {
		return nil, fmt.Errorf("ListWeaponsByTitle: %w", err)
	}
	defer rows.Close()

	byID := map[string]*canonical.AssetMeta{}
	var order []string
	for rows.Next() {
		var id, lang, name string
		if err := rows.Scan(&id, &lang, &name); err != nil {
			return nil, fmt.Errorf("ListWeaponsByTitle scan: %w", err)
		}
		loc, ok := i18n.Lookup(lang)
		if !ok || name == "" {
			continue
		}
		m := byID[id]
		if m == nil {
			m = &canonical.AssetMeta{ID: id, Names: i18n.Text{}}
			byID[id] = m
			order = append(order, id)
		}
		m.Names[loc] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]canonical.AssetMeta, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Names[i18n.Fallback] < out[j].Names[i18n.Fallback]
	})
	return out, nil
}

// ListMedalsByTitle retourne les médailles (nom + description dans chaque langue qui en porte
// une — le sprite h5 est chargé à part par le boot loader de l'Asset Drawer). Sans colonnes
// sprite ici → sûr pour TOUTE metadata.duckdb (HINF n'a pas ces colonnes). search filtre par
// nom, dans n'importe quelle langue de medal_translations (insensible à la casse).
//
// Les textes viennent de medal_translations (une ligne par langue, LoadMedalTexts) : le tab
// Assets est locale-neutre côté repo, AssetService choisit la langue de la requête (repli `en`).
// Tri par nom anglais ; les médailles sans nom anglais suivent, par identifiant.
func (r *MetadataRepo) ListMedalsByTitle(
	ctx context.Context,
	_ string,
	search string,
) ([]canonical.AssetMeta, error) {
	rows, err := r.meta.Query(ctx, `
		SELECT md.medal_name_id
		FROM medal_definitions md
		WHERE ? = ''
		   OR EXISTS (SELECT 1 FROM medal_translations mt
		              WHERE mt.medal_name_id = md.medal_name_id
		                AND lower(mt.name) LIKE lower('%' || ? || '%'))
		ORDER BY md.medal_name_id
	`, search, search)
	if err != nil {
		return nil, fmt.Errorf("ListMedalsByTitle: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("ListMedalsByTitle scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListMedalsByTitle rows: %w", err)
	}
	texts, err := LoadMedalTexts(ctx, r.meta, ids)
	if err != nil {
		return nil, fmt.Errorf("ListMedalsByTitle: textes: %w", err)
	}
	out := make([]canonical.AssetMeta, 0, len(ids))
	for _, id := range ids {
		t := texts[id]
		out = append(out, canonical.AssetMeta{ID: strconv.FormatInt(id, 10), Names: t.Name, Descriptions: t.Description})
	}
	sortMedalsByEnglishName(out)
	return out, nil
}

// sortMedalsByEnglishName trie par nom anglais (ordre des octets) ; les médailles sans nom anglais
// passent après, dans l'ordre reçu (identifiant croissant).
func sortMedalsByEnglishName(medals []canonical.AssetMeta) {
	sort.SliceStable(medals, func(i, j int) bool {
		a, b := medals[i].Names[i18n.Fallback], medals[j].Names[i18n.Fallback]
		if a == "" || b == "" {
			return a != "" && b == ""
		}
		return a < b
	})
}
