// Package api — registry_build_queue.go : runners de la FILE DURABLE de
// construction et du protocole ouvrier (piste F §1/§2/§4bis).
//
// LA FRONTIÈRE DE SÉCURITÉ EST ICI. EnqueueReplayBuild est le seul endroit du
// système où les tokens Halo servent pour un travail délégué : il résout le
// MANIFESTE du film (authentifié) et dépose les URL CDN PRÉ-SIGNÉES (non
// authentifiées) dans le job. Tout ce que l'ouvrier verra jamais, c'est ça.
// Déplacer cette résolution chez l'ouvrier lui imposerait un token Halo et
// ruinerait le découpage — c'est la ligne à ne pas franchir.
//
// Tous les runners dégradent proprement quand le store monitoring n'est pas
// ouvert (erreur explicite, jamais de panic) : sans lui, il n'y a pas de file, et
// le rejeu retombe sur son chemin local existant.
package wire

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	syncpkg "levelup/go-api/internal/sync"
)

// replayManifestRPS : débit de résolution des manifestes de film. Très
// conservateur — une mise en file est un geste unitaire d'admin ou de post-sync,
// jamais une rafale.
const replayManifestRPS = 3

// EnqueueReplayBuild met en file la construction du rejeu 2D d'un match, travail
// RÉSOLU compris (URL CDN pré-signées de tous les morceaux du film).
//
// Rend (job, créé). créé=false quand le même match est déjà en file ou en cours :
// re-cliquer ne double pas le travail.
//
// Un film absent côté serveur (~29 % du corpus, ils expirent) est une ERREUR de
// mise en file, pas un job qui échouera plus tard chez l'ouvrier : mieux vaut le
// dire tout de suite à celui qui clique.
func (r *ServiceRegistry) EnqueueReplayBuild(ctx context.Context, titleSlug, matchID string) (domain.BuildQueueJob, bool, error) {
	if r.monitoringStore == nil {
		return domain.BuildQueueJob{}, false, fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}
	fullID, names, err := replayMatchIdentity(ctx, sharedSQL, metaSQL, matchID)
	closeAll()
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}

	chunks, err := r.resolveFilmChunkURLs(ctx, titleSlug, fullID)
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}

	job, created, err := r.monitoringStore.EnqueueBuildJob(ctx, ops.EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: titleSlug,
		MatchID:   fullID,
		Payload: &domain.BuildQueuePayload{
			MatchID:   fullID,
			ShortID:   titlePkg.FilmShortMatchID(fullID),
			TitleSlug: titleSlug,
			MapNames:  names,
			Chunks:    chunks,
		},
	})
	if err != nil {
		return domain.BuildQueueJob{}, false, err
	}
	if created {
		observability.IncCounter("build_queue_replay_enqueued_total")
		monitoringLog.InfoContext(ctx, "build queue: rejeu 2D mis en file",
			"job_id", job.JobID, "match_id", fullID, "title", titleSlug, "chunks", len(chunks))
	}
	return job, created, nil
}

// resolveFilmChunkURLs résout le manifeste du film et rend les URL pré-signées de
// ses morceaux. C'est LE geste qui exige les tokens — et le seul.
func (r *ServiceRegistry) resolveFilmChunkURLs(ctx context.Context, titleSlug, matchID string) ([]domain.BuildQueueChunk, error) {
	tokens, err := r.haloTokensForDrain(ctx, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("résolution du manifeste de film: %w", err)
	}
	client := syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, replayManifestRPS)
	refs, found, err := client.GetFilmChunkURLs(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("résolution du manifeste de film: %w", err)
	}
	if !found || len(refs) == 0 {
		return nil, fmt.Errorf("film absent ou expiré côté serveur pour %s — rien à construire", matchID)
	}
	out := make([]domain.BuildQueueChunk, 0, len(refs))
	for _, ref := range refs {
		out = append(out, domain.BuildQueueChunk{
			Index: ref.Index, ChunkType: ref.ChunkType,
			StartMS: ref.StartMS, DurationMS: ref.DurationMS, URL: ref.URL,
		})
	}
	return out, nil
}

// ClaimBuildJob prend le prochain job pour un ouvrier (protocole ouvrier).
func (r *ServiceRegistry) ClaimBuildJob(ctx context.Context, workerID, hostname, version string) (*domain.BuildQueueJob, error) {
	if r.monitoringStore == nil {
		return nil, fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	return r.monitoringStore.ClaimBuildJob(ctx, workerID, hostname, version)
}

// CompleteBuildJob enregistre le résultat d'un job (protocole ouvrier).
func (r *ServiceRegistry) CompleteBuildJob(ctx context.Context, req ops.CompleteBuildJobRequest) error {
	if r.monitoringStore == nil {
		return fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	if err := r.monitoringStore.CompleteBuildJob(ctx, req); err != nil {
		return err
	}
	if req.Succeeded {
		observability.IncCounter("build_queue_jobs_succeeded_total")
	} else {
		observability.IncCounter("build_queue_jobs_failed_total")
	}
	return nil
}

// HeartbeatBuildWorker enregistre un battement d'ouvrier (protocole ouvrier).
func (r *ServiceRegistry) HeartbeatBuildWorker(ctx context.Context, req ops.HeartbeatRequest) error {
	if r.monitoringStore == nil {
		return fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	return r.monitoringStore.HeartbeatBuildWorker(ctx, req)
}

// BuildQueueReport agrège la file et les ouvriers pour le dashboard admin.
// Enabled dit si le protocole ouvrier est ouvert (jeton configuré) : sans lui, la
// file peut se remplir mais personne ne viendra la vider — l'admin doit le voir.
func (r *ServiceRegistry) BuildQueueReport(ctx context.Context, limit int) (domain.AdminBuildQueueResponse, error) {
	if r.monitoringStore == nil {
		return domain.AdminBuildQueueResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Jobs:        []domain.BuildQueueJob{},
			Workers:     []domain.BuildQueueWorker{},
		}, nil
	}
	resp, err := r.monitoringStore.BuildQueueReport(ctx, limit)
	if err != nil {
		return resp, err
	}
	resp.Enabled = strings.TrimSpace(r.cfg.BuildWorkerToken) != ""
	return resp, nil
}

// reclaimBuildQueue rend à la file les jobs dont le bail a expiré. Greffé sur la
// boucle de flush monitoring (pas un cron de plus) : la reprise doit se voir même
// quand plus aucun ouvrier vivant ne vient déclencher un claim.
func (r *ServiceRegistry) reclaimBuildQueue(ctx context.Context) {
	if r.monitoringStore == nil {
		return
	}
	n, err := r.monitoringStore.ReclaimExpiredBuildJobs(ctx)
	if err != nil {
		monitoringLog.WarnContext(ctx, "build queue: reprise des bails expirés échouée (best-effort)", "err", err)
		return
	}
	if n > 0 {
		observability.AddInt("build_queue_jobs_reclaimed_total", int64(n))
	}
}
