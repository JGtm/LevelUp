// Package duckdb - media_repo_translations.go : enrichMediaMapTranslations +
// loadMapCatalogNames + translateMap/ModeFilterOptions +
// loadAssetTranslationNames + loadMapImageURLsByID +
// loadModeNameTranslations. Decoupe de media_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// assetIDLikeRe matche un UUID brut (asset_id Halo). Sert de garde anti-GUID :
// un match non enrichi stocke l'asset_id de map dans match_registry.map_name au
// lieu du nom résolu — il ne doit JAMAIS s'afficher tel quel. Miroir local de
// halo_infinite/adapter_asset_urls.go:uuidRe (générique, gardé local pour éviter
// un couplage title-specific au package duckdb).
var assetIDLikeRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// looksLikeAssetID indique si s a la forme d'un asset_id brut (UUID).
func looksLikeAssetID(s string) bool {
	return assetIDLikeRe.MatchString(strings.TrimSpace(s))
}

func (r *MediaRepo) enrichMediaMapTranslations(ctx context.Context, rows []domain.MediaFileRow) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	ids := collectMediaMapIDs(rows)
	if len(ids) == 0 {
		return
	}

	// Résolution via maps_catalog (name_canonical TOUJOURS peuplé) + asset_translations
	// FR, même cascade que ListMapsByTitle. Corrige le bug "map = GUID" : un match non
	// enrichi stocke l'asset_id de map dans match_registry.map_name ; on le résout par
	// map_asset_id et on n'affiche jamais l'UUID brut.
	names := r.loadMapCatalogNames(ctx, ids)

	// GH2-B6 : locale-aware. Sous UI EN, le nom canonique EN prime (name_canonical) ;
	// sous FR, la traduction FR (asset_translations). resolvePlaylistNameForLocale
	// est le helper de préférence-par-locale du package (fallback croisé si un nom
	// manque) — réutilisé ici pour ne pas dupliquer la logique.
	locale := ctxkeys.Locale(ctx)
	for i := range rows {
		id := mediaMapAssetID(&rows[i])
		if id != "" {
			if n, ok := names[id]; ok {
				if resolved := resolvePlaylistNameForLocale(locale, n.fr, n.en); resolved != "" {
					rows[i].MapName = &resolved
					continue
				}
			}
		}
		// Map absente du catalogue : ne jamais laisser un asset_id brut à l'écran.
		if rows[i].MapName != nil && looksLikeAssetID(*rows[i].MapName) {
			rows[i].MapName = nil
		}
	}
}

// mediaMapAssetID retourne l'asset_id de map à résoudre pour une row : MapID s'il
// est présent, sinon MapName quand il a la forme d'un asset_id brut (match non
// enrichi → match_registry a stocké l'UUID dans map_name).
func mediaMapAssetID(row *domain.MediaFileRow) string {
	if row.MapID != nil && strings.TrimSpace(*row.MapID) != "" {
		return strings.TrimSpace(*row.MapID)
	}
	if row.MapName != nil && looksLikeAssetID(*row.MapName) {
		return strings.TrimSpace(*row.MapName)
	}
	return ""
}

// collectMediaMapIDs extrait les map_asset_id distincts à résoudre (MapID, ou
// MapName-as-asset-id pour les matchs non enrichis).
func collectMediaMapIDs(rows []domain.MediaFileRow) []string {
	seen := make(map[string]struct{})
	for i := range rows {
		if id := mediaMapAssetID(&rows[i]); id != "" {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

// mapCatalogName regroupe le nom canonique EN (maps_catalog.name_canonical) et la
// traduction FR (asset_translations) d'un map_asset_id.
type mapCatalogName struct {
	en string
	fr string
}

// loadMapCatalogNames résout map_asset_id → (name_canonical EN, name FR) via
// maps_catalog (source TOUJOURS peuplée via populate-playlists-catalog) enrichi de
// la traduction FR asset_translations. Garantit un nom lisible même quand
// asset_translations est vide — même cascade que ListMapsByTitle. Réutilisé par
// le picker (fallback image map) et l'enrichissement de la galerie média.
func (r *MediaRepo) loadMapCatalogNames(ctx context.Context, ids []string) map[string]mapCatalogName {
	out := make(map[string]mapCatalogName)
	if r.pdb == nil || r.pdb.Metadata == nil || len(ids) == 0 {
		return out
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, resolveMediaTitleSlug(r.pdb.TitleSlug))
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := `SELECT m.map_asset_id,
       COALESCE(m.name_canonical, '') AS name_en,
       COALESCE(fr.name, '')          AS name_fr
FROM maps_catalog m
LEFT JOIN asset_translations fr
  ON fr.asset_id = m.map_asset_id
 AND fr.asset_type = 'map'
 AND fr.lang IN ('fr-FR', 'fr')
WHERE m.title_slug = ?
  AND m.map_asset_id IN (` + joinStrings(placeholders) + `)
ORDER BY m.map_asset_id, CASE WHEN fr.lang = 'fr-FR' THEN 0 ELSE 1 END`
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, en, fr string
		if err := rows.Scan(&id, &en, &fr); err != nil {
			continue
		}
		cur := out[id]
		if en != "" {
			cur.en = en
		}
		if fr != "" && cur.fr == "" { // 1re ligne = fr-FR (cf. ORDER BY)
			cur.fr = fr
		}
		out[id] = cur
	}
	return out
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
		cat := r.modeTax.Classify(p.id)
		if cat == "" {
			cat = r.modeTax.Other
		}
		if cat == "" {
			continue // titre sans taxonomie de modes → pas de regroupement par catégorie
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
	args = append(args, resolveMediaTitleSlug(r.pdb.TitleSlug))
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

// mediaStaticTitleSlug est le slug de REPLI quand le PlayerDB courant n'a pas de
// titre résolu (les tables de référence maps_catalog / map_images_registry sont
// title-scopées par colonne). Le titre est désormais dérivé du PlayerDB via
// resolveMediaTitleSlug — même pattern que career_repo_csr.go::loadRankedPlaylistsCatalog.
const mediaStaticTitleSlug = "halo_infinite"

// resolveMediaTitleSlug retourne le titre du PlayerDB courant (rend les requêtes
// de traduction maps/modes title-aware), ou le repli Halo Infinite si vide.
func resolveMediaTitleSlug(titleSlug string) string {
	if s := strings.TrimSpace(titleSlug); s != "" {
		return s
	}
	return mediaStaticTitleSlug
}

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
