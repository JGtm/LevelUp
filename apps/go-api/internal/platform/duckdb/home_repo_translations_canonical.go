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
// canonicalAssetTranslations regroupe les résolutions FR/EN par kind + URLs map.
type canonicalAssetTranslations struct {
	mapNames        map[string]string
	playlistNames   map[string]string
	variantNames    map[string]string
	pairNames       map[string]string
	mapNamesEN      map[string]string
	playlistNamesEN map[string]string
	variantNamesEN  map[string]string
	pairNamesEN     map[string]string
	mapImageURLs    map[string]string
	modeNamesFR     map[string]string
}

func (r *HomeRepo) EnrichCanonicalAssetTranslations(ctx context.Context, rows []canonical.PlayerMatchRow) error {
	if r == nil || r.pdb == nil || r.pdb.Metadata == nil || len(rows) == 0 {
		return nil
	}

	t := r.resolveCanonicalAssetTranslations(ctx, rows)

	for i := range rows {
		applyCanonicalAssetFRBatch(&rows[i], t)
		applyCanonicalAssetENBatch(&rows[i], t)
		r.applyCanonicalPlaylistDisplay(rows[i].Summary.Playlist)
		r.applyCanonicalMapIconURL(rows[i].Summary.Map, t)
		applyCanonicalPairModeFR(rows[i].Summary.PairMode, t)
	}
	return nil
}

// applyCanonicalPlaylistDisplay applique le CHOKEPOINT UNIQUE de libellé de
// playlist (strip de catégorie + override playlist_labels.toml) sur Labels["fr"]
// APRÈS résolution (DB ou asset_translations) — INCONDITIONNEL, contrairement à
// applyCanonicalAssetFR gaté par needsHomeAssetTranslation. Sans ça, une playlist
// dont le nom FR est déjà peuplé en DB (« Super Fiesta Fête », « Arène delta :
// Héritage ») échappait au raccourcissement sur les tuiles de match et les
// sessions dominantes. Idempotent (Display d'un libellé déjà résolu = lui-même).
func (r *HomeRepo) applyCanonicalPlaylistDisplay(ref *canonical.AssetReference) {
	if ref == nil {
		return
	}
	// DefaultLabel : pour Halo 5, match_registry.playlist_name est synced en
	// FR-localisé (« Super Fiesta Fête ») et sert de repli quand Labels["fr"] est
	// absent (sessions dominantes → dominantNameFromRows retombe sur `en`). On le
	// résout donc aussi. Labels["en"] (vrai EN) reste intact — parité Match View
	// qui ne transforme que le FR.
	if ref.DefaultLabel != "" {
		ref.DefaultLabel = r.playlistDisplay.Display(ref.DefaultLabel)
	}
	if ref.Labels == nil {
		return
	}
	if fr := ref.Labels["fr"]; fr != "" {
		ref.Labels["fr"] = r.playlistDisplay.Display(fr)
	}
}

// resolveCanonicalAssetTranslations collecte les IDs nécessaires + résout les
// traductions FR/EN + URLs map + mode_name_tr en une seule passe.
func (r *HomeRepo) resolveCanonicalAssetTranslations(
	ctx context.Context, rows []canonical.PlayerMatchRow,
) canonicalAssetTranslations {
	t := canonicalAssetTranslations{
		mapNames:      r.resolveAssetNames(ctx, "map", collectCanonicalAssetIDsNeedingFR(rows, "map"), "fr"),
		playlistNames: r.resolveAssetNames(ctx, "playlist", collectCanonicalAssetIDsNeedingFR(rows, "playlist"), "fr"),
		variantNames:  r.resolveAssetNames(ctx, "game_variant", collectCanonicalAssetIDsNeedingFR(rows, "game_variant"), "fr"),
		pairNames:     r.resolveAssetNames(ctx, "pair", collectCanonicalAssetIDsNeedingFR(rows, "pair"), "fr"),
	}

	// match_registry.{*}_name peut avoir été synced en FR-localisé selon le
	// client de sync. Sans Labels["en"], labelForLocale("en", ...) leak du FR.
	allMapIDs := collectAllCanonicalMapIDs(rows)
	allPlaylistIDs := collectAllCanonicalAssetIDs(rows, "playlist")
	allVariantIDs := collectAllCanonicalAssetIDs(rows, "game_variant")
	allPairIDs := collectAllCanonicalAssetIDs(rows, "pair")

	t.mapNamesEN = r.resolveAssetNames(ctx, "map", allMapIDs, "en")
	t.playlistNamesEN = r.resolveAssetNames(ctx, "playlist", allPlaylistIDs, "en")
	t.variantNamesEN = r.resolveAssetNames(ctx, "game_variant", allVariantIDs, "en")
	t.pairNamesEN = r.resolveAssetNames(ctx, "pair", allPairIDs, "en")

	mapImageURLs, mapImageURLErr := r.loadHomeMapImageURLs(ctx, allMapIDs)
	if mapImageURLErr != nil {
		slog.WarnContext(ctx, "home: loadHomeMapImageURLs failed", "err", mapImageURLErr)
	}
	t.mapImageURLs = mapImageURLs

	t.modeNamesFR = r.loadCanonicalModeNamesFR(ctx, rows, t.pairNames)
	return t
}

