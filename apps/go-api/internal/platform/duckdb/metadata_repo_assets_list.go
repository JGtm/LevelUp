// Package duckdb — metadata_repo_assets_list.go : listing maps & armes pour l'Asset Drawer.
package duckdb

import (
	"context"
	"fmt"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// Compile-time check : MetadataRepo implémente port.AssetMetaRepository.
var _ port.AssetMetaRepository = (*MetadataRepo)(nil)

// ListMapsByTitle retourne les maps d'un titre avec leurs traductions EN/FR.
// Source primaire : maps_catalog (toujours peuplé via populate-playlists-catalog).
// Enrichissement optionnel : asset_translations (peuplé via populate-assets).
// Si asset_translations est vide, name_canonical de maps_catalog est utilisé comme
// name_en — le drawer affiche les noms même sans populate-assets.
// search filtre par nom EN ou FR (case-insensitive, LIKE %search%). Vide = tout.
func (r *MetadataRepo) ListMapsByTitle(
	ctx context.Context,
	titleID string,
	search string,
) ([]canonical.AssetMeta, error) {
	query := `
		SELECT DISTINCT ON (m.name_canonical)
		       m.map_asset_id                                     AS asset_id,
		       COALESCE(at_en.name, m.name_canonical, '')         AS name_en,
		       COALESCE(at_fr.name, '')                           AS name_fr
		FROM maps_catalog m
		LEFT JOIN asset_translations at_en
		    ON at_en.asset_id   = m.map_asset_id
		   AND at_en.asset_type = 'map'
		   AND at_en.lang       = 'en-US'
		LEFT JOIN asset_translations at_fr
		    ON at_fr.asset_id   = m.map_asset_id
		   AND at_fr.asset_type = 'map'
		   AND at_fr.lang       = 'fr-FR'
		WHERE m.title_slug = ?
		  AND COALESCE(m.name_canonical, '') NOT LIKE '% - %'
		  AND (? = ''
		       OR lower(COALESCE(at_en.name, m.name_canonical, '')) LIKE lower('%' || ? || '%')
		       OR lower(COALESCE(at_fr.name, ''))                    LIKE lower('%' || ? || '%'))
		ORDER BY m.name_canonical, at_en.name
	`

	rows, err := r.meta.Query(ctx, query, titleID, search, search, search)
	if err != nil {
		return nil, fmt.Errorf("ListMapsByTitle: %w", err)
	}
	defer rows.Close()

	var out []canonical.AssetMeta
	for rows.Next() {
		var m canonical.AssetMeta
		if err := rows.Scan(&m.ID, &m.NameEN, &m.NameFR); err != nil {
			return nil, fmt.Errorf("ListMapsByTitle scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListWeaponsByTitle retourne les armes avec leurs traductions EN/FR.
// search filtre par nom EN ou FR (case-insensitive, LIKE %search%). Vide = tout.
// titleID est accepté pour respecter l'interface — weapon_labels n'est pas segmenté par titre en V1.
func (r *MetadataRepo) ListWeaponsByTitle(
	ctx context.Context,
	_ string,
	search string,
) ([]canonical.AssetMeta, error) {
	query := `
		SELECT weapon_id::VARCHAR           AS id,
		       name_en,
		       COALESCE(name_fr, '')        AS name_fr
		FROM weapon_labels
		WHERE ? = ''
		   OR lower(name_en)                LIKE lower('%' || ? || '%')
		   OR lower(COALESCE(name_fr, ''))  LIKE lower('%' || ? || '%')
		ORDER BY name_en
	`

	rows, err := r.meta.Query(ctx, query, search, search, search)
	if err != nil {
		return nil, fmt.Errorf("ListWeaponsByTitle: %w", err)
	}
	defer rows.Close()

	var out []canonical.AssetMeta
	for rows.Next() {
		var m canonical.AssetMeta
		if err := rows.Scan(&m.ID, &m.NameEN, &m.NameFR); err != nil {
			return nil, fmt.Errorf("ListWeaponsByTitle scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListMedalsByTitle retourne les médailles (nom seul — le sprite h5 est chargé à
// part par le boot loader de l'Asset Drawer). Sans colonnes sprite ici → sûr pour
// TOUTE metadata.duckdb (HINF n'a pas ces colonnes). search filtre par nom EN/FR.
func (r *MetadataRepo) ListMedalsByTitle(
	ctx context.Context,
	_ string,
	search string,
) ([]canonical.AssetMeta, error) {
	query := `
		SELECT medal_name_id::VARCHAR AS id,
		       name_en,
		       COALESCE(name_fr, '')  AS name_fr
		FROM medal_definitions
		WHERE ? = ''
		   OR lower(name_en)               LIKE lower('%' || ? || '%')
		   OR lower(COALESCE(name_fr, '')) LIKE lower('%' || ? || '%')
		ORDER BY name_en
	`
	rows, err := r.meta.Query(ctx, query, search, search, search)
	if err != nil {
		return nil, fmt.Errorf("ListMedalsByTitle: %w", err)
	}
	defer rows.Close()

	var out []canonical.AssetMeta
	for rows.Next() {
		var m canonical.AssetMeta
		if err := rows.Scan(&m.ID, &m.NameEN, &m.NameFR); err != nil {
			return nil, fmt.Errorf("ListMedalsByTitle scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
