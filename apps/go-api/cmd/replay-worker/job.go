package main

// job.go — le TRAVAIL de l'ouvrier : télécharger, décoder, rendre.
//
// Le pont disque (filmcache.Write) est réutilisé tel quel : le décodeur lit un
// dossier de morceaux, et il n'existe qu'UNE disposition de cache film dans le
// dépôt (garde-rail filmcache_guard_test.go). L'ouvrier écrit donc dans cette
// disposition-là, pas dans une mise en page à lui.

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/replaybuild"
)

// heartbeatInterval : cadence des signes de vie pendant un travail long. Trois
// battements manqués valent « hors ligne » côté serveur (domain.WorkerOfflineAfter).
const heartbeatInterval = 30 * time.Second

// chunkDownloadTimeout borne le téléchargement d'un morceau (CDN Azure).
const chunkDownloadTimeout = 60 * time.Second

// worker : l'état LOCAL de l'ouvrier. Rien d'autre que des compteurs — l'état de
// la file vit côté web, et c'est ce qui rend le travail distant observable.
type worker struct {
	identity   workerIdentity
	client     *protocolClient
	repoRoot   string
	workDir    string
	jobsDone   int64
	jobsFailed int64
}

// processJob traite un job pris de bout en bout et en rend le résultat. Ne
// retourne jamais d'erreur : un travail raté est un job `failed` rendu au
// serveur, pas un ouvrier qui tombe.
func (w *worker) processJob(ctx context.Context, job *domain.BuildQueueJob) {
	slog.InfoContext(ctx, "replay-worker: job pris",
		"job_id", job.JobID, "match_id", job.MatchID, "attempt", job.Attempt)

	beatCtx, stopBeat := context.WithCancel(ctx)
	go w.beatUntil(beatCtx, job.JobID)

	result, err := w.buildReplay(ctx, job)
	stopBeat()

	req := handlers.BuildQueueCompleteRequest{JobID: job.JobID, WorkerID: w.identity.workerID}
	if err != nil {
		w.jobsFailed++
		req.Succeeded = false
		req.ErrorCode = "replay_build_failed"
		req.ErrorMessage = err.Error()
		slog.WarnContext(ctx, "replay-worker: job échoué",
			"job_id", job.JobID, "match_id", job.MatchID, "err", err)
	} else {
		w.jobsDone++
		req.Succeeded = true
		req.ResultJSON = result
		slog.InfoContext(ctx, "replay-worker: job réussi",
			"job_id", job.JobID, "match_id", job.MatchID, "result", result)
	}
	// Le rendu part sur un contexte frais : un Ctrl-C pendant le décodage ne doit
	// pas empêcher de dire au serveur ce qui s'est passé. Sans ça, le job
	// resterait `running` jusqu'à l'expiration de son bail — vrai, mais lent.
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpTimeout)
	defer cancel()
	if cerr := w.client.complete(rctx, req); cerr != nil {
		slog.ErrorContext(ctx, "replay-worker: rendu du résultat échoué",
			"job_id", job.JobID, "err", cerr)
	}
}

// buildReplay télécharge les morceaux puis construit l'artefact. Rend le résumé
// JSON du travail (ce que l'admin verra dans la colonne résultat).
func (w *worker) buildReplay(ctx context.Context, job *domain.BuildQueueJob) (string, error) {
	p := job.Payload
	if p == nil || len(p.Chunks) == 0 {
		return "", fmt.Errorf("job sans travail résolu (aucune URL de morceau)")
	}
	chunks, err := w.fetchChunks(ctx, p)
	if err != nil {
		return "", err
	}
	if err := filmcache.Write(w.workDir, p.ShortID, chunks); err != nil {
		return "", fmt.Errorf("écriture du cache film : %w", err)
	}

	builder, err := replaybuild.NewBuilder(w.repoRoot, p.TitleSlug)
	if err != nil {
		return "", fmt.Errorf("catalogue de titre indisponible : %w", err)
	}
	out, err := builder.BuildMatch(p.MatchID, p.MapNames, filmcache.ChunkDir(w.workDir, p.ShortID))
	if err != nil {
		return "", err
	}
	summary, err := json.Marshal(map[string]any{
		"match_id":      p.MatchID,
		"module":        out.Module,
		"tracks":        out.Tracks,
		"bytes":         out.Bytes,
		"artifact_path": out.ArtifactPath,
		"chunks":        len(chunks),
	})
	if err != nil {
		return "", fmt.Errorf("sérialisation du résultat : %w", err)
	}
	return string(summary), nil
}

// fetchChunks télécharge les morceaux depuis les URL PRÉ-SIGNÉES du job et les
// décompresse (le CDN Azure des films rend du zlib brut). Aucune authentification
// n'est présentée : c'est exactement la propriété qui permet à cet ouvrier de
// n'avoir aucun secret Halo.
func (w *worker) fetchChunks(ctx context.Context, p *domain.BuildQueuePayload) ([]filmcache.WriteChunk, error) {
	client := &http.Client{Timeout: chunkDownloadTimeout}
	out := make([]filmcache.WriteChunk, 0, len(p.Chunks))
	for _, c := range p.Chunks {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		data, err := downloadChunk(ctx, client, c.URL)
		if err != nil {
			return nil, fmt.Errorf("morceau %d : %w", c.Index, err)
		}
		out = append(out, filmcache.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType,
			StartMS: c.StartMS, DurationMS: c.DurationMS, Data: data,
		})
	}
	return out, nil
}

// downloadChunk lit un blob CDN et le décompresse.
func downloadChunk(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d (URL pré-signée expirée ?)", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("en-tête zlib : %w", err)
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

// beatUntil bat tant que le job est en cours. C'est ce battement qui prolonge le
// bail : sans lui, un décodage plus long que le bail verrait son job repris par
// la file alors qu'il avance très bien.
func (w *worker) beatUntil(ctx context.Context, jobID string) {
	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			bctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), httpTimeout)
			err := w.client.heartbeat(bctx, w.identity, heartbeat{
				jobID: jobID, note: "décodage en cours",
				done: w.jobsDone, failed: w.jobsFailed,
			})
			cancel()
			if err != nil {
				// Un battement perdu n'est pas fatal : le bail tient 5 min, il en
				// reste plusieurs à venir. Loguer, jamais avaler (règle n°3).
				slog.WarnContext(ctx, "replay-worker: battement non transmis", "job_id", jobID, "err", err)
			}
		}
	}
}