// loadCanonicalModeNamesFR construit le set des mode_en (DefaultLabel + asset FR
// re-normalisé) puis charge les traductions FR depuis mode_name_tr.
func (r *HomeRepo) loadCanonicalModeNamesFR(
	ctx context.Context, rows []canonical.PlayerMatchRow, pairNames map[string]string,
) map[string]string {
	modeENSet := map[string]struct{}{}
	for i := range rows {
		pair := rows[i].Summary.PairMode
		if pair == nil {
			continue
		}
		if en := analysis.NormalizeModeLabel(pair.DefaultLabel); en != "" {
			modeENSet[en] = struct{}{}
		}
		if name := strings.TrimSpace(pairNames[pair.ID]); name != "" {
			if en := analysis.NormalizeModeLabel(name); en != "" {
				modeENSet[en] = struct{}{}
			}
		}
	}
	modeENList := make([]string, 0, len(modeENSet))
	for k := range modeENSet {
		modeENList = append(modeENList, k)
	}
	modeNamesFR, _ := r.loadHomeModeNameTranslations(ctx, modeENList)
	return modeNamesFR
}

// applyCanonicalAssetFRBatch applique applyCanonicalAssetFR aux 3 assets Map/Playlist/GameVariant.
func applyCanonicalAssetFRBatch(row *canonical.PlayerMatchRow, t canonicalAssetTranslations) {
	applyCanonicalAssetFR(row.Summary.Map, t.mapNames)
	applyCanonicalAssetFR(row.Summary.Playlist, t.playlistNames)
	applyCanonicalAssetFR(row.Summary.GameVariant, t.variantNames)
}

// applyCanonicalAssetENBatch applique applyCanonicalAssetEN aux 4 assets.
func applyCanonicalAssetENBatch(row *canonical.PlayerMatchRow, t canonicalAssetTranslations) {
	applyCanonicalAssetEN(row.Summary.Map, t.mapNamesEN)
	applyCanonicalAssetEN(row.Summary.Playlist, t.playlistNamesEN)
	applyCanonicalAssetEN(row.Summary.GameVariant, t.variantNamesEN)
	applyCanonicalAssetEN(row.Summary.PairMode, t.pairNamesEN)
}

// applyCanonicalMapIconURL hydrate Map.IconURL via cascade :
//  1. map_images_registry (DB cache).
//  2. AssetURLAdapter par GUID (Map.ID) — titres qui indexent par UUID (Halo 5,
//     mapURLByID). NO-OP Infinite : son adapter rejette les UUID (uuidRe) → "".
//  3. AssetURLAdapter avec nom EN résolu (si câblé via WithAssetURL).
//
// Le repli par GUID est indispensable aux titres LIVE (Halo 5) dont
// asset_translations peut être vide : sans nom EN, l'étape 3 échoue et la tuile
// reste sans image, alors que l'adapter sait résoudre l'URL depuis le GUID seul.
func (r *HomeRepo) applyCanonicalMapIconURL(m *canonical.AssetReference, t canonicalAssetTranslations) {
	if m == nil || m.ID == "" {
		return
	}
	if u, ok := t.mapImageURLs[m.ID]; ok && u != "" {
		m.IconURL = u
		return
	}
	if r.assetURL == nil {
		return
	}
	// 2. Résolution par GUID (avant le nom). N'écrase pas une URL déjà résolue.
	if u := r.assetURL.MapImageURL(strings.TrimSpace(m.ID)); u != "" {
		m.IconURL = u
		return
	}
	if enName := strings.TrimSpace(t.mapNamesEN[m.ID]); enName != "" {
		if u := r.assetURL.MapImageURL(enName); u != "" {
			m.IconURL = u
		}
	}
}

// applyCanonicalPairModeFR applique la cascade ResolvePairNameFR (mode_name_tr → asset → raw).
func applyCanonicalPairModeFR(pair *canonical.AssetReference, t canonicalAssetTranslations) {
	if pair == nil {
		return
	}
	if pair.Labels == nil {
		pair.Labels = map[string]string{}
	}
	if fr := analysis.ResolvePairNameFR(
		pair.DefaultLabel,
		pair.Labels["fr"],
		t.pairNames[pair.ID],
		t.modeNamesFR,
	); fr != "" {
		pair.Labels["fr"] = fr
	}
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
