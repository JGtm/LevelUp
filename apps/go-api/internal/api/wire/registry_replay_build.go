// Package api — registry_replay_build.go : runner de l'action admin « construire le
// rejeu 2D d'un match ».
//
// IN-PROCESS PAR LA LIBRAIRIE internal/replaybuild — JAMAIS un exec de CLI (règle
// admin_actions : conflit DuckDB mono-process, et un exec échapperait au verrou process
// filmdec partagé avec killsource). Single-flight : le décodage d'un film sature un
// coeur pendant des secondes à minutes ; deux jobs simultanés s'entasseraient de toute
// façon derrière le verrou filmdec.
package wire

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	gosync "sync"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
)

// replayBuildMu sérialise l'action replay-build (single-flight, pattern registryNamesMu).
var replayBuildMu gosync.Mutex

// RunReplayBuild construit l'artefact de rejeu 2D d'un match depuis le cache film LOCAL.
// Bloquant (appelé dans la goroutine du job). Les handles base sont relâchés AVANT le
// décodage : on ne tient pas une lecture shared pendant des minutes de CPU.
func (r *ServiceRegistry) RunReplayBuild(ctx context.Context, titleSlug, matchID string) (map[string]any, error) {
	if !replayBuildMu.TryLock() {
		return nil, ErrActionBusy
	}
	defer replayBuildMu.Unlock()

	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	fullID, names, err := replayMatchIdentity(ctx, sharedSQL, metaSQL, matchID)
	closeAll()
	if err != nil {
		return nil, err
	}

	cacheRoot := titlePkg.NewPathResolver(r.cfg.RepoRoot).CacheRootDir()
	short := titlePkg.FilmShortMatchID(fullID)
	if _, found, ferr := filmcache.Open(cacheRoot, short); ferr != nil {
		return nil, fmt.Errorf("cache film illisible: %w", ferr)
	} else if !found {
		return nil, fmt.Errorf("film absent du cache local pour %s — cette action est hors ligne, elle ne télécharge rien", fullID)
	}

	builder, err := replaybuild.NewBuilder(r.cfg.RepoRoot, titleSlug)
	if err != nil {
		return nil, err
	}
	out, err := builder.BuildMatch(fullID, names, filmcache.ChunkDir(cacheRoot, short))
	if err != nil {
		return nil, err
	}
	monitoringLog.InfoContext(ctx, "admin_actions: rejeu 2D construit",
		"title", titleSlug, "match_id", fullID, "tracks", out.Tracks,
		"bytes", out.Bytes, "module", out.Module)
	observability.IncCounter("admin_action_replay_build_total")
	return map[string]any{
		"match_id": fullID,
		"module":   out.Module,
		"tracks":   out.Tracks,
		"bytes":    out.Bytes,
		"path":     out.ArtifactPath,
	}, nil
}

// replayMatchIdentity résout le match (forme courte ACCEPTÉE — piège lot 4 :
// match_registry est indexé par match_id COMPLET) et rend ses identités de carte
// candidates, du plus fiable au moins fiable (nom d'asset EN puis map_name brut, même
// ordre que ReplayMapRepo). metaSQL peut être nil (dégradation : brut seul).
func replayMatchIdentity(ctx context.Context, sharedSQL, metaSQL *sql.DB, matchID string) (string, []string, error) {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" {
		return "", nil, fmt.Errorf("match_id vide")
	}
	rows, err := sharedSQL.QueryContext(ctx,
		`SELECT match_id, map_name, map_id FROM match_registry
		 WHERE match_id = ? OR match_id LIKE ? LIMIT 3`,
		matchID, matchID+"%")
	if err != nil {
		return "", nil, fmt.Errorf("lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type hit struct {
		id      string
		rawName string
		mapID   string
	}
	var hits []hit
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		if err := rows.Scan(&id, &rawName, &mapID); err != nil {
			return "", nil, fmt.Errorf("lecture match_registry (scan): %w", err)
		}
		hits = append(hits, hit{id: id, rawName: strings.TrimSpace(rawName.String), mapID: strings.TrimSpace(mapID.String)})
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	if len(hits) == 0 {
		return "", nil, fmt.Errorf("match inconnu du registre: %s", matchID)
	}
	if len(hits) > 1 {
		return "", nil, fmt.Errorf("préfixe ambigu %s (%d matchs) — donner le match_id complet", matchID, len(hits))
	}
	h := hits[0]
	var names []string
	if h.mapID != "" && metaSQL != nil {
		var en string
		if err := metaSQL.QueryRowContext(ctx,
			`SELECT name FROM asset_translations WHERE asset_type = 'map' AND asset_id = ? AND lang = 'en-US'`,
			h.mapID).Scan(&en); err == nil && strings.TrimSpace(en) != "" {
			names = append(names, strings.TrimSpace(en))
		}
	}
	if h.rawName != "" {
		names = append(names, h.rawName)
	}
	return h.id, names, nil
}
