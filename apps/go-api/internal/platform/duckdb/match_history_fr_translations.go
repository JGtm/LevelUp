// Package duckdb — enrichissement des noms (carte, mode, sélection, variante) des MatchHistoryRawRow,
// best-effort, dans toutes les langues connues.
//
// Reproduit le pattern de home_repo.go : résolution par UUID directement (map_id, pair_id,
// playlist_id) dans asset_translations (assetNameTexts), sans aller-retour par les noms EN, puis
// mode_name_tr pour les sous-modes (modeTextsByLocale). Les noms sont rassemblés pour chaque langue
// de config/locales.toml : le choix de la langue de la requête se fait en aval (Text.Pick).
//
// Cohérent avec ce qui marche déjà sur la home page et les tuiles de match.
// Best-effort : une lecture en échec est journalisée et consignée, sans bloquer le rendu.
package duckdb

import (
	"context"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// matchHistoryNames regroupe, par type d'actif, les noms de chaque langue des actifs d'une page
// d'historique, et les noms de mode de mode_name_tr.
type matchHistoryNames struct {
	maps, pairs, playlists, variants map[string]i18n.Text
	modes                            map[i18n.Locale]map[string]string
}

// applyMatchHistoryFRTranslations enrichit les rows en utilisant les UUIDs
// (map_id, pair_id, playlist_id) pour faire un lookup direct dans
// metadata.asset_translations + mode_name_tr. Best-effort.
func applyMatchHistoryFRTranslations(ctx context.Context, pdb *PlayerDB, rows []domain.MatchHistoryRawRow) {
	if pdb == nil || pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	names := matchHistoryNames{
		maps:      assetNameTexts(ctx, pdb.Metadata, assetTypeMap, collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.MapID })),
		pairs:     assetNameTexts(ctx, pdb.Metadata, assetTypePair, collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.PairID })),
		playlists: assetNameTexts(ctx, pdb.Metadata, assetTypePlaylist, collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.PlaylistID })),
		// game_variant : source de MODE pour les titres sans pair_name (Halo 5), résolu dans toutes
		// les langues (registry NULL en H5) — parité avec le fallback GameVariant de la home
		// (analysis.modeLabels).
		variants: assetNameTexts(ctx, pdb.Metadata, assetTypeGameVariant, collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.GameVariantID })),
	}
	names.modes = modeTextsByLocale(ctx, pdb.Metadata, modeKeysOfHistory(rows, names.pairs))

	for i := range rows {
		applyMatchHistoryRow(&rows[i], names)
	}
}

// modeKeysOfHistory collecte les clés de mode (noms normalisés de la langue de repli) à traduire par
// mode_name_tr : le pair_name brut de chaque row et, quand il est un UUID, le nom de l'actif de la
// paire dans chaque langue.
func modeKeysOfHistory(rows []domain.MatchHistoryRawRow, pairs map[string]i18n.Text) []string {
	keys := modeKeySet{}
	for _, r := range rows {
		keys.addLabel(r.PairName[i18n.Fallback])
		if r.PairID != nil {
			keys.addTexts(pairs[*r.PairID])
		}
	}
	return keys.list()
}

// setKeys retourne les clés d'un ensemble (ordre quelconque).
func setKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// applyMatchHistoryRow applique les noms (map, pair, playlist, game_variant) à une row :
//   - carte : le libellé canonique (libelleDeCarte, map_labels.go), celui que l'Explorateur affiche,
//     propose dans son filtre et compare à `?maps=` ;
//   - paire : la cascade unifiée mode_name_tr → asset re-lookup → raw ;
//   - sélection : la traduction de l'actif quand le nom de la langue manque ;
//   - variante de jeu : tous les noms de l'actif que le registre ne porte pas (source de MODE des
//     titres sans pair, registre NULL en Halo 5).
func applyMatchHistoryRow(row *domain.MatchHistoryRawRow, names matchHistoryNames) {
	if row.MapID != nil {
		fillUntranslated(&row.MapName, names.maps[strings.TrimSpace(*row.MapID)])
	}
	var pair i18n.Text
	if row.PairID != nil {
		pair = names.pairs[*row.PairID]
	}
	applyPairNameTexts(&row.PairName, pair, names.modes)
	if row.PlaylistID != nil {
		fillUntranslated(&row.PlaylistName, names.playlists[strings.TrimSpace(*row.PlaylistID)])
	}
	if row.GameVariantID != nil {
		fillAbsent(&row.GameVariantName, names.variants[strings.TrimSpace(*row.GameVariantID)])
	}
}

// collectDistinctIDs extrait les IDs uniques non-vides d'un slice de rows
// via un getter. Préserve l'ordre d'apparition pour stabilité des tests.
func collectDistinctIDs(rows []domain.MatchHistoryRawRow, get func(domain.MatchHistoryRawRow) *string) []string {
	seen := make(map[string]struct{}, len(rows))
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		p := get(r)
		if p == nil {
			continue
		}
		id := strings.TrimSpace(*p)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// loadPairAssetNames charge, pour les pair_id donnés, le nom de la paire dans chaque langue
// (asset_translations, type pair). Helper partagé entre match_history et filters pour le fallback de
// re-lookup mode_name_tr (cf. analysis.ResolvePairName). Best-effort : une erreur est journalisée et
// consignée (assetNameTexts : un chargement mis en cache qui l'appelle n'est pas caché) ; rend ce
// qui a été lu.
func loadPairAssetNames(ctx context.Context, meta *DB, pairIDs []string) map[string]i18n.Text {
	if meta == nil || len(pairIDs) == 0 {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return assetNameTexts(ctx2, meta, assetTypePair, pairIDs)
}
