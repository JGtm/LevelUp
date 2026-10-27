// Package duckdb — home_repo_translations_canonical.go : enrichissements des noms, dans toutes les
// langues connues, des canonical.PlayerMatchRow (AssetReference Map/Playlist/GameVariant/PairMode) —
// cascade asset_translations + mode_name_tr + map_images_registry.
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// canonicalAssetTranslations regroupe les noms résolus par type d'actif (une entrée par langue),
// les URLs de carte et les noms de mode (mode_name_tr) par langue.
type canonicalAssetTranslations struct {
	maps, playlists, variants, pairs map[string]i18n.Text
	mapImageURLs                     map[string]string
	modes                            map[i18n.Locale]map[string]string
}

// EnrichCanonicalAssetTranslations complète Labels des AssetReference (Map, Playlist, GameVariant,
// PairMode) des canonical rows, pour CHAQUE langue connue, depuis metadata.asset_translations +
// mode_name_tr quand match_registry.{...}_name est absent ou n'est que le nom de la langue de repli.
// Sans cela, modes/maps/playlists restent dans la langue du registre sur la home (Faits marquants,
// KPIs, sessions, tuiles match). Les libellés de toutes les langues sont posés : l'enrichissement ne
// dépend pas de la langue de la requête (les lignes enrichies peuvent être mises en cache).
func (r *HomeRepo) EnrichCanonicalAssetTranslations(ctx context.Context, rows []canonical.PlayerMatchRow) error {
	if r == nil || r.pdb == nil || r.pdb.Metadata == nil || len(rows) == 0 {
		return nil
	}

	t := r.resolveCanonicalAssetTranslations(ctx, rows)

	for i := range rows {
		fillCanonicalAssetLabelsBatch(&rows[i], t)
		overwriteCanonicalAssetLabelsBatch(&rows[i], t)
		r.applyCanonicalPlaylistDisplay(rows[i].Summary.Playlist)
		r.applyCanonicalMapIconURL(rows[i].Summary.Map, t)
		applyCanonicalPairMode(rows[i].Summary.PairMode, t)
		normalizeGameVariantMode(rows[i].Summary.PairMode)
	}
	return nil
}

// normalizeGameVariantMode ramène les libellés d'un mode porté par le type de jeu à la forme d'un mode
// (analysis.NormalizeModeLabel : préfixe technique « Conteneur:Mode », suffixe de carte), la forme des
// valeurs d'option des filtres. Fait ICI, une fois, pour que toute page lise le même libellé ; un libellé
// que la normalisation viderait reste tel quel. Sans effet sur une partie (pair) ni sur un mode sans nom.
func normalizeGameVariantMode(mode *canonical.AssetReference) {
	if !modeIsGameVariant(mode) {
		return
	}
	if n := analysis.NormalizeModeLabel(mode.DefaultLabel); n != "" {
		mode.DefaultLabel = n
	}
	for lang, label := range mode.Labels {
		if n := analysis.NormalizeModeLabel(label); n != "" {
			mode.Labels[lang] = n
		}
	}
}

// applyCanonicalPlaylistDisplay applique le CHOKEPOINT UNIQUE de libellé de
// playlist (strip de catégorie + override playlist_labels.toml) sur le libellé de la langue par
// défaut APRÈS résolution (DB ou asset_translations) — INCONDITIONNEL, contrairement à
// fillCanonicalAssetLabel gaté par namePlaceholder. Sans ça, une playlist
// dont le nom est déjà peuplé en DB (« Super Fiesta Fête », « Arène delta :
// Héritage ») échappait au raccourcissement sur les tuiles de match et les
// sessions dominantes. Idempotent (Display d'un libellé déjà résolu = lui-même).
func (r *HomeRepo) applyCanonicalPlaylistDisplay(ref *canonical.AssetReference) {
	if ref == nil {
		return
	}
	// DefaultLabel : pour Halo 5, match_registry.playlist_name est synced dans la langue par défaut
	// (« Super Fiesta Fête ») et sert de repli quand le libellé de cette langue est absent (sessions
	// dominantes → dominantNameFromRows retombe sur la langue de repli). On le résout donc aussi. Le
	// libellé de la langue de repli reste intact — parité Match View qui ne transforme que la langue
	// par défaut.
	if ref.DefaultLabel != "" {
		ref.DefaultLabel = r.playlistDisplay.Display(ref.DefaultLabel)
	}
	if ref.Labels == nil {
		return
	}
	if label := ref.Labels[string(i18n.Default)]; label != "" {
		ref.Labels[string(i18n.Default)] = r.playlistDisplay.Display(label)
	}
}

