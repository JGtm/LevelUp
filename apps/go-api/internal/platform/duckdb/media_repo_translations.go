// Package duckdb - media_repo_translations.go : enrichMediaMapTranslations +
// loadMediaMapFRTranslations + translateMap/ModeFilterOptions +
// loadAssetTranslationNames + loadMapImageURLsByID +
// loadModeNameTranslations. Decoupe de media_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
)

func (r *MediaRepo) enrichMediaMapTranslations(ctx context.Context, rows []domain.MediaFileRow) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	ids := collectMediaMapIDs(rows)
	if len(ids) == 0 {
		return
	}

	translations := r.loadMediaMapFRTranslations(ctx, ids)
	if len(translations) == 0 {
		return
	}

	for i := range rows {
		if rows[i].MapID == nil {
			continue
		}
		if nameFR, ok := translations[*rows[i].MapID]; ok && nameFR != "" {
			rows[i].MapName = &nameFR
		}
	}
}

// collectMediaMapIDs extrait les map_id distincts non-vides depuis les rows.
func collectMediaMapIDs(rows []domain.MediaFileRow) []string {
	seen := make(map[string]struct{})
	for _, row := range rows {
		if row.MapID != nil && *row.MapID != "" {
			seen[*row.MapID] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

// loadMediaMapFRTranslations charge asset_translations (fr-FR > fr) pour les map_id donnés.
func (r *MediaRepo) loadMediaMapFRTranslations(ctx context.Context, ids []string) map[string]string {
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, "map")
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	q := `SELECT asset_id, name
FROM asset_translations
WHERE asset_type = ?
  AND lang IN ('fr-FR', 'fr')
  AND asset_id IN (` + joinStrings(placeholders) + `)
ORDER BY lang DESC`

	dbRows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer dbRows.Close()

	translations := make(map[string]string)
	for dbRows.Next() {
		var assetID, name string
		if err := dbRows.Scan(&assetID, &name); err != nil {
			continue
		}
		if _, ok := translations[assetID]; !ok {
			translations[assetID] = name
		}
	}
	return translations
}

// mediaFilterOptionPair regroupe l'id source (map_id ou pair_name brut) et le
// label SQL utilisÃ© pour le filtrage et l'affichage par dÃ©faut.
type mediaFilterOptionPair struct {
	id    string
	label string
}

// (loadMediaIDLabelPairs supprimée — extraction des pairs est désormais
// faite côté Go via extractMapPairs/extractModePairs/extractPlaylistPairs
// dans media_repo_q37_pipeline.go.)

// translateMapFilterOptions enrichit les libellÃ©s de cartes en FR via
// asset_translations + dÃ©dup par map_id. Value = map_id (stable, structurel)
// pour permettre un filtrage non ambigu cÃ´tÃ© backend (sinon "Altitude" FR ne
// matche pas "High Ground" raw EN dans match_registry, et le filtre devient
// inutilisable). Label = FR enrichi pour l'affichage.
func (r *MediaRepo) translateMapFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}
	idsSet := make(map[string]struct{})
	for _, p := range pairs {
		if p.id != "" {
			idsSet[p.id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idsSet))
	for id := range idsSet {
		ids = append(ids, id)
	}
	translations := r.loadAssetTranslationNames(ctx, "map", ids)

	// DÃ©dup par map_id : si plusieurs raw labels mappent vers le mÃªme map_id
	// (ex: "High Ground" et "Altitude" pour la mÃªme carte selon match_name_fr),
	// on regroupe sous une seule entrÃ©e. Si map_id absent, fallback sur label.
	seenIDs := make(map[string]bool)
	seenLabels := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		labelFR := translations[p.id]
		if labelFR == "" {
			labelFR = p.label
		}
		// Value = map_id si dispo (stable), sinon label (fallback mÃ©dias sans match)
		value := p.id
		if value == "" {
			value = p.label
			if seenLabels[value] {
				continue
			}
			seenLabels[value] = true
		} else {
			if seenIDs[value] {
				continue
			}
			seenIDs[value] = true
		}
		options = append(options, domain.LabelValue{Label: labelFR, Value: value})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })
	return options
}

