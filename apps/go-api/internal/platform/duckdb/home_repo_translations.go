// Package duckdb — home_repo_translations.go : enrichissements FR/EN des
// tuiles match Home (HomeMatchRow legacy) — asset_translations,
// mode_name_tr, map_images_registry — + helpers de résolution partagés
// par le sous-module canonical.
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/legacymatch"
)

//nolint:gocyclo // 4 enrichments séquentiels (map/pair/playlist/variant) avec multiples Valid checks
func (r *HomeRepo) enrichHomeMatchTranslations(ctx context.Context, matches []legacymatch.HomeMatchRow) {
	if len(matches) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return
	}

	mapNames := r.resolveAssetNames(ctx, "map", collectMissingHomeAssetIDs(matches, "map"), "fr")
	pairNames := r.resolveAssetNames(ctx, "pair", collectMissingHomeAssetIDs(matches, "pair"), "fr")
	gameVariantNames := r.resolveAssetNames(ctx, "game_variant", collectMissingHomeAssetIDs(matches, "game_variant"), "fr")
	playlistNames := r.resolveAssetNames(ctx, "playlist", collectMissingHomeAssetIDs(matches, "playlist"), "fr")

	// Pattern asset kinds : lookup local_path par (title_id, map_id) dans
	// map_images_registry (peuplé par cmd/migrate-static-maps). Le name n'est
	// jamais utilisé comme clé — la stabilité vient du UUID map_id.
	mapImageURLs, _ := r.loadHomeMapImageURLs(ctx, collectAllMapIDs(matches))

	// mode_name_tr : collecte les modes EN de TOUS les matchs (sans filtre needsTranslation).
	// Quand pair_name est un UUID brut, fallback sur le nom dans pairNames (asset_translations).
	modeENSet := make(map[string]struct{})
	for _, m := range matches {
		if en := analysis.NormalizeModeLabel(m.PairName); en != "" {
			modeENSet[en] = struct{}{}
		}
		// Si pair_name est un UUID (non normalisable en mode lisible), enrichir depuis asset_translations
		if assetName := strings.TrimSpace(pairNames[m.PairID]); assetName != "" {
			if en2 := analysis.NormalizeModeLabel(assetName); en2 != "" {
				modeENSet[en2] = struct{}{}
			}
		}
	}
	modeENList := make([]string, 0, len(modeENSet))
	for k := range modeENSet {
		modeENList = append(modeENList, k)
	}
	modeNamesFR, _ := r.loadHomeModeNameTranslations(ctx, modeENList)

	for i := range matches {
		if needsHomeAssetTranslation(matches[i].MapNameFR, matches[i].MapName) {
			if name := strings.TrimSpace(mapNames[matches[i].MapID]); name != "" {
				matches[i].MapNameFR = name
			}
		}
		if matches[i].MapID != "" {
			matches[i].MapImageURL = mapImageURLs[matches[i].MapID]
		}
		// Pair / Mode : cascade unifiée via analysis.ResolvePairNameFR (mode_name_tr
		// puis re-lookup via asset_translations puis raw fallback). Source unique de
		// vérité partagée avec match_history et filters (cf. thought_log 2026-05-09).
		if fr := analysis.ResolvePairNameFR(
			matches[i].PairName,
			matches[i].PairNameFR,
			pairNames[matches[i].PairID],
			modeNamesFR,
		); fr != "" {
			matches[i].PairNameFR = fr
		}
		if needsHomeAssetTranslation(matches[i].GameVariantNameFR, matches[i].GameVariantName) {
			if name := strings.TrimSpace(gameVariantNames[matches[i].GameVariantID]); name != "" {
				matches[i].GameVariantNameFR = name
			}
		}
		if needsHomeAssetTranslation(matches[i].PlaylistNameFR, matches[i].PlaylistName) {
			if name := strings.TrimSpace(playlistNames[matches[i].PlaylistID]); name != "" {
				matches[i].PlaylistNameFR = name
			}
		}
	}
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

func collectMissingHomeAssetIDs(matches []legacymatch.HomeMatchRow, assetType string) []string {
	ids := make(map[string]struct{})
	for _, match := range matches {
		var assetID string
		var labelFR string
		var labelEN string

		switch assetType {
		case "map":
			assetID = match.MapID
			labelFR = match.MapNameFR
			labelEN = match.MapName
		case "pair":
			assetID = match.PairID
			labelFR = match.PairNameFR
			labelEN = match.PairName
		case "game_variant":
			assetID = match.GameVariantID
			labelFR = match.GameVariantNameFR
			labelEN = match.GameVariantName
		case "playlist":
			assetID = match.PlaylistID
			labelFR = match.PlaylistNameFR
			labelEN = match.PlaylistName
		default:
			return nil
		}

		if strings.TrimSpace(assetID) == "" || !needsHomeAssetTranslation(labelFR, labelEN) {
			continue
		}
		ids[assetID] = struct{}{}
	}

	result := make([]string, 0, len(ids))
	for assetID := range ids {
		result = append(result, assetID)
	}
	return result
}

// resolveAssetNames résout les noms d'asset (map / pair / playlist / game_variant)
// pour la locale donnée ("fr" ou "en"), via le résolveur unifié
// `MetadataRepo.ResolveAssetNamesBulk` (cf. metadata_repo_assets.go).
//
// Source unique pour la home tile (canonical + legacy). La cascade locale →
// l'autre langue → langue alphabétique assure qu'un asset retourne toujours
// quelque chose si une traduction existe. `isTableNotFoundErr` traité comme
// "pas de table → pas de résultat" pour les setups frais (pre-migration).
//
// Remplace les ex-`loadHomeAssetTranslationNames` (FR) et
// `loadHomeAssetTranslationNamesEN` (EN) — un seul helper paramétré au lieu
// de deux wrappers spécialisés (refactor 2026-05-08).
func (r *HomeRepo) resolveAssetNames(ctx context.Context, assetType string, assetIDs []string, locale string) map[string]string {
	if len(assetIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil
	}
	out, err := NewMetadataRepoFromDB(r.pdb.Metadata).ResolveAssetNamesBulk(
		ctx, assetType, assetIDs, PreferredLangsForLocale(locale),
	)
	if err != nil && !isTableNotFoundErr(err) {
		slog.WarnContext(ctx, "home: resolveAssetNames failed",
			"asset_type", assetType, "locale", locale, "err", err)
	}
	return out
}

// loadHomeModeNameTranslations résout les noms FR des modes depuis mode_name_tr.
// modeENNames est la liste de noms EN extraits de pair_name via NormalizeModeLabel.
// Le SQL vit dans mode_name_tr.go, source unique du littéral (garde-rail
// no_mode_name_tr_literal_test.go).
func (r *HomeRepo) loadHomeModeNameTranslations(ctx context.Context, modeENNames []string) (map[string]string, error) {
	if len(modeENNames) == 0 {
		return nil, nil
	}
	return queryModeNameTrFR(ctx, r.pdb.Metadata, modeENNames)
}

func needsHomeAssetTranslation(labelFR, labelEN string) bool {
	trimmedFR := strings.TrimSpace(labelFR)
	if trimmedFR == "" {
		return true
	}
	trimmedEN := strings.TrimSpace(labelEN)
	return trimmedEN != "" && strings.EqualFold(trimmedFR, trimmedEN)
}
