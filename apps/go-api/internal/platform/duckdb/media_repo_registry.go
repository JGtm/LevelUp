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
)

// mediaMatchRegistryInfo regroupe les colonnes de shared.match_registry
// nécessaires au pipeline Q37 média. Chargée via SharedReader.
type mediaMatchRegistryInfo struct {
	MatchID        string
	MapID          string
	MapName        string
	MapNameFR      string
	PairName       string
	PairNameFR     string
	PlaylistID     string
	PlaylistName   string
	PlaylistNameFR string
	StartTime      *time.Time // TIMESTAMP naïf (UTC par convention v6+)
	StartTimeUTC   *time.Time // TIMESTAMPTZ (UTC garanti après migration)
	EndTime        *time.Time
	EndTimeUTC     *time.Time
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
	return out, rows.Err()
}