// resolveCanonicalAssetTranslations collecte les IDs nécessaires + résout les noms de chaque
// langue + URLs map + mode_name_tr en une seule passe.
//
// match_registry.{*}_name peut avoir été synced dans la langue par défaut selon le client de sync.
// Sans libellé de la langue de repli, AssetReference.Text range ce DefaultLabel sous la langue de
// repli et le nom de la langue par défaut fuit en UI de la langue de repli : le nom de chaque
// actif est donc résolu pour TOUS les actifs, pas seulement ceux sans traduction.
func (r *HomeRepo) resolveCanonicalAssetTranslations(
	ctx context.Context, rows []canonical.PlayerMatchRow,
) canonicalAssetTranslations {
	meta := r.pdb.Metadata
	allMapIDs := collectCanonicalAssetIDs(rows, assetTypeMap)
	t := canonicalAssetTranslations{
		maps:      assetNameTexts(ctx, meta, assetTypeMap, allMapIDs),
		playlists: assetNameTexts(ctx, meta, assetTypePlaylist, collectCanonicalAssetIDs(rows, assetTypePlaylist)),
		variants:  assetNameTexts(ctx, meta, assetTypeGameVariant, collectCanonicalAssetIDs(rows, assetTypeGameVariant)),
		pairs:     assetNameTexts(ctx, meta, assetTypePair, collectCanonicalAssetIDs(rows, assetTypePair)),
	}

	mapImageURLs, mapImageURLErr := r.loadHomeMapImageURLs(ctx, allMapIDs)
	if mapImageURLErr != nil {
		slog.WarnContext(ctx, "home: loadHomeMapImageURLs failed", "err", mapImageURLErr)
		noteDegraded(ctx, "map_image_urls")
	}
	t.mapImageURLs = mapImageURLs

	t.modes = modeTextsByLocale(ctx, meta, canonicalModeKeys(rows, t.pairs))
	return t
}

// canonicalModeKeys collecte les clés de mode à traduire par mode_name_tr : le DefaultLabel de
// chaque paire et son nom d'actif dans chaque langue (re-normalisé).
func canonicalModeKeys(rows []canonical.PlayerMatchRow, pairs map[string]i18n.Text) []string {
	keys := modeKeySet{}
	for i := range rows {
		pair := rows[i].Summary.PairMode
		if pair == nil || modeIsGameVariant(pair) {
			continue
		}
		keys.addLabel(pair.DefaultLabel)
		keys.addTexts(pairs[pair.ID])
	}
	return keys.list()
}

// fillCanonicalAssetLabelsBatch applique fillCanonicalAssetLabel à chaque langue pour les 3 assets
// Map/Playlist/GameVariant. Un mode porté par le type de jeu prend le dictionnaire des types de jeu.
func fillCanonicalAssetLabelsBatch(row *canonical.PlayerMatchRow, t canonicalAssetTranslations) {
	fillCanonicalAssetLabel(row.Summary.Map, t.maps)
	fillCanonicalAssetLabel(row.Summary.Playlist, t.playlists)
	fillCanonicalAssetLabel(row.Summary.GameVariant, t.variants)
	if modeIsGameVariant(row.Summary.PairMode) {
		fillCanonicalAssetLabel(row.Summary.PairMode, t.variants)
	}
}

// overwriteCanonicalAssetLabelsBatch applique overwriteCanonicalAssetLabel (langue de repli) aux 4
// assets (le mode selon son type d'asset).
func overwriteCanonicalAssetLabelsBatch(row *canonical.PlayerMatchRow, t canonicalAssetTranslations) {
	overwriteCanonicalAssetLabel(row.Summary.Map, t.maps)
	overwriteCanonicalAssetLabel(row.Summary.Playlist, t.playlists)
	overwriteCanonicalAssetLabel(row.Summary.GameVariant, t.variants)
	if modeIsGameVariant(row.Summary.PairMode) {
		overwriteCanonicalAssetLabel(row.Summary.PairMode, t.variants)
	} else {
		overwriteCanonicalAssetLabel(row.Summary.PairMode, t.pairs)
	}
}

// applyCanonicalMapIconURL hydrate Map.IconURL via cascade :
//  1. map_images_registry (DB cache).
//  2. AssetURLAdapter par GUID (Map.ID) — titres qui indexent par UUID (Halo 5,
//     mapURLByID). NO-OP Infinite : son adapter rejette les UUID (uuidRe) → "".
//  3. AssetURLAdapter avec le nom résolu de la langue de repli (si câblé via WithAssetURL).
//
// Le repli par GUID est indispensable aux titres LIVE (Halo 5) dont
// asset_translations peut être vide : sans nom, l'étape 3 échoue et la tuile
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
	if name := strings.TrimSpace(pickAcrossChain(i18n.Fallback, t.maps[m.ID])); name != "" {
		if u := r.assetURL.MapImageURL(name); u != "" {
			m.IconURL = u
		}
	}
}

