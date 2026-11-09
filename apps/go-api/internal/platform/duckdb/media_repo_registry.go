// Package duckdb — media_repo_registry.go : helper pour charger match_registry
// via SharedReader pour le pipeline Q37 média (LoadMediaFiles/Count/FilterOptions).
//
// Contexte : ADR 0016 a retiré tout ATTACH `shared` du pool. Les queries Q37
// qui mixaient `media_files` (SharedSocial) avec `shared.match_registry`
// (shared) cassaient en prod ("schema shared does not exist"). Le refactor
// P1 sépare en 2 phases : (A) media_files+associations sur SharedSocial,
// (B) match_registry via SharedReader, puis jointure + filtres + tri + dédup
// + pagination côté Go.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/i18n"
)

// mediaMatchRegistryInfo regroupe les colonnes de shared.match_registry
// nécessaires au pipeline Q37 média. Chargée via SharedReader.
type mediaMatchRegistryInfo struct {
	MatchID       string
	MapID         string
	MapName       i18n.Text
	PairName      i18n.Text
	GameVariantID string // fallback mode des titres sans pair (Halo 5) — cf. resolveMediaRegistryNameFallbacks
	// ModeNameFallback : libellé de mode résolu via asset_translations game_variant
	// quand le titre n'a PAS de pair (Halo 5). Champ DISTINCT de PairName/PairNameFR :
	// ce n'est pas un pair — il ne doit ni alimenter PairNameRaw (classification par
	// catégorie Infinite) ni être re-normalisé (déjà un libellé propre).
	ModeNameFallback i18n.Text
	PlaylistID       string
	PlaylistName     i18n.Text
	StartTime        *time.Time // TIMESTAMP naïf (UTC par convention v6+)
	StartTimeUTC     *time.Time // TIMESTAMPTZ (UTC garanti après migration)
	EndTime          *time.Time
	EndTimeUTC       *time.Time
}

// effectiveStart retourne start_time_utc si dispo, sinon start_time (réputé UTC).
// Pattern aligné sur le COALESCE des queries Q37 historiques.
func (m mediaMatchRegistryInfo) effectiveStart() *time.Time {
	if m.StartTimeUTC != nil {
		return m.StartTimeUTC
	}
	return m.StartTime
}

// loadMediaMatchRegistry charge en bulk les rows match_registry pour les
// match_ids fournis, via SharedReader. Retourne map[match_id]→info pour
// permettre l'enrichissement côté Go.
//
// Si matchIDs est vide, retourne une map vide sans toucher la DB.
// Les match_ids absents de la DB (orphelins) sont simplement absents de la
// map retournée — le caller gère ce cas comme "média sans assoc match
// reconnaissable".
func (r *MediaRepo) loadMediaMatchRegistry(
	ctx context.Context,
	matchIDs []string,
) (map[string]mediaMatchRegistryInfo, error) {
	out := make(map[string]mediaMatchRegistryInfo, len(matchIDs))
	if len(matchIDs) == 0 {
		return out, nil
	}

	// Dédupliquer les match_ids pour éviter un IN gigantesque inutile.
	seen := make(map[string]struct{}, len(matchIDs))
	dedup := make([]string, 0, len(matchIDs))
	for _, id := range matchIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dedup = append(dedup, id)
	}
	if len(dedup) == 0 {
		return out, nil
	}

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("loadMediaMatchRegistry: shared reader: %w", err)
	}
	defer release()

	placeholders := make([]string, len(dedup))
	args := make([]any, len(dedup))
	for i, id := range dedup {
		placeholders[i] = "?"
		args[i] = id
	}

	// query exécutée sur la conn shared → table accédée en root-level
	// (sans préfixe `shared.`). Cf. ADR 0016.
	q := `
		SELECT
			match_id,
			COALESCE(map_id, ''),
			COALESCE(map_name, ''),
			COALESCE(map_name_fr, ''),
			COALESCE(pair_name, ''),
			COALESCE(pair_name_fr, ''),
			COALESCE(game_variant_id, ''),
			COALESCE(playlist_id, ''),
			COALESCE(playlist_name, ''),
			COALESCE(playlist_name_fr, ''),
			start_time, start_time_utc,
			end_time, end_time_utc
		FROM match_registry
		WHERE match_id IN (` + strings.Join(placeholders, ",") + `)
	`
	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("loadMediaMatchRegistry: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var info mediaMatchRegistryInfo
		var mapEN, mapFR, pairEN, pairFR, playlistEN, playlistFR string
		var startT, startUTC, endT, endUTC sql.NullTime
		if err := rows.Scan(
			&info.MatchID,
			&info.MapID,
			&mapEN,
			&mapFR,
			&pairEN,
			&pairFR,
			&info.GameVariantID,
			&info.PlaylistID,
			&playlistEN,
			&playlistFR,
			&startT, &startUTC,
			&endT, &endUTC,
		); err != nil {
			return nil, fmt.Errorf("loadMediaMatchRegistry: scan: %w", err)
		}
		info.MapName = i18n.FromColumns(mapEN, mapFR)
		info.PairName = i18n.FromColumns(pairEN, pairFR)
		info.PlaylistName = i18n.FromColumns(playlistEN, playlistFR)
		if startT.Valid {
			t := startT.Time
			info.StartTime = &t
		}
		if startUTC.Valid {
			t := startUTC.Time
			info.StartTimeUTC = &t
		}
		if endT.Valid {
			t := endT.Time
			info.EndTime = &t
		}
		if endUTC.Valid {
			t := endUTC.Time
			info.EndTimeUTC = &t
		}
		out[info.MatchID] = info
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.resolveMediaRegistryNameFallbacks(ctx, out)
	return out, nil
}

