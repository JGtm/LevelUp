// Package api — registry_replay_build.go : runner de l'action admin « construire le
// rejeu 2D d'un match ».
//
// LE DÉCODAGE PART HORS DU PROCESSUS SERVEUR (lot J2.13, constat OPS-2, 2026-09-26), par le
// chemin de l'étape 1.58 du post-sync : `replayartifacts.ConstruireEtRanger` avec la stratégie
// `replayartifacts.SpawnBuildOne` — un ENFANT borne (sentinelle mémoire, priorité basse) qui
// prend le verrou solo `filmproc.AcquireSolo` en refus immédiat et rend les octets, que
// `replaybuild.StoreArtifact` range ICI (garde anti-régression, notification). L'action
// décodait auparavant dans le serveur, hors verrou et sans plafond : le septième point
// d'entrée du décodage, que l'ADR 0034 ne comptait pas. Les faits du match sont lus en base par
// le parent, jamais par l'enfant (modèle mono-processus DuckDB).
//
// Single-flight (`replayBuildMu`) : une seule construction admin à la fois dans ce processus ;
// le verrou solo, lui, arbitre ENTRE les processus (post-sync, passes, outils).
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
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/replayartifacts"
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
	// LES FAITS SE LISENT AVANT DE RELÂCHER LES HANDLES, et jamais pendant le décodage : deux
	// SELECT courts, puis la base n'est plus tenue (le décodage dure des secondes à minutes).
	var facts port.MatchFacts
	if err == nil {
		facts = replayMatchFacts(ctx, sharedSQL, fullID)
	}
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

	return construireRejeuAdmin(ctx, replayartifacts.SpawnBuildOne, replayartifacts.BuildOneRequest{
		MatchID: fullID, TitleSlug: titleSlug, RepoRoot: r.cfg.RepoRoot,
		MapNames: names, FilmDir: filmcache.ChunkDir(cacheRoot, short), Facts: facts,
	})
}

// construireRejeuAdmin construit et range l'artefact d'UN match pour l'action admin, par le
// chemin de l'etape 1.58 : `build` (en production l'enfant borne) rend les octets, et
// `StoreArtifact` les range ici. Le decodage ne se fait JAMAIS dans ce processus.
func construireRejeuAdmin(ctx context.Context, build replayartifacts.BuildOneFunc,
	req replayartifacts.BuildOneRequest,
) (map[string]any, error) {
	stored, res, err := replayartifacts.ConstruireEtRanger(ctx, build, req)
	if err != nil {
		return nil, err
	}
	monitoringLog.InfoContext(ctx, "admin_actions: rejeu 2D construit",
		"title", req.TitleSlug, "match_id", req.MatchID, "tracks", stored.Tracks,
		"bytes", stored.Bytes, "duration", res.Dur, "pic_octets", res.Peak)
	observability.IncCounter("admin_action_replay_build_total")
	return map[string]any{
		"match_id":       req.MatchID,
		"tracks":         stored.Tracks,
		"bytes":          stored.Bytes,
		"schema_version": stored.SchemaVersion,
		"path":           stored.Path,
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

// replayMatchFacts lit CE QUE LA BASE SAIT DU MATCH et que le film ne dit pas : les lignes de
// match (pont d'identité des joueurs), les scores des deux camps (identité des camps) et le nom
// de variante (famille d'objectif). Dégradation journalisée, jamais fatale : un artefact sans
// ces faits reste valide, seulement sans compteurs de joueur ni actions d'objectif.
func replayMatchFacts(ctx context.Context, sharedSQL *sql.DB, matchID string) port.MatchFacts {
	// Le type concret ne sort pas d ici : l appelant ne connait que le port.
	var repo port.ReplayFactsRepo = duckdb.NewReplayFactsRepo(sharedSQL)
	facts, err := repo.FactsForMatch(ctx, matchID)
	if err != nil {
		// DEGRADATION, PAS ECHEC : un artefact sans ces faits reste valide et servi ; refuser de
		// construire pour une lecture ratee couterait le rejeu entier.
		monitoringLog.WarnContext(ctx, "admin_actions: faits de match illisibles — rejeu sans courbe de score complète",
			"err", err, "match_id", matchID)
		return port.MatchFacts{}
	}
	return facts
}
