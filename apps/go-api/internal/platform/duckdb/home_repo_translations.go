// Package duckdb — home_repo_translations.go : enrichissements des noms, dans toutes les langues
// connues, des tuiles match Home (HomeMatchRow legacy) — asset_translations, mode_name_tr,
// map_images_registry — + helpers de résolution partagés par le sous-module canonical.
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/legacymatch"
)

// homeAssetNames regroupe, par type d'actif, le nom de chaque langue des actifs des tuiles, et les
// noms de mode de mode_name_tr par langue.
type homeAssetNames struct {
	maps, pairs, variants, playlists map[string]i18n.Text
	modes                            map[i18n.Locale]map[string]string
}

// enrichHomeMatchTranslations complète les noms (carte, mode, variante, sélection) des tuiles dans
// TOUTES les langues connues : les noms viennent de asset_translations (assetNameTexts) et de
// mode_name_tr (modeTextsByLocale), un nom déjà traduit n'est jamais écrasé (fillUntranslated) et la
// langue de repli garde le nom du registre. Le choix de la langue de la requête se fait en aval
// (Text.Pick).
func (r *HomeRepo) enrichHomeMatchTranslations(ctx context.Context, matches []legacymatch.HomeMatchRow) {
	if len(matches) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}
	meta := r.pdb.Metadata
	names := homeAssetNames{
		maps:      assetNameTexts(ctx, meta, assetTypeMap, collectHomeAssetIDs(matches, func(m *legacymatch.HomeMatchRow) string { return m.MapID })),
		pairs:     assetNameTexts(ctx, meta, assetTypePair, collectHomeAssetIDs(matches, func(m *legacymatch.HomeMatchRow) string { return m.PairID })),
		variants:  assetNameTexts(ctx, meta, assetTypeGameVariant, collectHomeAssetIDs(matches, func(m *legacymatch.HomeMatchRow) string { return m.GameVariantID })),
		playlists: assetNameTexts(ctx, meta, assetTypePlaylist, collectHomeAssetIDs(matches, func(m *legacymatch.HomeMatchRow) string { return m.PlaylistID })),
	}
	names.modes = modeTextsByLocale(ctx, meta, homeModeKeys(matches, names.pairs))

	// Pattern asset kinds : lookup local_path par (title_id, map_id) dans
	// map_images_registry (peuplé par cmd/migrate-static-maps). Le name n'est
	// jamais utilisé comme clé — la stabilité vient du UUID map_id.
	mapImageURLs, err := r.loadHomeMapImageURLs(ctx, collectAllMapIDs(matches))
	if err != nil {
		slog.WarnContext(ctx, "home: loadHomeMapImageURLs failed", "err", err)
		noteDegraded(ctx, "map_image_urls")
	}

	for i := range matches {
		m := &matches[i]
		fillUntranslated(&m.MapName, names.maps[m.MapID])
		if m.MapID != "" {
			m.MapImageURL = mapImageURLs[m.MapID]
		}
		// Pair / Mode : cascade unifiée via analysis.ResolvePairName (mode_name_tr
		// puis re-lookup via asset_translations puis raw fallback). Source unique de
		// vérité partagée avec match_history et filters (cf. thought_log 2026-05-09).
		applyPairNameTexts(&m.PairName, names.pairs[m.PairID], names.modes)
		fillUntranslated(&m.GameVariantName, names.variants[m.GameVariantID])
		fillUntranslated(&m.PlaylistName, names.playlists[m.PlaylistID])
	}
}

// homeModeKeys collecte les clés de mode (noms normalisés) à traduire par mode_name_tr : le
// pair_name brut de chaque tuile et le nom de la paire dans chaque langue (un pair_name peut être
// un UUID).
func homeModeKeys(matches []legacymatch.HomeMatchRow, pairs map[string]i18n.Text) []string {
	keys := modeKeySet{}
	for _, m := range matches {
		keys.addLabel(m.PairName[i18n.Fallback])
		keys.addTexts(pairs[m.PairID])
	}
	return keys.list()
}

// collectAllMapIDs retourne la liste des map_id distincts présents dans les
// matchs. Contrairement à collectMissingHomeAssetIDs qui filtre sur l'absence
// de traduction, ici on veut TOUTES les maps : la résolution d'URL par map_id
// (asset kinds pattern) est indépendante des labels FR/EN.
func collectAllMapIDs(matches []legacymatch.HomeMatchRow) []string {
	ids := make(map[string]struct{})
	for _, m := range matches {
		if id := strings.TrimSpace(m.MapID); id != "" {
			ids[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

// loadHomeMapImageURLs résout local_path depuis map_images_registry pour les
// map_ids donnés. Pattern asset kinds (cf. internal/assets/) : lookup par ID
// dans la table cache, pas par nom. Le registry est peuplé par
// cmd/migrate-static-maps qui scanne static/maps/{titleSlug}/.
//
// Retourne map[map_id]local_path. Les map_ids absents du registry sont
// simplement absents du résultat — pas d'erreur, pas de fallback name-based.
// Le caller émet alors nil et le frontend dégrade gracieusement.
func (r *HomeRepo) loadHomeMapImageURLs(ctx context.Context, mapIDs []string) (map[string]string, error) {
	if len(mapIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(mapIDs)), ",")
	query := fmt.Sprintf(`
		SELECT map_id, local_path
		FROM map_images_registry
		WHERE title_id = ?
		  AND TRIM(local_path) != ''
		  AND map_id IN (%s)
	`, placeholders)
	args := make([]any, 0, len(mapIDs)+1)
	args = append(args, r.titleSlug())
	for _, id := range mapIDs {
		args = append(args, id)
	}
	// QueryRecovered : auto-réparation si handle metadata FATAL-invalidated (bug ART).
	rows, err := r.pdb.Metadata.QueryRecovered(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, len(mapIDs))
	for rows.Next() {
		var mapID, localPath string
		if err := rows.Scan(&mapID, &localPath); err != nil {
			return nil, err
		}
		out[mapID] = localPath
	}
	return out, rows.Err()
}

// collectHomeAssetIDs retourne les identifiants distincts non vides que `idOf` extrait des tuiles.
func collectHomeAssetIDs(matches []legacymatch.HomeMatchRow, idOf func(*legacymatch.HomeMatchRow) string) []string {
	ids := make(map[string]struct{})
	for i := range matches {
		if id := idOf(&matches[i]); strings.TrimSpace(id) != "" {
			ids[id] = struct{}{}
		}
	}
	return setKeys(ids)
}
