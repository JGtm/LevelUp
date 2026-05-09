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

	// Lookup bulk via MetadataRepo (même chemin que home_repo).
	metaRepo := NewMetadataRepoFromDB(pdb.Metadata)
	langs := PreferredLangsForLocale("fr")

	mapNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "map", mapIDs, langs)
	pairNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "pair", pairIDs, langs)
	playlistNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "playlist", playlistIDs, langs)

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

	// Application : helper unifié analysis.ResolvePairNameFR pour les paires
	// (mode_name_tr puis re-lookup via asset_translations puis raw fallback).
	// Map / Playlist : pas de table de traduction sub-name → fallback raw direct
	// si COALESCE SQL a renvoyé l'EN.
	for i := range rows {
		// Map : si MapNameFR == MapName (= COALESCE fallback), enrichir.
		if rows[i].MapID != nil {
			id := *rows[i].MapID
			if needsHomeAssetTranslation(derefString(rows[i].MapNameFR), derefString(rows[i].MapName)) {
				if name := strings.TrimSpace(mapNames[id]); name != "" {
					rows[i].MapNameFR = &name
				}
			}
		}

		// Pair / Mode : cascade unifiée (mode_name_tr → asset re-lookup → raw).
		var assetName string
		if rows[i].PairID != nil {
			assetName = pairNames[*rows[i].PairID]
		}
		if fr := analysis.ResolvePairNameFR(
			derefString(rows[i].PairName),
			derefString(rows[i].PairNameFR),
			assetName,
			modeFR,
		); fr != "" {
			rows[i].PairNameFR = &fr
		}

		// Playlist : si PlaylistName == PlaylistNameEN (FR manquant), enrichir.
		if rows[i].PlaylistID != nil {
			id := *rows[i].PlaylistID
			if needsHomeAssetTranslation(derefString(rows[i].PlaylistName), derefString(rows[i].PlaylistNameEN)) {
				if name := strings.TrimSpace(playlistNames[id]); name != "" {
					rows[i].PlaylistName = &name
				}
			}
		}
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

// loadModeNamesFRForKeys charge mode_name_tr[lang='fr'] pour les mode_en
// normalisés donnés. Helper partagé entre match_history et filters.
// Best-effort : retourne nil/map vide en cas d'erreur (loggée pour les
// erreurs autres que table absente).
func loadModeNamesFRForKeys(ctx context.Context, meta *DB, enKeys []string) map[string]string {
	if meta == nil || len(enKeys) == 0 {
		return nil
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(enKeys)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, ph)
	args := make([]any, len(enKeys))
	for i, n := range enKeys {
		args[i] = n
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := meta.Query(ctx2, q, args...)
	if err != nil {
		if !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "fr_translations: loadModeNamesFRForKeys failed", "err", err)
		}
		return nil
	}
	defer rows.Close()
	out := make(map[string]string, len(enKeys))
	for rows.Next() {
		var en, fr string
		if rows.Scan(&en, &fr) == nil && strings.TrimSpace(fr) != "" {
			out[en] = fr
		}
	}
	return out
}

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
	rows, err := meta.Query(ctx2, q, args...)
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
