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
	"log/slog"
	"strings"
	"time"
)

// mediaMatchRegistryInfo regroupe les colonnes de shared.match_registry
// nécessaires au pipeline Q37 média. Chargée via SharedReader.
type mediaMatchRegistryInfo struct {
	MatchID       string
	MapID         string
	MapName       string
	MapNameFR     string
	PairName      string
	PairNameFR    string
	GameVariantID string // fallback mode des titres sans pair (Halo 5) — cf. resolveMediaRegistryNameFallbacks
	// ModeNameFallback : libellé de mode résolu via asset_translations game_variant
	// quand le titre n'a PAS de pair (Halo 5). Champ DISTINCT de PairName/PairNameFR :
	// ce n'est pas un pair — il ne doit ni alimenter PairNameRaw (classification par
	// catégorie Infinite) ni être re-normalisé (déjà un libellé propre).
	ModeNameFallback string
	PlaylistID       string
	PlaylistName     string
	PlaylistNameFR   string
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
		var startT, startUTC, endT, endUTC sql.NullTime
		if err := rows.Scan(
			&info.MatchID,
			&info.MapID,
			&info.MapName,
			&info.MapNameFR,
			&info.PairName,
			&info.PairNameFR,
			&info.GameVariantID,
			&info.PlaylistID,
			&info.PlaylistName,
			&info.PlaylistNameFR,
			&startT, &startUTC,
			&endT, &endUTC,
		); err != nil {
			return nil, fmt.Errorf("loadMediaMatchRegistry: scan: %w", err)
		}
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
// via la cascade asset_translations (ResolveAssetNamesBulk, fr-FR→fr→en-US→en) —
// DEC-7 (résidus H5) : les colonnes noms de match_registry sont NULL sur 100 % des
// matchs Halo 5 → la galerie média affichait « Carte inconnue » et ses filtres
// map/mode/playlist restaient vides alors que le lien média→match fonctionnait.
//   - map      : MapNameFR ← type "map" par MapID ;
//   - mode     : PairNameFR ← type "game_variant" par GameVariantID quand le titre
//     n'a PAS de pair (même fallback data-driven que GetMatchMeta, lot A2) ;
//   - playlist : PlaylistNameFR ← type "playlist" par PlaylistID.
//
// No-op quand les noms sont déjà présents (Infinite : colonnes remplies → aucun
// appel metadata). Best-effort : metadata indisponible → rows inchangées (le caller
// dégrade comme avant).
func (r *MediaRepo) resolveMediaRegistryNameFallbacks(ctx context.Context, rows map[string]mediaMatchRegistryInfo) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}
	slots := mediaNameFallbackSlots()
	if collectMediaFallbackIDs(slots, rows) == 0 {
		return // noms déjà présents (Infinite) → aucune requête metadata.
	}
	r.resolveMediaFallbackNames(ctx, slots)
	applyMediaFallbackNames(slots, rows)
}

// assetTypeGameVariant / assetTypePlaylist / assetTypeMap : types d'asset de la
// table asset_translations consommés par le fallback média.
const (
	assetTypeMap         = "map"
	assetTypeGameVariant = "game_variant"
	assetTypePlaylist    = "playlist"
)

// mediaNameFallbackSlot décrit la résolution de fallback d'UN type d'asset :
// quel ID sert de clé, le nom est-il manquant, où écrire le nom résolu.
type mediaNameFallbackSlot struct {
	assetType string
	id        func(*mediaMatchRegistryInfo) string
	missing   func(*mediaMatchRegistryInfo) bool
	set       func(*mediaMatchRegistryInfo, string)
	ids       []string
	names     map[string]string
}

func mediaNameFallbackSlots() []mediaNameFallbackSlot {
	return []mediaNameFallbackSlot{
		{assetType: assetTypeMap,
			id:      func(i *mediaMatchRegistryInfo) string { return i.MapID },
			missing: func(i *mediaMatchRegistryInfo) bool { return i.MapName == "" && i.MapNameFR == "" },
			set:     func(i *mediaMatchRegistryInfo, n string) { i.MapNameFR = n }},
		{assetType: assetTypeGameVariant,
			id: func(i *mediaMatchRegistryInfo) string { return i.GameVariantID },
			missing: func(i *mediaMatchRegistryInfo) bool {
				return i.PairName == "" && i.PairNameFR == "" && i.ModeNameFallback == ""
			},
			set: func(i *mediaMatchRegistryInfo, n string) { i.ModeNameFallback = n }},
		{assetType: assetTypePlaylist,
			id:      func(i *mediaMatchRegistryInfo) string { return i.PlaylistID },
			missing: func(i *mediaMatchRegistryInfo) bool { return i.PlaylistName == "" && i.PlaylistNameFR == "" },
			set:     func(i *mediaMatchRegistryInfo, n string) { i.PlaylistNameFR = n }},
	}
}

// collectMediaFallbackIDs remplit slots[].ids avec les IDs des rows au nom
// manquant. Retourne le total collecté (0 = rien à résoudre).
func collectMediaFallbackIDs(slots []mediaNameFallbackSlot, rows map[string]mediaMatchRegistryInfo) int {
	total := 0
	for s := range slots {
		for _, info := range rows {
			if id := slots[s].id(&info); id != "" && slots[s].missing(&info) {
				slots[s].ids = append(slots[s].ids, id)
			}
		}
		total += len(slots[s].ids)
	}
	return total
}

// resolveMediaFallbackNames résout chaque slot via ResolveAssetNamesBulk
// (cascade fr-FR→fr→en-US→en). Best-effort par slot.
func (r *MediaRepo) resolveMediaFallbackNames(ctx context.Context, slots []mediaNameFallbackSlot) {
	meta := NewMetadataRepoFromDB(r.pdb.Metadata)
	langs := PreferredLangsForLocale("fr")
	for s := range slots {
		if len(slots[s].ids) == 0 {
			continue
		}
		names, err := meta.ResolveAssetNamesBulk(ctx, slots[s].assetType, slots[s].ids, langs)
		if err != nil {
			slog.WarnContext(ctx, "media: fallback asset_translations échoué",
				"asset_type", slots[s].assetType, "err", err)
			continue
		}
		slots[s].names = names
	}
}

// applyMediaFallbackNames écrit les noms résolus dans les rows (slots remplis
// par resolveMediaFallbackNames).
func applyMediaFallbackNames(slots []mediaNameFallbackSlot, rows map[string]mediaMatchRegistryInfo) {
	for id, info := range rows {
		changed := false
		for s := range slots {
			if !slots[s].missing(&info) {
				continue
			}
			if n := strings.TrimSpace(slots[s].names[slots[s].id(&info)]); n != "" {
				slots[s].set(&info, n)
				changed = true
			}
		}
		if changed {
			rows[id] = info
		}
	}
}
