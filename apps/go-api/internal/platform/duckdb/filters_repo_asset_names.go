// Package duckdb — filters_repo_asset_names.go : résolution read-side ID->nom
// des assets (carte / sélection / mode) pour le pipeline de filtres.
//
// Contrat d'extension multi-titre (branché sur la DONNÉE, jamais sur le slug) :
// un titre fournit les noms d'assets par l'une de deux voies —
//   - à l'ingestion : les noms registry (map_name, playlist_name, pair_name)
//     sont écrits dans shared_matches_v2 (voie Halo Infinite) ;
//   - metadata-side : seuls les ids sont écrits au registry (noms NULL) et les
//     libellés bilingues vivent dans metadata.asset_translations (voie Halo 5,
//     qui n'a pas non plus de pair — game_variant tient lieu de MODE).
//
// applyAssetNamesFromMetadata absorbe cette différence : elle se déclenche sur
// « nom vide ET id présent » et résout les noms EN/FR depuis asset_translations.
// Sur Infinite les noms sont déjà remplis → la collecte est vide → ZÉRO requête
// metadata (no-op strict). Un futur titre marche automatiquement quel que soit
// son modèle, sans branchement par slug ni mapping TOML (les libellés viennent
// de la donnée bilingue par titre).
//
// Les cascades FR historiques (applyModeFRTranslations / applyMapFRTranslations /
// applyPlaylistFRTranslations, déménagées ici depuis filters_repo.go pour tenir
// le seuil 500 L) restent des raffinements idempotents appliqués APRÈS cette
// résolution.
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

// resolvedAssetNames regroupe les noms EN/FR résolus par kind depuis
// asset_translations, pour passer sous la limite de 5 paramètres.
type resolvedAssetNames struct {
	mapEN, mapFR           map[string]string
	playlistEN, playlistFR map[string]string
	variantEN, variantFR   map[string]string
}

// applyAssetNamesFromMetadata remplit les noms de carte / sélection / mode à
// partir de metadata.asset_translations, pour les titres qui n'écrivent que les
// ids au registry (noms NULL, voie Halo 5). Déclenchée par la donnée (nom vide +
// id présent), jamais par le slug : no-op strict sur les titres qui portent déjà
// les noms (collecte vide → aucune requête metadata). Best-effort : les erreurs
// de bulk sont loggées (WarnContext) puis on continue.
func (r *FiltersRepo) applyAssetNamesFromMetadata(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}
	mapIDs, playlistIDs, variantIDs := collectAssetIDsNeedingNames(rows)
	if len(mapIDs) == 0 && len(playlistIDs) == 0 && len(variantIDs) == 0 {
		return // voie Infinite : noms déjà présents → zéro requête metadata
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	metaRepo := NewMetadataRepoFromDB(r.pdb.Metadata)
	langsFR := PreferredLangsForLocale("fr")
	langsEN := PreferredLangsForLocale("en")
	res := resolvedAssetNames{
		mapEN:      r.resolveAssetNamesBulkBestEffort(ctx2, metaRepo, "map", mapIDs, langsEN),
		mapFR:      r.resolveAssetNamesBulkBestEffort(ctx2, metaRepo, "map", mapIDs, langsFR),
		playlistEN: r.resolveAssetNamesBulkBestEffort(ctx2, metaRepo, "playlist", playlistIDs, langsEN),
		playlistFR: r.resolveAssetNamesBulkBestEffort(ctx2, metaRepo, "playlist", playlistIDs, langsFR),
		variantEN:  r.resolveAssetNamesBulkBestEffort(ctx2, metaRepo, "game_variant", variantIDs, langsEN),
		variantFR:  r.resolveAssetNamesBulkBestEffort(ctx2, metaRepo, "game_variant", variantIDs, langsFR),
	}

	for i := range rows {
		applyResolvedAssetNamesRow(&rows[i], res)
	}
	logResolvedAssetNames(ctx, res, len(mapIDs), len(playlistIDs), len(variantIDs))
}

// resolveAssetNamesBulkBestEffort encapsule ResolveAssetNamesBulk avec log
// WarnContext sur erreur (ne pas avaler en silence) puis retour best-effort.
// Retourne nil pour une liste d'ids vide (aucune requête).
func (r *FiltersRepo) resolveAssetNamesBulkBestEffort(
	ctx context.Context,
	metaRepo *MetadataRepo,
	kind string,
	ids, preferredLangs []string,
) map[string]string {
	if len(ids) == 0 {
		return nil
	}
	names, err := metaRepo.ResolveAssetNamesBulk(ctx, kind, ids, preferredLangs)
	if err != nil {
		slog.WarnContext(ctx, "filters: résolution asset_translations échouée",
			"kind", kind, "err", err)
		return nil
	}
	return names
}

