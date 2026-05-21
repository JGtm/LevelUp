// Package duckdb — home_repo_translations_canonical.go : enrichissements
// FR/EN des canonical.PlayerMatchRow (AssetReference Map/Playlist/
// GameVariant/PairMode) — cascade asset_translations + mode_name_tr +
// map_images_registry.
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
)

// EnrichCanonicalAssetTranslations remplit Labels["fr"] sur les
// AssetReference (Map, Playlist, GameVariant, PairMode) des canonical rows
// depuis metadata.asset_translations + mode_name_tr quand match_registry
// .{...}_name_fr est NULL en DB. Bug #2/#7 cascade : sans ça, modes/maps/
// playlists restent en EN sur la home (Faits marquants, KPIs, sessions,
// tuiles match).
func (r *HomeRepo) EnrichCanonicalAssetTranslations(ctx context.Context, rows []canonical.PlayerMatchRow) error {
	if r == nil || r.pdb == nil || r.pdb.Metadata == nil || len(rows) == 0 {
		return nil
	}

	mapIDs := collectCanonicalAssetIDsNeedingFR(rows, "map")
	playlistIDs := collectCanonicalAssetIDsNeedingFR(rows, "playlist")
	variantIDs := collectCanonicalAssetIDsNeedingFR(rows, "game_variant")
	pairIDs := collectCanonicalAssetIDsNeedingFR(rows, "pair")

	mapNames := r.resolveAssetNames(ctx, "map", mapIDs, "fr")
	playlistNames := r.resolveAssetNames(ctx, "playlist", playlistIDs, "fr")
	variantNames := r.resolveAssetNames(ctx, "game_variant", variantIDs, "fr")
	pairNames := r.resolveAssetNames(ctx, "pair", pairIDs, "fr")

	// Hydrater aussi Labels["en"] pour TOUS les map_ids/playlist_ids/etc. :
	// match_registry.{*}_name peut avoir été synced en FR-localisé selon le
	// client de sync. Sans Labels["en"], labelForLocale("en", ...) leak du FR.
	allMapIDs := collectAllCanonicalMapIDs(rows)
	allPlaylistIDs := collectAllCanonicalAssetIDs(rows, "playlist")
	allVariantIDs := collectAllCanonicalAssetIDs(rows, "game_variant")
	allPairIDs := collectAllCanonicalAssetIDs(rows, "pair")

	mapNamesEN := r.resolveAssetNames(ctx, "map", allMapIDs, "en")
	playlistNamesEN := r.resolveAssetNames(ctx, "playlist", allPlaylistIDs, "en")
	variantNamesEN := r.resolveAssetNames(ctx, "game_variant", allVariantIDs, "en")
	pairNamesEN := r.resolveAssetNames(ctx, "pair", allPairIDs, "en")

	// Pattern asset kinds : lookup map_image local_path par map_id dans
	// map_images_registry. La résolution d'URL est indépendante des labels.
	mapImageURLs, mapImageURLErr := r.loadHomeMapImageURLs(ctx, allMapIDs)
	if mapImageURLErr != nil {
		slog.WarnContext(ctx, "home: loadHomeMapImageURLs failed", "err", mapImageURLErr)
	}

	modeENSet := map[string]struct{}{}
	for i := range rows {
		if pair := rows[i].Summary.PairMode; pair != nil {
			if en := analysis.NormalizeModeLabel(pair.DefaultLabel); en != "" {
				modeENSet[en] = struct{}{}
			}
			if name := strings.TrimSpace(pairNames[pair.ID]); name != "" {
				if en := analysis.NormalizeModeLabel(name); en != "" {
					modeENSet[en] = struct{}{}
				}
			}
		}
	}
	modeENList := make([]string, 0, len(modeENSet))
	for k := range modeENSet {
		modeENList = append(modeENList, k)
	}
	modeNamesFR, _ := r.loadHomeModeNameTranslations(ctx, modeENList)

	for i := range rows {
		applyCanonicalAssetFR(rows[i].Summary.Map, mapNames)
		applyCanonicalAssetFR(rows[i].Summary.Playlist, playlistNames)
		applyCanonicalAssetFR(rows[i].Summary.GameVariant, variantNames)

		// Hydrate Labels["en"] depuis asset_translations[en-US] pour assurer
		// que locale=en n'utilise pas le DefaultLabel (parfois FR-localisé en
		// DB selon le path de sync).
		applyCanonicalAssetEN(rows[i].Summary.Map, mapNamesEN)
		applyCanonicalAssetEN(rows[i].Summary.Playlist, playlistNamesEN)
		applyCanonicalAssetEN(rows[i].Summary.GameVariant, variantNamesEN)
		applyCanonicalAssetEN(rows[i].Summary.PairMode, pairNamesEN)

		// Hydrate Map.IconURL — cascade :
		//   1. map_images_registry (DB cache, pattern asset kinds, lookup par
		//      map_id stable). Peuplé par cmd/migrate-static-maps.
		//   2. AssetURLAdapter avec **nom EN résolu** depuis asset_translations
		//      en-US (uniquement si l'adapter est câblé via WithAssetURL).
		//      L'adapter scanne `static/maps/halo_infinite/` au boot et indexe
		//      les fichiers par nom EN. Évite la dépendance à la CLI quand de
		//      nouveaux fichiers static sont ajoutés.
		if m := rows[i].Summary.Map; m != nil && m.ID != "" {
			if u, ok := mapImageURLs[m.ID]; ok && u != "" {
				m.IconURL = u
			} else if r.assetURL != nil {
				if enName := strings.TrimSpace(mapNamesEN[m.ID]); enName != "" {
					if u := r.assetURL.MapImageURL(enName); u != "" {
						m.IconURL = u
					}
				}
			}
		}

		if pair := rows[i].Summary.PairMode; pair != nil {
			if pair.Labels == nil {
				pair.Labels = map[string]string{}
			}
			if fr := analysis.ResolvePairNameFR(
				pair.DefaultLabel,
				pair.Labels["fr"],
				pairNames[pair.ID],
				modeNamesFR,
			); fr != "" {
				pair.Labels["fr"] = fr
			}
		}
	}
	return nil
}

