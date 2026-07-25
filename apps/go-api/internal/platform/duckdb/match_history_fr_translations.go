// Package duckdb — enrichissement FR best-effort des MatchHistoryRawRow.
//
// Reproduit STRICTEMENT le pattern home_repo.go (resolveAssetNames + mode_name_tr) :
// résolution par UUID directement (map_id, pair_id, playlist_id) via
// MetadataRepo.ResolveAssetNamesBulk, sans aller-retour par les noms EN.
//
// Cohérent avec ce qui marche déjà sur la home page et les tuiles de match.
// Best-effort : erreurs silencieuses pour ne pas bloquer le rendu.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// applyMatchHistoryFRTranslations enrichit les rows en utilisant les UUIDs
// (map_id, pair_id, playlist_id) pour faire un lookup direct dans
// metadata.asset_translations + mode_name_tr. Best-effort.
func applyMatchHistoryFRTranslations(ctx context.Context, pdb *PlayerDB, rows []domain.MatchHistoryRawRow) {
	if pdb == nil || pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	// Collecter les IDs distincts à résoudre.
	mapIDs := collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.MapID })
	pairIDs := collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.PairID })
	playlistIDs := collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.PlaylistID })
	variantIDs := collectDistinctIDs(rows, func(r domain.MatchHistoryRawRow) *string { return r.GameVariantID })

	// Lookup bulk via MetadataRepo (même chemin que home_repo).
	metaRepo := NewMetadataRepoFromDB(pdb.Metadata)
	langs := PreferredLangsForLocale("fr")
	langsEN := PreferredLangsForLocale("en")

	mapNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "map", mapIDs, langs)
	pairNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "pair", pairIDs, langs)
	playlistNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "playlist", playlistIDs, langs)
	// game_variant : source de MODE pour les titres sans pair_name (Halo 5). Résolu en
	// FR (cascade fr→en) ET EN (registry NULL en H5) — parité avec le fallback
	// GameVariant de la home (analysis.modeLabels).
	variantNamesFR, _ := metaRepo.ResolveAssetNamesBulk(ctx, "game_variant", variantIDs, langs)
	variantNamesEN, _ := metaRepo.ResolveAssetNamesBulk(ctx, "game_variant", variantIDs, langsEN)

	// Lookup mode_name_tr (FR) pour les modes normalisés EN — gère les sous-modes
	// (ex: "Slayer" → "Assassin"). Cohérent avec home_repo.loadHomeModeNameTranslations.
	modeENSet := make(map[string]struct{}, 16)
	for _, r := range rows {
		if en := analysis.NormalizeModeLabel(derefString(r.PairName)); en != "" {
			modeENSet[en] = struct{}{}
		}
		// Si le pair_name brut était un UUID, l'asset_translations donne un nom EN
		// qu'on doit aussi traduire via mode_name_tr.
		if r.PairID != nil {
			if assetName := strings.TrimSpace(pairNames[*r.PairID]); assetName != "" {
				if en2 := analysis.NormalizeModeLabel(assetName); en2 != "" {
					modeENSet[en2] = struct{}{}
				}
			}
		}
	}
	modeFR := loadModeFRBatch(ctx, pdb, modeENSet)

	applyMatchHistoryFRRow(rows, mapNames, pairNames, playlistNames, variantNamesFR, variantNamesEN, modeFR)
}

// applyMatchHistoryFRRow applique les traductions FR (map, pair, playlist, game_variant)
// à chaque row. Extraction pour réduire la complexité de applyMatchHistoryFRTranslations.
func applyMatchHistoryFRRow(
	rows []domain.MatchHistoryRawRow,
	mapNames, pairNames, playlistNames, variantNamesFR, variantNamesEN map[string]string,
	modeFR map[string]string,
) {
	for i := range rows {
		applyMatchHistoryMapFR(&rows[i], mapNames)
		applyMatchHistoryPairFR(&rows[i], pairNames, modeFR)
		applyMatchHistoryPlaylistFR(&rows[i], playlistNames)
		applyMatchHistoryGameVariant(&rows[i], variantNamesFR, variantNamesEN)
	}
}

// applyMatchHistoryGameVariant résout le nom du game_variant (FR + EN) depuis
// asset_translations. Sert de source de MODE aux titres sans pair_name (Halo 5) : le
// nom registry est souvent NULL → on remplit GameVariantName (EN) et GameVariantNameFR.
// N'écrase pas une valeur registry déjà présente.
func applyMatchHistoryGameVariant(row *domain.MatchHistoryRawRow, variantNamesFR, variantNamesEN map[string]string) {
	if row.GameVariantID == nil {
		return
	}
	id := strings.TrimSpace(*row.GameVariantID)
	if id == "" {
		return
	}
	if fr := strings.TrimSpace(variantNamesFR[id]); fr != "" {
		row.GameVariantNameFR = &fr
	}
	// EN : ne remplir que si le registry n'a pas déjà un nom (NULL/vide en H5).
	if strings.TrimSpace(derefString(row.GameVariantName)) == "" {
		if en := strings.TrimSpace(variantNamesEN[id]); en != "" {
			row.GameVariantName = &en
		}
	}
}

