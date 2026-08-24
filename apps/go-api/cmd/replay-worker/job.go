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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/port"
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
	identity workerIdentity
	client   *protocolClient
	repoRoot string
	workDir  string
	// keepsFilms : le dossier de travail EST le cache film du dépôt (archive
	// perpétuelle) — les morceaux ne sont alors jamais supprimés. Cf. cleanupFilm.
	keepsFilms bool
	jobsDone   int64
	jobsFailed int64
}

// processJob traite un job pris de bout en bout et en rend le résultat. Ne
// retourne jamais d'erreur : un travail raté est un job `failed` rendu au
// serveur, pas un ouvrier qui tombe.
//
// L'ORDRE EST L'ARTEFACT PUIS LE COMPTE RENDU, et c'est le cœur du transport :
// tant que le fichier n'est pas arrivé chez le web, il n'y a rien à annoncer. Si
// l'envoi échoue, l'ouvrier ne marque RIEN — le job reste `running`, son bail
// expire, et il repart en file. Un compte rendu de succès sans artefact serait un
// mensonge que le serveur refuserait de toute façon.
func (w *worker) processJob(ctx context.Context, job *domain.BuildQueueJob) {
	slog.InfoContext(ctx, "replay-worker: job pris",
		"job_id", job.JobID, "match_id", job.MatchID, "attempt", job.Attempt)

	beatCtx, stopBeat := context.WithCancel(ctx)
	go w.beatUntil(beatCtx, job.JobID)

	result, err := w.buildAndSend(ctx, job)
	stopBeat()
	w.cleanupFilm(ctx, job)

	if errors.Is(err, errArtifactNotDelivered) {
		// Rien à rendre : le serveur n'a pas le fichier, le bail tranchera.
		slog.ErrorContext(ctx, "replay-worker: artefact non transmis — job laissé au bail",
			"job_id", job.JobID, "match_id", job.MatchID, "err", err)
		w.jobsFailed++
		return
	}

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

// errArtifactNotDelivered : l'artefact a été construit mais n'est pas arrivé chez
// le web. Distinct d'un échec de travail : il ne se rend PAS au serveur (rien à
// annoncer), il laisse le bail expirer.
var errArtifactNotDelivered = errors.New("artefact non transmis")

// buildAndSend construit l'artefact PUIS le pousse au web. Rend le résumé JSON du
// travail (ce que l'admin verra dans la colonne résultat) — l'accusé du serveur y
// figure, parce que c'est lui, et pas la construction locale, qui prouve la
// livraison.
func (w *worker) buildAndSend(ctx context.Context, job *domain.BuildQueueJob) (string, error) {
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
	// L'OUVRIER N'A PAS DE BASE : les faits du match lui arrivent DANS LE JOB, résolus par le
	// web au moment de la mise en file (cf. domain.BuildQueuePayload.Facts). C'est ce qui lui
	// permet de rendre un artefact COMPLET sans jamais toucher une DuckDB.
	var facts port.MatchFacts
	if p.Facts != nil {
		facts = *p.Facts
	}
	if facts.Empty() {
		// DÉGRADATION ANNONCÉE, JAMAIS MUETTE. Cas réel : un match hors registre, ou un job
		// enfilé par une version antérieure du serveur (le payload est stocké tel quel dans la
		// file, il ne se met pas à jour tout seul). L'artefact reste VALIDE, seulement appauvri —
		// et la liste ci-dessous est MESURÉE (témoin 7344d24f, 2026-08-24), pas supposée.
		slog.WarnContext(ctx, "replay-worker: aucun fait de match dans le job — artefact sans "+
			"actions d'objectif, sans zones de mode, sans socles de drapeau et sans compteurs de "+
			"joueur, identité des camps au mieux par les frags",
			"job_id", job.JobID, "match_id", p.MatchID)
	} else {
		slog.InfoContext(ctx, "replay-worker: faits du match reçus dans le job",
			"job_id", job.JobID, "match_id", p.MatchID, "joueurs", len(facts.Players),
			"variante", facts.GameVariantName, "carte", facts.MapID)
	}
	out, err := builder.BuildMatch(p.MatchID, p.MapNames, filmcache.ChunkDir(w.workDir, p.ShortID), facts)
	if err != nil {
		return "", err
	}
	blob, err := os.ReadFile(out.ArtifactPath)
	if err != nil {
		return "", fmt.Errorf("relecture de l'artefact construit : %w", err)
	}
	// Contexte frais : un Ctrl-C pendant le décodage ne doit pas jeter un artefact
	// qui a coûté 50 s de CPU alors qu'il ne reste qu'à l'envoyer.
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), artifactUploadTimeout)
	defer cancel()
	receipt, err := w.client.sendArtifact(sctx, job.JobID, w.identity.workerID, blob)
	if err != nil {
		return "", fmt.Errorf("%w : %v", errArtifactNotDelivered, err)
	}
	slog.InfoContext(ctx, "replay-worker: artefact transmis",
		"job_id", job.JobID, "match_id", p.MatchID, "bytes", receipt.Bytes,
		"schema", receipt.SchemaVersion)

	summary, err := json.Marshal(map[string]any{
		"match_id":      p.MatchID,
		"module":        out.Module,
		"tracks":        out.Tracks,
		"bytes":         receipt.Bytes,
		"artifact_path": out.ArtifactPath,
		"chunks":        len(chunks),
	})
	if err != nil {
		return "", fmt.Errorf("sérialisation du résultat : %w", err)
	}
	return string(summary), nil
}

// cleanupFilm supprime les morceaux de film que CET ouvrier a téléchargés : il ne
// conserve rien (piste F §1), et 24 Mo par match rempliraient vite une machine de
// calcul.
//
// SAUF SI SON DOSSIER DE TRAVAIL EST LE CACHE FILM DU DÉPÔT. C'est le cas par
// défaut quand l'ouvrier tourne sur le poste de développement, et ce cache est une
// ARCHIVE IRREMPLAÇABLE : les films expirent côté serveur Halo (29,3 % du corpus
// déjà perdus), un film effacé ne se re-télécharge pas. Un ouvrier ne détruit
// jamais l'archive de la machine qui l'héberge ; pour un ouvrier distant qui doit
// nettoyer, --work désigne un dossier de travail à lui.
func (w *worker) cleanupFilm(ctx context.Context, job *domain.BuildQueueJob) {
	if job.Payload == nil || job.Payload.ShortID == "" {
		return
	}
	if w.keepsFilms {
		slog.DebugContext(ctx, "replay-worker: morceaux conservés (cache film du dépôt, archive perpétuelle)",
			"match_id", job.MatchID)
		return
	}
	dir := filmcache.ChunkDir(w.workDir, job.Payload.ShortID)
	if err := os.RemoveAll(dir); err != nil {
		// Non fatal : un morceau qui traîne coûte du disque, pas de la justesse.
		slog.WarnContext(ctx, "replay-worker: morceaux de film non supprimés",
			"dir", dir, "err", err)
		return
	}
	slog.InfoContext(ctx, "replay-worker: morceaux de film supprimés", "dir", dir)
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