// translateModeFilterOptions retourne une liste hiÃ©rarchique :
//   - 1 entrÃ©e racine par catÃ©gorie prÃ©sente : {Label: "Assassin", Value: "Assassin"}
//     (label EN canonique â†’ frontend traduit via i18n local)
//   - N entrÃ©es sous-mode par catÃ©gorie : {Label: "Slayer" (ou trad FR via
//     mode_name_tr si dispo), Value: "Assassin/Slayer", Parent: "Assassin"}
//
// Le format value "CatÃ©gorie/SousMode" permet au backend de filtrer finement :
// le WHERE dÃ©tecte le sÃ©parateur "/" et applique catÃ©gorie + sous-mode normalisÃ©.
func (r *MediaRepo) translateModeFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}

	// 1) Grouper par catÃ©gorie + collecter les sous-modes EN distincts
	type catBucket struct {
		category string
		subEN    map[string]struct{} // sous-modes EN canoniques (ex: "Slayer", "Team Slayer")
	}
	buckets := make(map[string]*catBucket)
	subEnSet := make(map[string]struct{})
	for _, p := range pairs {
		cat := halo_infinite.InferModeCategoryFromPairName(p.id)
		if cat == "" {
			cat = halo_infinite.ModeCategoryOther
		}
		if buckets[cat] == nil {
			buckets[cat] = &catBucket{category: cat, subEN: make(map[string]struct{})}
		}
		// Sous-mode EN canonique via NormalizeModeLabel ("Arena:Slayer on X" â†’ "Slayer").
		if sub := analysis.NormalizeModeLabel(p.id); sub != "" {
			buckets[cat].subEN[sub] = struct{}{}
			subEnSet[sub] = struct{}{}
		}
	}

	// 2) Traduire les sous-modes EN â†’ FR via mode_name_tr (best-effort)
	subEnList := make([]string, 0, len(subEnSet))
	for en := range subEnSet {
		subEnList = append(subEnList, en)
	}
	subTranslations := r.loadModeNameTranslations(ctx, subEnList)

	// 3) Construire la liste plate : header catÃ©gorie + sous-modes triÃ©s
	categories := make([]string, 0, len(buckets))
	for cat := range buckets {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	options := make([]domain.LabelValue, 0)
	for _, cat := range categories {
		b := buckets[cat]
		// Header catÃ©gorie (label EN, le frontend traduit via i18n.ts)
		options = append(options, domain.LabelValue{Label: cat, Value: cat})
		// Sous-modes triÃ©s par label localisÃ©
		subs := make([]domain.LabelValue, 0, len(b.subEN))
		for en := range b.subEN {
			label := en
			if fr, ok := subTranslations[en]; ok && fr != "" {
				label = fr
			}
			subs = append(subs, domain.LabelValue{
				Label:  label,
				Value:  cat + "/" + en, // value canonique EN pour matcher cÃ´tÃ© WHERE
				Parent: cat,
			})
		}
		sort.Slice(subs, func(i, j int) bool { return subs[i].Label < subs[j].Label })
		options = append(options, subs...)
	}
	return options
}

// loadAssetTranslationNames lit les traductions FR depuis metadata.asset_translations.
// Retourne map[asset_id]â†’nom FR. Best-effort.
func (r *MediaRepo) loadAssetTranslationNames(ctx context.Context, assetType string, assetIDs []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || r.pdb.Metadata == nil || len(assetIDs) == 0 {
		return out
	}

	placeholders := make([]string, len(assetIDs))
	args := make([]any, 0, len(assetIDs)+1)
	args = append(args, assetType)
	for i, id := range assetIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	q := `SELECT asset_id, name
FROM asset_translations
WHERE asset_type = ?
  AND lang IN ('fr-FR', 'fr')
  AND name IS NOT NULL
  AND TRIM(name) != ''
  AND asset_id IN (` + joinStrings(placeholders) + `)
ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`

	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		if _, exists := out[id]; !exists {
			out[id] = name
		}
	}
	return out
}

// loadMapImageURLsByID lit local_path depuis map_images_registry pour les
// mapIDs donnés (pattern asset kinds — lookup par ID dans la table cache,
// peuplée par cmd/migrate-static-maps). Retourne map[map_id]→local_path.
// map_ids absents du registry sont simplement absents de la map (best-effort).
func (r *MediaRepo) loadMapImageURLsByID(ctx context.Context, mapIDs []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || r.pdb.Metadata == nil || len(mapIDs) == 0 {
		return out
	}
	placeholders := make([]string, len(mapIDs))
	args := make([]any, 0, len(mapIDs)+1)
	args = append(args, mediaStaticTitleSlug)
	for i, id := range mapIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := `SELECT map_id, local_path
FROM map_images_registry
WHERE title_id = ?
  AND TRIM(local_path) != ''
  AND map_id IN (` + joinStrings(placeholders) + `)`
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, localPath string
		if err := rows.Scan(&id, &localPath); err != nil {
			continue
		}
		out[id] = localPath
	}
	return out
}

// mediaStaticTitleSlug est le slug de titre utilisé pour résoudre les URLs
// statiques côté media. Halo Infinite uniquement pour le moment ; quand un
// 2e titre arrivera, ce slug sera dérivé du contexte (cf. PathResolver).
const mediaStaticTitleSlug = "halo_infinite"

// loadModeNameTranslations lit les traductions FR depuis metadata.mode_name_tr,
// keyed par mode_en (dÃ©jÃ  normalisÃ© via analysis.NormalizeModeLabel).
func (r *MediaRepo) loadModeNameTranslations(ctx context.Context, modeENNames []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || r.pdb.Metadata == nil || len(modeENNames) == 0 {
		return out
	}

	placeholders := make([]string, len(modeENNames))
	args := make([]any, len(modeENNames))
	for i, name := range modeENNames {
		placeholders[i] = "?"
		args[i] = name
	}

	q := `SELECT mode_en, name
FROM mode_name_tr
WHERE lang = 'fr'
  AND mode_en IN (` + joinStrings(placeholders) + `)`

	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var modeEN, name string
		if err := rows.Scan(&modeEN, &name); err != nil {
			continue
		}
		if strings.TrimSpace(name) != "" {
			out[modeEN] = name
		}
	}
	return out
}