// collectAssetIDsNeedingNames extrait, par kind, les ids distincts À RÉSOUDRE :
// id présent ET nom cible vide. Déclencheur = la donnée (jamais le slug).
func collectAssetIDsNeedingNames(rows []domain.FilterMatchRow) (mapIDs, playlistIDs, variantIDs []string) {
	seenMap := make(map[string]struct{}, len(rows))
	seenPlaylist := make(map[string]struct{}, len(rows))
	seenVariant := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if id := assetIDNeedingName(row.MapID, row.MapName); id != "" {
			appendDistinct(&mapIDs, seenMap, id)
		}
		if id := assetIDNeedingName(row.PlaylistID, row.PlaylistNameEN); id != "" {
			appendDistinct(&playlistIDs, seenPlaylist, id)
		}
		if id := assetIDNeedingName(row.GameVariantID, row.GameVariantName); id != "" {
			appendDistinct(&variantIDs, seenVariant, id)
		}
	}
	return mapIDs, playlistIDs, variantIDs
}

// assetIDNeedingName retourne l'id nettoyé si l'id est présent ET le nom cible
// vide (cas à résoudre), sinon la chaîne vide.
func assetIDNeedingName(idPtr, namePtr *string) string {
	id := strings.TrimSpace(derefString(idPtr))
	if id == "" {
		return ""
	}
	if strings.TrimSpace(derefString(namePtr)) != "" {
		return ""
	}
	return id
}

// appendDistinct ajoute id à out s'il n'a pas déjà été vu (préserve l'ordre).
func appendDistinct(out *[]string, seen map[string]struct{}, id string) {
	if _, ok := seen[id]; ok {
		return
	}
	seen[id] = struct{}{}
	*out = append(*out, id)
}

// applyResolvedAssetNamesRow remplit les champs d'une row UNIQUEMENT s'ils sont
// vides, et jamais avec un id/UUID : si la traduction manque, le champ reste
// vide (pas de fuite d'UUID dans l'UI).
func applyResolvedAssetNamesRow(row *domain.FilterMatchRow, res resolvedAssetNames) {
	applyNamePair(row.MapID, res.mapEN, res.mapFR, &row.MapName, &row.MapNameFR)
	applyNamePair(row.PlaylistID, res.playlistEN, res.playlistFR, &row.PlaylistNameEN, &row.PlaylistName)
	applyNamePair(row.GameVariantID, res.variantEN, res.variantFR, &row.GameVariantName, &row.GameVariantNameFR)
}

// applyNamePair pose le nom EN (si champ EN vide) et le nom FR-sinon-EN (si champ
// FR vide) pour un id donné. N'écrit jamais une valeur vide.
func applyNamePair(idPtr *string, enMap, frMap map[string]string, enField, frField **string) {
	id := strings.TrimSpace(derefString(idPtr))
	if id == "" {
		return
	}
	en := strings.TrimSpace(enMap[id])
	if en != "" && derefString(*enField) == "" {
		enCopy := en
		*enField = &enCopy
	}
	if derefString(*frField) == "" {
		if fr := frThenEN(frMap[id], en); fr != "" {
			frCopy := fr
			*frField = &frCopy
		}
	}
}

// frThenEN retourne le FR nettoyé s'il est non vide, sinon l'EN (déjà nettoyé).
func frThenEN(fr, en string) string {
	if v := strings.TrimSpace(fr); v != "" {
		return v
	}
	return en
}

// logResolvedAssetNames émet les compteurs de résolution par kind (union EN/FR)
// face aux ids demandés. WarnContext si au moins un kind demandé ne résout RIEN
// (metadata incomplète → catégorie de filtres vide SANS erreur : cette
// dégradation best-effort doit laisser une trace, règle logging n°3 — c'est
// exactement le mode de panne silencieux du bug « filtres H5 vides »).
// DebugContext sinon (compteurs de suivi).
func logResolvedAssetNames(ctx context.Context, res resolvedAssetNames, mapReq, plReq, varReq int) {
	mapN := countResolved(res.mapEN, res.mapFR)
	plN := countResolved(res.playlistEN, res.playlistFR)
	varN := countResolved(res.variantEN, res.variantFR)
	attrs := []any{
		"maps_resolus", mapN, "maps_demandes", mapReq,
		"playlists_resolues", plN, "playlists_demandees", plReq,
		"game_variants_resolus", varN, "game_variants_demandes", varReq,
	}
	if (mapReq > 0 && mapN == 0) || (plReq > 0 && plN == 0) || (varReq > 0 && varN == 0) {
		slog.WarnContext(ctx, "filters: aucun nom résolu pour un kind demandé — "+
			"asset_translations incomplet, catégorie de filtres potentiellement vide", attrs...)
		return
	}
	if mapN == 0 && plN == 0 && varN == 0 {
		return
	}
	slog.DebugContext(ctx, "filters: noms d'assets résolus depuis metadata", attrs...)
}