// collectAllCanonicalAssetIDs retourne les asset_ids distincts non-vides pour
// le kind donné. Sans filtre needsTranslation : utilisé pour les lookups par
// ID stable (registry, traductions canoniques en-US).
func collectAllCanonicalAssetIDs(rows []canonical.PlayerMatchRow, kind string) []string {
	ids := map[string]struct{}{}
	for i := range rows {
		var ref *canonical.AssetReference
		switch kind {
		case "map":
			ref = rows[i].Summary.Map
		case "playlist":
			ref = rows[i].Summary.Playlist
		case "game_variant":
			ref = rows[i].Summary.GameVariant
		case "pair":
			ref = rows[i].Summary.PairMode
		default:
			return nil
		}
		if ref == nil {
			continue
		}
		if id := strings.TrimSpace(ref.ID); id != "" {
			ids[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

// collectAllCanonicalMapIDs retourne les map_ids distincts non-vides présents
// dans les rows (toutes maps, sans filtre needsTranslation). Utilisé pour le
// lookup map_images_registry qui dépend uniquement du map_id stable.
func collectAllCanonicalMapIDs(rows []canonical.PlayerMatchRow) []string {
	ids := map[string]struct{}{}
	for i := range rows {
		m := rows[i].Summary.Map
		if m == nil {
			continue
		}
		if id := strings.TrimSpace(m.ID); id != "" {
			ids[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

func collectCanonicalAssetIDsNeedingFR(rows []canonical.PlayerMatchRow, kind string) []string {
	ids := map[string]struct{}{}
	for i := range rows {
		var ref *canonical.AssetReference
		switch kind {
		case "map":
			ref = rows[i].Summary.Map
		case "playlist":
			ref = rows[i].Summary.Playlist
		case "game_variant":
			ref = rows[i].Summary.GameVariant
		case "pair":
			ref = rows[i].Summary.PairMode
		}
		if ref == nil || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		fr := ""
		en := ref.DefaultLabel
		if ref.Labels != nil {
			fr = ref.Labels["fr"]
			if v, ok := ref.Labels["en"]; ok && v != "" {
				en = v
			}
		}
		if needsHomeAssetTranslation(fr, en) {
			ids[ref.ID] = struct{}{}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out
}

func applyCanonicalAssetFR(ref *canonical.AssetReference, translations map[string]string) {
	if ref == nil || len(translations) == 0 {
		return
	}
	name := strings.TrimSpace(translations[ref.ID])
	if name == "" {
		return
	}
	if ref.Labels == nil {
		ref.Labels = map[string]string{}
	}
	en := ref.DefaultLabel
	if v, ok := ref.Labels["en"]; ok && v != "" {
		en = v
	}
	if needsHomeAssetTranslation(ref.Labels["fr"], en) {
		ref.Labels["fr"] = name
	}
}

// applyCanonicalAssetEN écrase Labels["en"] avec le nom canonique en-US depuis
// asset_translations. Sans ça, locale=en peut leak un DefaultLabel FR-localisé
// (selon le path de sync, match_registry.{*}_name peut contenir le label FR).
func applyCanonicalAssetEN(ref *canonical.AssetReference, translations map[string]string) {
	if ref == nil || len(translations) == 0 {
		return
	}
	name := strings.TrimSpace(translations[ref.ID])
	if name == "" {
		return
	}
	if ref.Labels == nil {
		ref.Labels = map[string]string{}
	}
	ref.Labels["en"] = name
}
