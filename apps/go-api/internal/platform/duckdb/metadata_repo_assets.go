// Package duckdb — metadata_repo_assets.go : gestion asset_translations multilingues.
//
// Sprint 54 : peuplement asset_translations depuis Discovery UGC API.
package duckdb

import (
	"context"
	"fmt"
	"strings"
)

// GetDistinctAssetIDs retourne les asset_ids distincts depuis match_registry.
// Nécessite une connexion à shared_matches_v2.duckdb (paramètre sharedDB).
//
// Colonnes par asset_type :
//   - "map" → map_id
//   - "playlist" → playlist_id
//   - "pair" → pair_id
//   - "game_variant" → game_variant_id
func (r *MetadataRepo) GetDistinctAssetIDs(
	ctx context.Context,
	assetType string,
	sharedDB *DB,
) ([]string, error) {
	columnMap := map[string]string{
		"map":          "map_id",
		"playlist":     "playlist_id",
		"pair":         "pair_id",
		"game_variant": "game_variant_id",
	}

	column, ok := columnMap[assetType]
	if !ok {
		return nil, fmt.Errorf("asset_type invalide : %s", assetType)
	}

	query := fmt.Sprintf(
		"SELECT DISTINCT %s FROM match_registry WHERE %s IS NOT NULL",
		column, column,
	)

	rows, err := sharedDB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDistinctAssetIDs(%s): %w", assetType, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan asset_id: %w", err)
		}
		if id != "" {
			ids = append(ids, id)
		}
	}

	return ids, rows.Err()
}

// GetExistingTranslations retourne un set des asset_ids déjà présents et frais.
// Fraîcheur = fetched_at >= now() - freshnessDays jours.
// Retourne map[asset_id]bool pour lookup O(1).
func (r *MetadataRepo) GetExistingTranslations(
	ctx context.Context,
	assetType string,
	lang string,
	freshnessDays int,
) (map[string]bool, error) {
	query := fmt.Sprintf(`
		SELECT asset_id
		FROM asset_translations
		WHERE asset_type = ?
		  AND lang = ?
		  AND fetched_at >= now() - INTERVAL '%d DAY'
	`, freshnessDays)

	rows, err := r.meta.Query(ctx, query, assetType, lang)
	if err != nil {
		return nil, fmt.Errorf("GetExistingTranslations(%s, %s): %w", assetType, lang, err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			return nil, fmt.Errorf("scan asset_id: %w", err)
		}
		existing[assetID] = true
	}

	return existing, rows.Err()
}

// UpsertAssetTranslation insère ou met à jour une traduction d'asset.
func (r *MetadataRepo) UpsertAssetTranslation(
	ctx context.Context,
	assetID string,
	assetType string,
	lang string,
	name string,
	description string,
) error {
	query := `
		INSERT INTO asset_translations (asset_id, asset_type, lang, name, description, fetched_at)
		VALUES (?, ?, ?, ?, ?, now())
		ON CONFLICT (asset_id, asset_type, lang) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			fetched_at = now()
	`

	_, err := r.meta.Exec(ctx, query, assetID, assetType, lang, name, description)
	if err != nil {
		return fmt.Errorf("UpsertAssetTranslation(%s, %s, %s): %w", assetType, assetID, lang, err)
	}
	return nil
}