// resolveMediaRegistryNameFallbacks remplit les noms manquants des rows registre
// depuis asset_translations (noms de toutes les langues, assetNameTexts) —
// DEC-7 (résidus H5) : les colonnes noms de match_registry sont NULL sur 100 % des
// matchs Halo 5 → la galerie média affichait « Carte inconnue » et ses filtres
// map/mode/playlist restaient vides alors que le lien média→match fonctionnait.
//   - map      : MapName ← type "map" par MapID ;
//   - mode     : ModeNameFallback ← type "game_variant" par GameVariantID quand le titre
//     n'a PAS de pair (même fallback data-driven que GetMatchMeta, lot A2) ;
//   - playlist : PlaylistName ← type "playlist" par PlaylistID.
//
// No-op quand les noms sont déjà présents (Infinite : colonnes remplies → aucun
// appel metadata). Best-effort : metadata indisponible → rows inchangées (le caller
// dégrade comme avant).
func (r *MediaRepo) resolveMediaRegistryNameFallbacks(ctx context.Context, rows map[string]mediaMatchRegistryInfo) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}
	// map : noms ← asset_translations type "map" par map_id.
	r.applyMediaNameFallback(ctx, rows, assetTypeMap, mediaNameAccessor{
		idOf:    func(i *mediaMatchRegistryInfo) string { return i.MapID },
		missing: func(i *mediaMatchRegistryInfo) bool { return i.MapName.Empty() },
		setName: func(i *mediaMatchRegistryInfo, n i18n.Text) { i.MapName = n.Clone() },
	})

	// mode : ModeNameFallback ← type "game_variant" par game_variant_id, pour les
	// titres SANS pair (Halo 5). Champ DISTINCT de PairName* (cf. mediaMatchRegistryInfo).
	r.applyMediaNameFallback(ctx, rows, assetTypeGameVariant, mediaNameAccessor{
		idOf: func(i *mediaMatchRegistryInfo) string { return i.GameVariantID },
		missing: func(i *mediaMatchRegistryInfo) bool {
			return i.PairName.Empty() && i.ModeNameFallback.Empty()
		},
		setName: func(i *mediaMatchRegistryInfo, n i18n.Text) { i.ModeNameFallback = n.Clone() },
	})

	// playlist : noms ← type "playlist" par playlist_id.
	r.applyMediaNameFallback(ctx, rows, assetTypePlaylist, mediaNameAccessor{
		idOf:    func(i *mediaMatchRegistryInfo) string { return i.PlaylistID },
		missing: func(i *mediaMatchRegistryInfo) bool { return i.PlaylistName.Empty() },
		setName: func(i *mediaMatchRegistryInfo, n i18n.Text) { i.PlaylistName = n.Clone() },
	})
}

// mediaNameAccessor décrit l'accès aux champs d'UN type d'asset pour le fallback de
// nom : quel ID sert de clé, le nom est-il manquant, où écrire le nom résolu.
type mediaNameAccessor struct {
	idOf    func(*mediaMatchRegistryInfo) string
	missing func(*mediaMatchRegistryInfo) bool
	setName func(*mediaMatchRegistryInfo, i18n.Text)
}

// assetTypeGameVariant / assetTypePlaylist / assetTypeMap : types d'asset de la
// table asset_translations consommés par le fallback média.
const (
	assetTypeMap         = "map"
	assetTypeGameVariant = "game_variant"
	assetTypePlaylist    = "playlist"
	assetTypePair        = "pair"
)

// applyMediaNameFallback résout UN type d'asset pour les rows au nom manquant puis
// y écrit les libellés résolus. Appelé 3× (map / mode / playlist). Fast-path : si
// aucune row n'a de nom manquant pour ce type → return sans appel metadata (Infinite :
// colonnes remplies). Bulk-résolution via assetNameTexts (noms de toutes les langues).
// Best-effort : une erreur metadata laisse les rows inchangées (journalisée, consignée).
func (r *MediaRepo) applyMediaNameFallback(
	ctx context.Context,
	rows map[string]mediaMatchRegistryInfo,
	assetType string,
	acc mediaNameAccessor,
) {
	var ids []string
	seen := make(map[string]struct{})
	for _, info := range rows {
		if id := acc.idOf(&info); id != "" && acc.missing(&info) {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	names := assetNameTexts(ctx, r.pdb.Metadata, assetType, ids)
	for key, info := range rows {
		if !acc.missing(&info) {
			continue
		}
		if n := names[acc.idOf(&info)]; !n.Empty() {
			acc.setName(&info, n)
			rows[key] = info
		}
	}
}
