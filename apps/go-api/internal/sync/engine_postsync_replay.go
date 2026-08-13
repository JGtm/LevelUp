// Package sync — engine_postsync_replay.go : étape post-sync « artefacts de rejeu 2D »
// (fil de l'eau, lot 6 v7.5).
//
// APRÈS runWeaponKills (le film vient d'être exploité pour les kills : c'est le moment où
// il est disponible ET récent), pour chaque match inséré dans la fenêtre
// replay_retention_months :
//
//  1. LE PONT DISQUE — les chunks COMPLETS du film (en-tête + réplication + kill-feed)
//     sont téléchargés et persistés au cache via filmcache.Write. C'est plus qu'un
//     préalable au décodage : les films EXPIRENT côté serveur Halo (~29 % du corpus déjà
//     perdus), et un film persisté est une archive IRREMPLAÇABLE.
//  2. L'ARTEFACT — replaybuild.BuildMatch décode le film du disque et écrit
//     data/cache/replays/{title}/{short8}.json. Le décodage est sérialisé par le verrou
//     process filmdec (partagé avec killsource).
//
// CONDITIONNÉ LOCAL : l'étape n'existe que si le wiring installe le hook, et le wiring ne
// l'installe qu'en environnement non-production — « le VPS web ne décode JAMAIS » (le
// garde de service replay_local_gate protège la lecture ; ce gate-ci protège le CPU du
// VPS). Best-effort : aucun échec ne casse le pipeline post-sync.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/haloclient"
)

// postsyncReplayMaxPerCycle borne le nombre d'artefacts construits par cycle : un sync
// initial insère des centaines de matchs, et décoder des centaines de films dans le
// post-sync bloquerait le cycle pendant des heures. Le solde relève du backfill CLI
// (levelup backfill-replay), idempotent et repris par SchemaVersion.
const postsyncReplayMaxPerCycle = 5

// ReplayArtifactsHook : réglage du fil de l'eau replay, injecté au wiring (cf.
// SyncEngine.replayArtifacts pour le contrat LOCAL).
type ReplayArtifactsHook struct {
	// RetentionMonths relit la fenêtre à CHAQUE cycle (patron scheduler : un PATCH
	// /settings prend effet sans redémarrage). 0 = illimité.
	RetentionMonths func() int
	// CacheRoot : racine du cache film (PathResolver.CacheRootDir).
	CacheRoot string
}

// WithReplayArtifacts installe le fil de l'eau des artefacts de rejeu 2D. À N'APPELER
// QU'EN LOCAL (cfg.IsProduction() == false) : le VPS web ne décode jamais.
func (e *SyncEngine) WithReplayArtifacts(h *ReplayArtifactsHook) *SyncEngine {
	e.replayArtifacts = h
	return e
}

// NewReplayArtifactsHook construit le hook du fil de l'eau replay — LA fabrique unique
// des trois sites de wiring (BuildEngine scheduler, factory V2, handler HTTP), pour que
// le cache root ne se résolve qu'à un endroit. retention est relu à chaque cycle.
func NewReplayArtifactsHook(repoRoot string, retention func() int) *ReplayArtifactsHook {
	return &ReplayArtifactsHook{
		RetentionMonths: retention,
		CacheRoot:       titlePkg.NewPathResolver(repoRoot).CacheRootDir(),
	}
}

// filmChunksFetcher : capacité OPTIONNELLE du client (assertion, pas extension de
// l'interface HaloClient — les mocks des autres étapes n'ont pas à la porter).
type filmChunksFetcher interface {
	GetFilmChunks(ctx context.Context, matchID string) ([]haloclient.FilmChunk, bool, error)
}

// replayBuildWork : un match à construire, avec ses identités de carte candidates.
type replayBuildWork struct {
	matchID  string
	mapNames []string
}

