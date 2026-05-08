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

	// Application : priorité à mode_name_tr (sous-mode FR), fallback asset_translations.
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

		// Pair / Mode : priorité 1 = mode_name_tr (sous-mode normalisé en FR).
		modeEN := analysis.NormalizeModeLabel(derefString(rows[i].PairName))
		if fr := modeFR[modeEN]; fr != "" {
			rows[i].PairNameFR = &fr
		} else if rows[i].PairID != nil {
			id := *rows[i].PairID
			// Priorité 2 : asset_translations (nom complet de paire).
			if needsHomeAssetTranslation(derefString(rows[i].PairNameFR), derefString(rows[i].PairName)) {
				if name := strings.TrimSpace(pairNames[id]); name != "" {
					rows[i].PairNameFR = &name
				}
			}
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
func loadModeFRBatch(ctx context.Context, pdb *PlayerDB, modeENSet map[string]struct{}) map[string]string {
	if len(modeENSet) == 0 || pdb.Metadata == nil {
		return nil
	}
	enList := make([]string, 0, len(modeENSet))
	for k := range modeENSet {
		enList = append(enList, k)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(enList)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, placeholders)
	args := make([]any, len(enList))
	for i, n := range enList {
		args[i] = n
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := pdb.Metadata.Query(ctx2, q, args...)
	if err != nil {
		if !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "match_history: loadModeFRBatch failed", "err", err)
		}
		return nil
	}
	defer rows.Close()
	out := make(map[string]string, len(enList))
	for rows.Next() {
		var en, fr string
		if rows.Scan(&en, &fr) == nil && strings.TrimSpace(fr) != "" {
			out[en] = fr
		}
	}
	return out
}
