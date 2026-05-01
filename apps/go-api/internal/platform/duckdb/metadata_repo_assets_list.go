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
// search filtre par nom anglais (case-insensitive, LIKE %search%). Vide = tout.
// titleID est conservé pour la compatibilité d'interface ; asset_translations ne segmente
// pas par titre (les asset_id de map sont des UUIDs globalement uniques).
// Retourne un slice vide (sans erreur) si asset_translations ne contient pas de maps.
func (r *MetadataRepo) ListMapsByTitle(
	ctx context.Context,
	_ string, // titleID — unused : asset_translations n'a pas de colonne title_id
	search string,
) ([]canonical.AssetMeta, error) {
	// Régression corrigée : l'ancien INNER JOIN sur map_images_registry causait
	// 0 résultats quand la table était vide (images pas encore mises en cache).
	// On interroge asset_translations directement ; l'image_url est injectée par
	// AssetService.ListMaps() même sans entrée dans map_images_registry.
	query := `
		SELECT at_en.asset_id,
		       at_en.name                   AS name_en,
		       COALESCE(at_fr.name, '')     AS name_fr
		FROM asset_translations at_en
		LEFT JOIN asset_translations at_fr
		    ON at_fr.asset_id   = at_en.asset_id
		   AND at_fr.asset_type = 'map'
		   AND at_fr.lang       = 'fr-FR'
		WHERE at_en.asset_type = 'map'
		  AND at_en.lang       = 'en-US'
		  AND (? = '' OR lower(at_en.name) LIKE lower('%' || ? || '%'))
		ORDER BY at_en.name
	`

	rows, err := r.meta.Query(ctx, query, search, search)
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
// search filtre par nom anglais (case-insensitive, LIKE %search%). Vide = tout.
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
		WHERE ? = '' OR lower(name_en) LIKE lower('%' || ? || '%')
		ORDER BY name_en
	`

	rows, err := r.meta.Query(ctx, query, search, search)
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