// applyMatchHistoryMapFR enrichit MapNameFR si COALESCE SQL a renvoyé l'EN.
func applyMatchHistoryMapFR(row *domain.MatchHistoryRawRow, mapNames map[string]string) {
	if row.MapID == nil {
		return
	}
	if !needsHomeAssetTranslation(derefString(row.MapNameFR), derefString(row.MapName)) {
		return
	}
	if name := strings.TrimSpace(mapNames[*row.MapID]); name != "" {
		row.MapNameFR = &name
	}
}

// applyMatchHistoryPairFR applique la cascade unifiée mode_name_tr → asset re-lookup → raw.
func applyMatchHistoryPairFR(
	row *domain.MatchHistoryRawRow,
	pairNames map[string]string,
	modeFR map[string]string,
) {
	var assetName string
	if row.PairID != nil {
		assetName = pairNames[*row.PairID]
	}
	if fr := analysis.ResolvePairNameFR(
		derefString(row.PairName),
		derefString(row.PairNameFR),
		assetName,
		modeFR,
	); fr != "" {
		row.PairNameFR = &fr
	}
}

// applyMatchHistoryPlaylistFR enrichit PlaylistName si COALESCE SQL a renvoyé l'EN.
func applyMatchHistoryPlaylistFR(row *domain.MatchHistoryRawRow, playlistNames map[string]string) {
	if row.PlaylistID == nil {
		return
	}
	if !needsHomeAssetTranslation(derefString(row.PlaylistName), derefString(row.PlaylistNameEN)) {
		return
	}
	if name := strings.TrimSpace(playlistNames[*row.PlaylistID]); name != "" {
		row.PlaylistName = &name
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

// loadModeFRBatch charge mode_name_tr (FR) pour la liste de modes EN normalisés.
// Best-effort : retourne un map vide si erreur ou table absente.
//
// Wrapper map[string]struct{} → loadModeNamesFRForKeys, conservé pour préserver
// la signature historique du caller match_history.
func loadModeFRBatch(ctx context.Context, pdb *PlayerDB, modeENSet map[string]struct{}) map[string]string {
	if len(modeENSet) == 0 || pdb == nil || pdb.Metadata == nil {
		return nil
	}
	enList := make([]string, 0, len(modeENSet))
	for k := range modeENSet {
		enList = append(enList, k)
	}
	return loadModeNamesFRForKeys(ctx, pdb.Metadata, enList)
}

// loadModeNamesFRForKeys / loadKnownModesEN : DÉPLACÉS dans mode_name_tr.go
// (2026-07-25) — source unique du SQL sur metadata.mode_name_tr, cf. le
// garde-rail no_mode_name_tr_literal_test.go. Aucun changement de signature ni
// de comportement pour les callers de ce fichier.

// loadPairAssetNamesFR charge asset_translations[asset_type='pair', lang='fr'|'fr-FR']
// pour les pair_id donnés. Helper partagé entre match_history et filters pour
// le fallback de re-lookup mode_name_tr (cf. analysis.ResolvePairNameFR).
// Best-effort : retourne nil en cas d'erreur.
func loadPairAssetNamesFR(ctx context.Context, meta *DB, pairIDs []string) map[string]string {
	if meta == nil || len(pairIDs) == 0 {
		return nil
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(pairIDs)), ",")
	q := fmt.Sprintf(`SELECT asset_id, name FROM asset_translations
		WHERE asset_type = 'pair' AND lang IN ('fr-FR', 'fr') AND asset_id IN (%s)
		ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`, ph)
	args := make([]any, len(pairIDs))
	for i, id := range pairIDs {
		args[i] = id
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	// QueryRecovered : auto-réparation si handle metadata FATAL-invalidated (bug ART).
	rows, err := meta.QueryRecovered(ctx2, q, args...)
	if err != nil {
		if !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "fr_translations: loadPairAssetNamesFR failed", "err", err)
		}
		return nil
	}
	defer rows.Close()
	out := make(map[string]string, len(pairIDs))
	for rows.Next() {
		var id, name string
		if rows.Scan(&id, &name) == nil {
			if _, exists := out[id]; !exists {
				out[id] = strings.TrimSpace(name)
			}
		}
	}
	return out
}