// runReplayArtifacts — étape 1.58 : pont disque + artefacts de rejeu des matchs insérés.
func (s postSyncFilmSteps) runReplayArtifacts(ctx context.Context, insertedIDs []string) {
	e := s.engine
	h := e.replayArtifacts
	if h == nil || len(insertedIDs) == 0 {
		return
	}
	fetcher, ok := s.client.(filmChunksFetcher)
	if !ok {
		slog.DebugContext(ctx, "post-sync: rejeu 2D — client sans GetFilmChunks, étape ignorée",
			"gamertag", e.gamertag)
		return
	}
	months := 0
	if h.RetentionMonths != nil {
		months = h.RetentionMonths()
	}
	var work []replayBuildWork
	s.withRead(ctx, "replay_select", func(sharedDB *sql.DB) {
		work = selectReplayBuildWork(ctx, sharedDB, e.metaDB, insertedIDs, months)
	})
	if len(work) == 0 {
		return
	}
	if len(work) > postsyncReplayMaxPerCycle {
		slog.InfoContext(ctx, "post-sync: rejeu 2D — lot borné, solde au backfill CLI",
			"gamertag", e.gamertag, "selected", len(work), "cap", postsyncReplayMaxPerCycle)
		work = work[:postsyncReplayMaxPerCycle]
	}

	builder, err := replaybuild.NewBuilder(e.repoRoot, e.titleSlug)
	if err != nil {
		// Titre sans catalogue de bornes / labels : dégradation par absence de donnée
		// (title-agnostic), journalisée une fois par cycle — jamais un échec de sync.
		slog.DebugContext(ctx, "post-sync: rejeu 2D indisponible pour ce titre",
			"gamertag", e.gamertag, "titleSlug", e.titleSlug, "err", err)
		return
	}
	built, filmsSaved := 0, 0
	for _, w := range work {
		if ctx.Err() != nil {
			break
		}
		saved, ok := s.persistFilmToCache(ctx, fetcher, h.CacheRoot, w.matchID)
		if !ok {
			continue // film absent/expiré côté serveur : rien à construire (débité en debug)
		}
		if saved {
			filmsSaved++
		}
		short := titlePkg.FilmShortMatchID(w.matchID)
		if replaybuild.ArtifactUpToDate(titlePkg.NewPathResolver(e.repoRoot).ReplayArtifactPath(e.titleSlug, w.matchID)) {
			continue
		}
		out, berr := builder.BuildMatch(w.matchID, w.mapNames, filmcache.ChunkDir(h.CacheRoot, short))
		if berr != nil {
			// Carte hors catalogue = échec voulu (Forge) ; le reste = erreur réelle.
			// Les deux sont best-effort, mais seuls les seconds méritent un WARN.
			logFn := slog.WarnContext
			if strings.Contains(berr.Error(), replaybuild.ErrMapNotInCatalog.Error()) {
				logFn = slog.DebugContext
			}
			logFn(ctx, "post-sync: artefact rejeu non construit",
				"gamertag", e.gamertag, "match_id", w.matchID, "err", berr)
			continue
		}
		built++
		slog.InfoContext(ctx, "post-sync: artefact rejeu construit",
			"gamertag", e.gamertag, "match_id", w.matchID, "tracks", out.Tracks, "bytes", out.Bytes)
	}
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "postsync_replay_artifacts_built_total", int64(built))
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "postsync_replay_films_persisted_total", int64(filmsSaved))
	if built > 0 || filmsSaved > 0 {
		slog.InfoContext(ctx, "post-sync: rejeu 2D",
			"gamertag", e.gamertag, "built", built, "films_persisted", filmsSaved, "selected", len(work))
	}
}

// persistFilmToCache télécharge les chunks COMPLETS du film et les persiste au cache
// (pont disque). Rend (persisté, film disponible). Un film déjà entièrement en cache ne
// re-télécharge rien (GetFilmChunks est cache-first chunk par chunk).
func (s postSyncFilmSteps) persistFilmToCache(
	ctx context.Context, fetcher filmChunksFetcher, cacheRoot, matchID string,
) (saved, available bool) {
	chunks, found, err := fetcher.GetFilmChunks(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: film illisible — rejeu non construit",
			"gamertag", s.engine.gamertag, "match_id", matchID, "err", err)
		return false, false
	}
	if !found || len(chunks) == 0 {
		slog.DebugContext(ctx, "post-sync: film absent côté serveur — rejeu non construit",
			"match_id", matchID)
		return false, false
	}
	wc := make([]filmcache.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmcache.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS,
			DurationMS: c.DurationMS, Data: c.Data,
		})
	}
	if err := filmcache.Write(cacheRoot, titlePkg.FilmShortMatchID(matchID), wc); err != nil {
		slog.WarnContext(ctx, "post-sync: persistance du film au cache échouée",
			"gamertag", s.engine.gamertag, "match_id", matchID, "err", err)
		return false, false
	}
	return true, true
}

// selectReplayBuildWork lit les identités de carte des matchs insérés et applique la
// fenêtre de rétention (months <= 0 = illimité). metaDB peut être nil (pas de résolution
// EN : map_name brut seul, même dégradation que le backfill CLI).
func selectReplayBuildWork(
	ctx context.Context, sharedDB, metaDB *sql.DB, insertedIDs []string, months int,
) []replayBuildWork {
	if len(insertedIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(insertedIDs)), ",")
	args := make([]any, 0, len(insertedIDs))
	for _, id := range insertedIDs {
		args = append(args, id)
	}
	q := fmt.Sprintf(`SELECT match_id, map_name, map_id, %s AS start_canonical
		FROM match_registry WHERE match_id IN (%s)`,
		analysis.SQLStartTimeCanonical("match_registry"), placeholders)
	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: sélection rejeu échouée", "err", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	var cutoff time.Time
	if months > 0 {
		cutoff = time.Now().UTC().AddDate(0, -months, 0)
	}
	var out []replayBuildWork
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		var start sql.NullTime
		if err := rows.Scan(&id, &rawName, &mapID, &start); err != nil {
			slog.WarnContext(ctx, "post-sync: sélection rejeu (scan)", "err", err)
			return out
		}
		if months > 0 && start.Valid && start.Time.Before(cutoff) {
			continue // hors fenêtre de rétention : le backfill CLI reste libre de le faire
		}
		var names []string
		if en := resolveMapNameEN(ctx, metaDB, strings.TrimSpace(mapID.String)); en != "" {
			names = append(names, en)
		}
		if raw := strings.TrimSpace(rawName.String); raw != "" {
			names = append(names, raw)
		}
		out = append(out, replayBuildWork{matchID: id, mapNames: names})
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: sélection rejeu (rows)", "err", err)
	}
	return out
}

// resolveMapNameEN résout le nom EN d'une carte par son asset UGC (asset_translations).
// Best-effort : metaDB nil ou nom absent → "" (le candidat brut reste).
func resolveMapNameEN(ctx context.Context, metaDB *sql.DB, mapID string) string {
	if metaDB == nil || mapID == "" {
		return ""
	}
	var en string
	err := metaDB.QueryRowContext(ctx,
		`SELECT name FROM asset_translations WHERE asset_type = 'map' AND asset_id = ? AND lang = 'en-US'`,
		mapID).Scan(&en)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(en)
}