// countResolved compte les ids distincts présents dans l'un des deux maps.
func countResolved(enMap, frMap map[string]string) int {
	if len(enMap) == 0 && len(frMap) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(enMap)+len(frMap))
	for id := range enMap {
		seen[id] = struct{}{}
	}
	for id := range frMap {
		seen[id] = struct{}{}
	}
	return len(seen)
}

// applyModeFRTranslations enrichit PairNameFR dans les rows via la cascade
// unifiée analysis.ResolvePairNameFR (mode_name_tr puis re-lookup via
// asset_translations puis raw fallback).
//
// Source unique de vérité partagée avec home_repo et match_history. Sans le
// re-lookup asset_translations, certains pair_id corrompus (asset_translations
// retournant l'EN raw "Arena:CTF on X" pour toutes les langues) feraient
// cohabiter "CTF" + "Capture du drapeau" dans available_modes (cf. thought_log
// 2026-05-09 root cause P2).
//
// Best-effort : les erreurs sont silencieuses pour ne pas bloquer la
// résolution des filtres.
func (r *FiltersRepo) applyModeFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	// Étape 1 : collecter les pair_id distincts ET les mode_en normalisés
	// (depuis pair_name brut ET depuis l'asset name si déjà connu).
	pairIDs := collectDistinctPairIDsForFilters(rows)
	pairAssetNames := loadPairAssetNamesFR(ctx, r.pdb.Metadata, pairIDs)

	uniqueEN := make(map[string]struct{}, 32)
	for _, row := range rows {
		if en := analysis.NormalizeModeLabel(derefString(row.PairName)); en != "" {
			uniqueEN[en] = struct{}{}
		}
		if row.PairID != nil {
			if assetName := strings.TrimSpace(pairAssetNames[*row.PairID]); assetName != "" {
				if en := analysis.NormalizeModeLabel(assetName); en != "" {
					uniqueEN[en] = struct{}{}
				}
			}
		}
	}
	if len(uniqueEN) == 0 {
		return
	}

	enNames := make([]string, 0, len(uniqueEN))
	for en := range uniqueEN {
		enNames = append(enNames, en)
	}
	modeFR := loadModeNamesFRForKeys(ctx, r.pdb.Metadata, enNames)
	if len(modeFR) == 0 && len(pairAssetNames) == 0 {
		return
	}

	// Étape 2 : appliquer le helper canonique sur chaque row.
	for i := range rows {
		var assetName string
		if rows[i].PairID != nil {
			assetName = pairAssetNames[*rows[i].PairID]
		}
		if fr := analysis.ResolvePairNameFR(
			derefString(rows[i].PairName),
			derefString(rows[i].PairNameFR),
			assetName,
			modeFR,
		); fr != "" {
			rows[i].PairNameFR = &fr
		}
	}
}