// applyCanonicalPairMode applique la cascade ResolvePairName (mode_name_tr → asset → raw) au
// libellé de chaque langue autre que la langue de repli. Sans effet sur un mode porté par le type de
// jeu : ses libellés viennent des dictionnaires des types de jeu (fillCanonicalAssetLabelsBatch), pas
// de mode_name_tr.
func applyCanonicalPairMode(pair *canonical.AssetReference, t canonicalAssetTranslations) {
	if pair == nil || modeIsGameVariant(pair) {
		return
	}
	if pair.Labels == nil {
		pair.Labels = map[string]string{}
	}
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			continue
		}
		if resolved := analysis.ResolvePairName(
			pair.DefaultLabel,
			pair.Labels[string(l)],
			pickAcrossChain(l, t.pairs[pair.ID]),
			t.modes[l],
		); resolved != "" {
			pair.Labels[string(l)] = resolved
		}
	}
}

// collectCanonicalAssetIDs retourne les asset_ids distincts non-vides pour le type donné, sans filtre
// de traduction manquante : lookups par ID stable (noms de chaque langue, registre d'images).
func collectCanonicalAssetIDs(rows []canonical.PlayerMatchRow, kind string) []string {
	ids := map[string]struct{}{}
	for i := range rows {
		for _, ref := range canonicalAssetRefs(&rows[i].Summary, kind) {
			if ref == nil {
				continue
			}
			if id := strings.TrimSpace(ref.ID); id != "" {
				ids[id] = struct{}{}
			}
		}
	}
	return setKeys(ids)
}

// canonicalAssetRefs rend les références d'un match qui se traduisent par le type d'asset
// `kind` (map|playlist|game_variant|pair). Le mode (PairMode) se traduit par le type de SA
// référence : « game_variant » quand le titre porte son mode dans le type de jeu
// (ModeAssetReference), « pair » sinon. Nil pour un kind inconnu.
func canonicalAssetRefs(s *canonical.MatchSummary, kind string) []*canonical.AssetReference {
	modeIsVariant := modeIsGameVariant(s.PairMode)
	switch kind {
	case assetTypeMap:
		return []*canonical.AssetReference{s.Map}
	case assetTypePlaylist:
		return []*canonical.AssetReference{s.Playlist}
	case assetTypeGameVariant:
		if modeIsVariant {
			return []*canonical.AssetReference{s.GameVariant, s.PairMode}
		}
		return []*canonical.AssetReference{s.GameVariant}
	case assetTypePair:
		if modeIsVariant {
			return nil
		}
		return []*canonical.AssetReference{s.PairMode}
	}
	return nil
}

// modeIsGameVariant : le mode est une copie de la référence du type de jeu (cf. ModeAssetReference).
func modeIsGameVariant(mode *canonical.AssetReference) bool {
	return mode != nil && mode.Kind == games.AssetKindGameVariant
}

// fillCanonicalAssetLabel pose, pour chaque langue autre que la langue de repli, le nom résolu de
// l'actif (sa langue, sinon le long de i18n.Chain) dans Labels quand ce libellé est absent ou
// n'est que le repli sur le nom de base (namePlaceholder).
func fillCanonicalAssetLabel(ref *canonical.AssetReference, translations map[string]i18n.Text) {
	if ref == nil || len(translations) == 0 {
		return
	}
	texts := translations[ref.ID]
	base := ref.DefaultLabel
	if v, ok := ref.Labels[string(i18n.Fallback)]; ok && v != "" {
		base = v
	}
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			continue
		}
		name := strings.TrimSpace(pickAcrossChain(l, texts))
		if name == "" || !namePlaceholder(ref.Labels[string(l)], base) {
			continue
		}
		if ref.Labels == nil {
			ref.Labels = map[string]string{}
		}
		ref.Labels[string(l)] = name
	}
}

// overwriteCanonicalAssetLabel écrase le libellé de la langue de repli avec le nom canonique
// d'asset_translations. Sans ça, la langue de repli peut fuir un DefaultLabel dans la langue par
// défaut (selon le path de sync, match_registry.{*}_name peut contenir ce label).
func overwriteCanonicalAssetLabel(ref *canonical.AssetReference, translations map[string]i18n.Text) {
	if ref == nil || len(translations) == 0 {
		return
	}
	name := strings.TrimSpace(pickAcrossChain(i18n.Fallback, translations[ref.ID]))
	if name == "" {
		return
	}
	if ref.Labels == nil {
		ref.Labels = map[string]string{}
	}
	ref.Labels[string(i18n.Fallback)] = name
}
