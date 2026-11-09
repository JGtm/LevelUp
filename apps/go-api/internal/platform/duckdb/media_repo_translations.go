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
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
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
	// (toutes les langues), même source que ListMapsByTitle. Corrige le bug "map = GUID" : un match non
	// enrichi stocke l'asset_id de map dans match_registry.map_name ; on le résout par
	// map_asset_id et on n'affiche jamais l'UUID brut.
	names := r.loadMapCatalogNames(ctx, ids)

	// GH2-B6 : locale-aware. Le nom canonique (name_canonical) tient la langue de repli ;
	// les autres langues viennent d'asset_translations. pickName est le helper de
	// préférence-par-locale du package (repli croisé si un nom manque).
	locale := i18n.FromContext(ctx)
	for i := range rows {
		id := mediaMapAssetID(&rows[i])
		if id != "" {
			if n, ok := names[id]; ok {
				if resolved := pickName(locale, n.texts, n.canonical); resolved != "" {
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

// mapCatalogName regroupe le nom canonique (maps_catalog.name_canonical, langue de repli) et les
// noms par langue (asset_translations) d'un map_asset_id.
type mapCatalogName struct {
	canonical string
	texts     i18n.Text
}

// loadMapCatalogNames résout map_asset_id → (name_canonical, noms par langue) via
// maps_catalog (source TOUJOURS peuplée via populate-playlists-catalog) enrichi des
// traductions asset_translations. Garantit un nom lisible même quand asset_translations
// est vide — même source que ListMapsByTitle. Réutilisé par le picker (fallback image map)
// et l'enrichissement de la galerie média.
func (r *MediaRepo) loadMapCatalogNames(ctx context.Context, ids []string) map[string]mapCatalogName {
	out := make(map[string]mapCatalogName)
	if r.pdb == nil || r.pdb.Metadata == nil || len(ids) == 0 {
		return out
	}
	args := make([]any, 0, len(ids)+1)
	args = append(args, resolveMediaTitleSlug(r.pdb.TitleSlug))
	for _, id := range ids {
		args = append(args, id)
	}
	q := `SELECT m.map_asset_id, COALESCE(m.name_canonical, '') AS name_canonical
FROM maps_catalog m
WHERE m.title_slug = ?
  AND m.map_asset_id IN (` + Placeholders(len(ids)) + `)
ORDER BY m.map_asset_id`
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		bestEffortFailed(ctx, "media_map_catalog", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, canonical string
		if err := rows.Scan(&id, &canonical); err != nil {
			bestEffortFailed(ctx, "media_map_catalog", err)
			continue
		}
		cur := out[id]
		if canonical != "" {
			cur.canonical = canonical
		}
		out[id] = cur
	}
	for id, texts := range assetNameTexts(ctx, r.pdb.Metadata, assetTypeMap, ids) {
		cur := out[id]
		cur.texts = texts
		out[id] = cur
	}
	return out
}

// mediaFilterOptionPair regroupe l'id source (map_id ou pair_name brut) et le
// label SQL utilisé pour le filtrage et l'affichage par défaut.
type mediaFilterOptionPair struct {
	id    string
	label string // nom de la langue par défaut : valeur de repli quand il n'y a pas d'id, clé de regroupement
	// display : libellé de repli AFFICHÉ (langue de la requête) quand l'actif n'a pas de nom traduit.
	display string
}

// (loadMediaIDLabelPairs supprimée — extraction des pairs est désormais
// faite côté Go via extractMapPairs/extractModePairs/extractPlaylistPairs
// dans media_repo_q37_pipeline.go.)

// translateMapFilterOptions enrichit les libellés de cartes dans la langue de la requête via
// asset_translations + dédup par map_id. Value = map_id (stable, structurel)
// pour permettre un filtrage non ambigu côté backend (sinon "Altitude" FR ne
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
	translations := assetNameTexts(ctx, r.pdb.Metadata, assetTypeMap, ids)
	loc := i18n.FromContext(ctx)

	// Dédup par map_id : si plusieurs raw labels mappent vers le même map_id
	// (ex: "High Ground" et "Altitude" pour la même carte selon la langue du registre),
	// on regroupe sous une seule entrée. Si map_id absent, fallback sur label.
	seenIDs := make(map[string]bool)
	seenLabels := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		label := pickAcrossChain(loc, translations[p.id])
		if label == "" {
			label = p.display
		}
		if label == "" {
			label = p.label
		}
		// Value = map_id si dispo (stable), sinon label (fallback médias sans match)
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
		options = append(options, domain.LabelValue{Label: label, Value: value})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })
	return options
}

// translateModeFilterOptions retourne une liste hiérarchique :
//   - 1 entrée racine par catégorie présente : {Label: "Assassin", Value: "Assassin"}
//     (label EN canonique → frontend traduit via i18n local)
//   - N entrées sous-mode par catégorie : {Label: "Slayer" (ou sa traduction dans la langue de
//     la requête via mode_name_tr si dispo), Value: "Assassin/Slayer", Parent: "Assassin"}
//
// Le format value "Catégorie/SousMode" permet au backend de filtrer finement :
// le WHERE détecte le séparateur "/" et applique catégorie + sous-mode normalisé.
func (r *MediaRepo) translateModeFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}

	// 1) Grouper par catégorie + collecter les sous-modes EN distincts
	type catBucket struct {
		category string
		subModes map[string]struct{} // sous-modes EN canoniques (ex: "Slayer", "Team Slayer")
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
			buckets[cat] = &catBucket{category: cat, subModes: make(map[string]struct{})}
		}
		// Sous-mode EN canonique via NormalizeModeLabel ("Arena:Slayer on X" → "Slayer").
		if sub := analysis.NormalizeModeLabel(p.id); sub != "" {
			buckets[cat].subModes[sub] = struct{}{}
			subEnSet[sub] = struct{}{}
		}
	}

	// 2) Traduire les sous-modes dans la langue de la requête via mode_name_tr (best-effort)
	subEnList := make([]string, 0, len(subEnSet))
	for en := range subEnSet {
		subEnList = append(subEnList, en)
	}
	subTranslations := r.loadModeNameTranslations(ctx, i18n.FromContext(ctx), subEnList)

	// 3) Construire la liste plate : header catégorie + sous-modes triés
	categories := make([]string, 0, len(buckets))
	for cat := range buckets {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	options := make([]domain.LabelValue, 0)
	for _, cat := range categories {
		b := buckets[cat]
		// Header catégorie (label EN, le frontend traduit via i18n.ts)
		options = append(options, domain.LabelValue{Label: cat, Value: cat})
		// Sous-modes triés par label localisé
		subs := make([]domain.LabelValue, 0, len(b.subModes))
		for en := range b.subModes {
			label := en
			if tr, ok := subTranslations[en]; ok && tr != "" {
				label = tr
			}
			subs = append(subs, domain.LabelValue{
				Label:  label,
				Value:  cat + "/" + en, // value canonique EN pour matcher côté WHERE
				Parent: cat,
			})
		}
		sort.Slice(subs, func(i, j int) bool { return subs[i].Label < subs[j].Label })
		options = append(options, subs...)
	}
	return options
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

// loadModeNameTranslations lit les traductions de la langue `loc` depuis metadata.mode_name_tr,
// keyed par mode_en (déjà normalisé via analysis.NormalizeModeLabel).
// Le SQL vit dans mode_name_tr.go, source unique du littéral (garde-rail
// no_mode_name_tr_literal_test.go). Variante best-effort : l'erreur est loguée
// (plus avalée en silence) et la map retournée vide — jamais nil, le caller
// indexe directement dedans.
func (r *MediaRepo) loadModeNameTranslations(ctx context.Context, loc i18n.Locale, modeENNames []string) map[string]string {
	out := make(map[string]string)
	if r.pdb == nil || len(modeENNames) == 0 {
		return out
	}
	for modeEN, name := range loadModeNamesForKeys(ctx, r.pdb.Metadata, loc, modeENNames) {
		out[modeEN] = name
	}
	return out
}