// collectDistinctPairIDsForFilters extrait les pair_id uniques non-vides.
func collectDistinctPairIDsForFilters(rows []domain.FilterMatchRow) []string {
	seen := make(map[string]struct{}, len(rows))
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.PairID == nil {
			continue
		}
		id := strings.TrimSpace(*r.PairID)
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

// applyMapFRTranslations enrichit MapNameFR quand map_name_fr est absent de match_registry
// (MapNameFR == MapName = COALESCE fallback EN). Interroge match_registry pour les map_id
// puis asset_translations pour les noms FR. Best-effort : erreurs silencieuses.
func (r *FiltersRepo) applyMapFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil {
		return
	}
	uniqueEN := r.collectMapENNeedingFR(rows)
	if len(uniqueEN) == 0 {
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nameToID, ok := r.resolveSharedAssetNameToID(ctx2, uniqueEN, "map_name", "map_id")
	if !ok || len(nameToID) == 0 {
		return
	}

	idToFR := r.loadAssetFRTranslations(ctx2, nameToID, "map")
	if len(idToFR) == 0 {
		return
	}

	for i := range rows {
		en := derefString(rows[i].MapName)
		if en == "" || en != derefString(rows[i].MapNameFR) {
			continue
		}
		if mapID, ok := nameToID[en]; ok {
			if fr, ok2 := idToFR[mapID]; ok2 && fr != "" {
				rows[i].MapNameFR = &fr
			}
		}
	}
}

// collectMapENNeedingFR retourne les noms EN distincts pour lesquels la traduction FR
// est manquante (MapNameFR == MapName : fallback COALESCE).
func (r *FiltersRepo) collectMapENNeedingFR(rows []domain.FilterMatchRow) map[string]struct{} {
	uniqueEN := make(map[string]struct{}, 16)
	for _, row := range rows {
		en := derefString(row.MapName)
		if en != "" && en == derefString(row.MapNameFR) {
			uniqueEN[en] = struct{}{}
		}
	}
	return uniqueEN
}

// resolveSharedAssetNameToID résout en map (name → id) depuis shared.match_registry pour
// un set de noms EN. Helper générique pour FR translations (map, playlist).
func (r *FiltersRepo) resolveSharedAssetNameToID(
	ctx context.Context,
	uniqueEN map[string]struct{},
	nameCol, idCol string,
) (map[string]string, bool) {
	names := make([]string, 0, len(uniqueEN))
	for n := range uniqueEN {
		names = append(names, n)
	}
	ph := Placeholders(len(names))
	q := fmt.Sprintf(
		`SELECT DISTINCT %s, %s FROM match_registry WHERE %s IN (%s) AND %s IS NOT NULL`,
		nameCol, idCol, nameCol, ph, idCol,
	)
	args := make([]any, len(names))
	for i, n := range names {
		args[i] = n
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, false
	}
	defer release()

	idRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, false
	}
	defer idRows.Close()

	nameToID := make(map[string]string, len(names))
	for idRows.Next() {
		var name, id string
		if idRows.Scan(&name, &id) == nil && name != "" && id != "" {
			nameToID[name] = id
		}
	}
	return nameToID, true
}

// loadAssetFRTranslations charge les traductions FR (fr-FR > fr) depuis metadata.asset_translations
// pour un set d'IDs résolus, en priorisant fr-FR > fr.
func (r *FiltersRepo) loadAssetFRTranslations(
	ctx context.Context,
	nameToID map[string]string,
	assetType string,
) map[string]string {
	ids := make([]string, 0, len(nameToID))
	for _, id := range nameToID {
		ids = append(ids, id)
	}
	ph := Placeholders(len(ids))
	q := fmt.Sprintf(
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = ? AND lang IN ('fr-FR', 'fr') AND asset_id IN (%s) ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`,
		ph,
	)
	args := make([]any, 0, len(ids)+1)
	args = append(args, assetType)
	for _, id := range ids {
		args = append(args, id)
	}

	// QueryRecovered : auto-réparation si handle metadata FATAL-invalidated (bug ART).
	trRows, err := r.pdb.Metadata.QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer trRows.Close()

	idToFR := make(map[string]string, len(ids))
	for trRows.Next() {
		var assetID, name string
		if trRows.Scan(&assetID, &name) == nil {
			if _, exists := idToFR[assetID]; !exists {
				idToFR[assetID] = name
			}
		}
	}
	return idToFR
}

// applyPlaylistFRTranslations enrichit PlaylistName quand playlist_name_fr est absent de
// match_registry (PlaylistName == PlaylistNameEN = COALESCE fallback EN).
// Best-effort : erreurs silencieuses.
func (r *FiltersRepo) applyPlaylistFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil {
		return
	}
	uniqueEN := r.collectPlaylistENNeedingFR(rows)
	if len(uniqueEN) == 0 {
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nameToID, ok := r.resolveSharedAssetNameToID(ctx2, uniqueEN, "playlist_name", "playlist_id")
	if !ok || len(nameToID) == 0 {
		return
	}

	idToFR := r.loadAssetFRTranslations(ctx2, nameToID, "playlist")
	if len(idToFR) == 0 {
		return
	}

	for i := range rows {
		en := derefString(rows[i].PlaylistNameEN)
		if en == "" || en != derefString(rows[i].PlaylistName) {
			continue
		}
		if plID, ok := nameToID[en]; ok {
			if fr, ok2 := idToFR[plID]; ok2 && fr != "" {
				rows[i].PlaylistName = &fr
			}
		}
	}
}

// collectPlaylistENNeedingFR retourne les playlist names EN distincts dont la traduction FR
// est absente de match_registry.
func (r *FiltersRepo) collectPlaylistENNeedingFR(rows []domain.FilterMatchRow) map[string]struct{} {
	uniqueEN := make(map[string]struct{}, 8)
	for _, row := range rows {
		en := derefString(row.PlaylistNameEN)
		if en != "" && en == derefString(row.PlaylistName) {
			uniqueEN[en] = struct{}{}
		}
	}
	return uniqueEN
}