// GetAssetTranslationCount retourne le nombre de traductions par langue pour un asset_type.
// Utile pour vérifier le peuplement après populate-assets.
func (r *MetadataRepo) GetAssetTranslationCount(
	ctx context.Context,
	assetType string,
) (map[string]int, error) {
	query := `
		SELECT lang, COUNT(*) as count
		FROM asset_translations
		WHERE asset_type = ?
		GROUP BY lang
		ORDER BY lang
	`

	rows, err := r.meta.Query(ctx, query, assetType)
	if err != nil {
		return nil, fmt.Errorf("GetAssetTranslationCount(%s): %w", assetType, err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var lang string
		var count int
		if err := rows.Scan(&lang, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[lang] = count
	}

	return counts, rows.Err()
}

// GetAssetNameIndex retourne un mapping name→asset_id pour un type d'asset donné (langue en-US).
// Utilisé par le script migrate-static-maps pour résoudre les noms de fichiers.
func (r *MetadataRepo) GetAssetNameIndex(
	ctx context.Context,
	assetType string,
) (map[string]string, error) {
	query := `
		SELECT asset_id, name
		FROM asset_translations
		WHERE asset_type = ?
		  AND lang = 'en-US'
	`

	rows, err := r.meta.Query(ctx, query, assetType)
	if err != nil {
		return nil, fmt.Errorf("query asset_translations: %w", err)
	}
	defer rows.Close()

	index := make(map[string]string)
	for rows.Next() {
		var assetID, name string
		if err := rows.Scan(&assetID, &name); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		index[name] = assetID
	}

	return index, rows.Err()
}

// ResolveAssetName retourne le meilleur nom disponible pour un (assetType, assetID),
// en respectant l'ordre de préférence linguistique fourni. Si aucune préférence
// ne matche, fallback sur n'importe quelle langue présente. Renvoie ok=false
// si aucune entrée n'existe pour cet asset.
//
// Une seule requête SQL : on charge toutes les traductions de l'asset puis on
// choisit en mémoire selon `preferredLangs`. Plus efficace qu'une cascade de
// N requêtes (1 par lang) et symétrique entre tous les callers (home tile,
// match-view, futurs consommateurs).
//
// Exemple : ResolveAssetName(ctx, "map", "2890782c-…", []string{"fr-FR","fr","en-US","en"})
// → ("Shiro", "fr-FR", true) si la table contient lang='fr-FR' name='Shiro'.
func (r *MetadataRepo) ResolveAssetName(
	ctx context.Context,
	assetType, assetID string,
	preferredLangs []string,
) (name, lang string, ok bool, err error) {
	rows, err := r.meta.Query(ctx, `
		SELECT lang, name
		FROM asset_translations
		WHERE asset_type = ? AND asset_id = ?
		  AND name IS NOT NULL AND TRIM(name) != ''
	`, assetType, assetID)
	if err != nil {
		return "", "", false, fmt.Errorf("ResolveAssetName(%s,%s): %w", assetType, assetID, err)
	}
	defer rows.Close()

	available := make(map[string]string)
	for rows.Next() {
		var l, n string
		if err := rows.Scan(&l, &n); err != nil {
			return "", "", false, err
		}
		available[l] = n
	}
	if err := rows.Err(); err != nil {
		return "", "", false, err
	}
	if len(available) == 0 {
		return "", "", false, nil
	}
	for _, pref := range preferredLangs {
		if n, present := available[pref]; present {
			return n, pref, true, nil
		}
	}
	// Aucune préférence trouvée : fallback déterministe sur la première lang
	// par ordre alphabétique pour stabilité (sinon map iter random).
	keys := make([]string, 0, len(available))
	for k := range available {
		keys = append(keys, k)
	}
	// tri minimal sans import sort: ce n'est qu'un fallback rare
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return available[keys[0]], keys[0], true, nil
}

// ResolveAssetNamesBulk fait pareil pour plusieurs asset_ids en une seule requête.
// Retourne map[asset_id]name ; les asset_ids sans aucune traduction sont absents
// du résultat. Utilisé par les pipelines bulk (home tile) qui résolvent des
// dizaines d'assets en parallèle.
func (r *MetadataRepo) ResolveAssetNamesBulk(
	ctx context.Context,
	assetType string,
	assetIDs []string,
	preferredLangs []string,
) (map[string]string, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(assetIDs)), ",")
	args := make([]any, 0, len(assetIDs)+1)
	args = append(args, assetType)
	for _, id := range assetIDs {
		args = append(args, id)
	}
	rows, err := r.meta.Query(ctx, fmt.Sprintf(`
		SELECT asset_id, lang, name
		FROM asset_translations
		WHERE asset_type = ?
		  AND asset_id IN (%s)
		  AND name IS NOT NULL AND TRIM(name) != ''
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("ResolveAssetNamesBulk(%s): %w", assetType, err)
	}
	defer rows.Close()

	// Collecte par asset_id : map[asset_id]map[lang]name
	perAsset := make(map[string]map[string]string)
	for rows.Next() {
		var assetID, lang, name string
		if err := rows.Scan(&assetID, &lang, &name); err != nil {
			return nil, err
		}
		if _, exists := perAsset[assetID]; !exists {
			perAsset[assetID] = make(map[string]string)
		}
		perAsset[assetID][lang] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(perAsset))
	for assetID, langs := range perAsset {
		picked := ""
		for _, pref := range preferredLangs {
			if n, present := langs[pref]; present {
				picked = n
				break
			}
		}
		if picked == "" {
			// fallback déterministe : première lang par ordre alphabétique
			keys := make([]string, 0, len(langs))
			for k := range langs {
				keys = append(keys, k)
			}
			for i := 1; i < len(keys); i++ {
				for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
					keys[j], keys[j-1] = keys[j-1], keys[j]
				}
			}
			if len(keys) > 0 {
				picked = langs[keys[0]]
			}
		}
		if picked != "" {
			out[assetID] = picked
		}
	}
	return out, nil
}

// PreferredLangsForLocale retourne l'ordre de préférence linguistique standard
// pour une locale UI courte (ex. "fr" → ["fr-FR","fr","en-US","en"]).
// Centralise la convention pour que tous les callers (match-view, home,
// citations…) utilisent la même cascade.
func PreferredLangsForLocale(locale string) []string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "fr", "fr-fr", "fr_fr":
		return []string{"fr-FR", "fr", "en-US", "en"}
	case "en", "en-us", "en_us":
		return []string{"en-US", "en", "fr-FR", "fr"}
	default:
		// Locale inconnue : préférence par défaut FR (le projet est FR-first),
		// puis EN, puis n'importe quoi.
		return []string{"fr-FR", "fr", "en-US", "en"}
	}
}

// UpsertMapImageRegistry insère ou met à jour une entrée dans map_images_registry.
// Utilisé par le script migrate-static-maps pour indexer les fichiers statiques.
func (r *MetadataRepo) UpsertMapImageRegistry(
	ctx context.Context,
	titleID, mapID, localPath string,
) error {
	query := `
		INSERT INTO map_images_registry (title_id, map_id, local_path, fetched_at)
		VALUES (?, ?, ?, now())
		ON CONFLICT (title_id, map_id) DO UPDATE SET
			local_path = EXCLUDED.local_path,
			fetched_at = now()
	`

	_, err := r.meta.Exec(ctx, query, titleID, mapID, localPath)
	return err
}
