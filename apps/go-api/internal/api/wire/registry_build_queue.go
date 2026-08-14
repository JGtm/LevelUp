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
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/replaybuild"
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

// ReplayPlacement rend la décision courante « où se construit un rejeu », lue au
// moment de l'appel (le réglage vit dans app_settings, éditable sans redémarrage).
//
// UN SEUL POINT DE DÉCISION, TROIS APPELANTS : le fil de l'eau post-sync
// (replayartifacts.Hook.Placement), l'action admin (ce runner), et — délibérément
// PAS — le CLI de backfill, outil d'opérateur qui garde son chemin direct.
func (r *ServiceRegistry) ReplayPlacement() replaybuild.Placement {
	setting := ""
	if r.settingsStore != nil {
		if s, _ := r.settingsStore.Load(); s != nil {
			setting = strings.TrimSpace(s.ReplayBuildLocation)
		}
	}
	p, err := replaybuild.DecidePlacement(setting, replaybuild.PlacementEnv{
		Production:       r.cfg.IsProduction(),
		WorkerConfigured: strings.TrimSpace(r.cfg.BuildWorkerToken) != "",
	})
	replaybuild.LogPlacement("admin", p, err)
	return p
}

// EnqueueReplayBuildJob est la forme attendue par le fil de l'eau post-sync
// (replayartifacts.EnqueueFunc) : le job et l'idempotence ne l'intéressent pas,
// seulement le fait que la mise en file ait abouti. Adaptateur, pas duplication.
func (r *ServiceRegistry) EnqueueReplayBuildJob(ctx context.Context, titleSlug, matchID string) error {
	_, _, err := r.EnqueueReplayBuild(ctx, titleSlug, matchID)
	return err
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

// StoreBuildArtifact valide l'artefact rendu par un ouvrier et le range à la
// place canonique du match. C'est le dernier maillon du transport : l'ouvrier
// pousse, le web valide et RANGE (piste F §1).
//
// L'IDENTITÉ DU MATCH VIENT DU JOB, JAMAIS DU FICHIER. Le job dit quel match cet
// ouvrier a le droit d'écrire ; l'artefact doit s'y conformer (sinon refus).
// C'est ce qui empêche un dépôt d'atterrir sous le nom d'un autre match.
func (r *ServiceRegistry) StoreBuildArtifact(ctx context.Context, jobID, workerID string, blob []byte) (domain.BuildArtifactReceipt, error) {
	if r.monitoringStore == nil {
		return domain.BuildArtifactReceipt{}, fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	job, err := r.monitoringStore.ClaimedBuildJob(ctx, jobID, workerID)
	if err != nil {
		observability.IncCounter("build_queue_artifacts_rejected_not_claimed_total")
		return domain.BuildArtifactReceipt{}, err
	}
	slug := job.TitleSlug
	if slug == "" {
		slug = titlePkg.DefaultSlug
	}
	stored, err := replaybuild.StoreArtifact(r.cfg.RepoRoot, slug, job.MatchID, blob)
	if err != nil {
		if errors.Is(err, domain.ErrBuildArtifactInvalid) {
			observability.IncCounter("build_queue_artifacts_rejected_invalid_total")
		} else {
			observability.IncCounter("build_queue_artifacts_rejected_write_total")
		}
		return domain.BuildArtifactReceipt{}, err
	}
	observability.IncCounter("build_queue_artifacts_received_total")
	observability.AddInt("build_queue_artifact_bytes_total", int64(stored.Bytes))
	monitoringLog.InfoContext(ctx, "build queue: artefact reçu et rangé",
		"job_id", jobID, "worker_id", workerID, "match_id", job.MatchID, "title", slug,
		"bytes", stored.Bytes, "tracks", stored.Tracks, "path", stored.Path)
	return domain.BuildArtifactReceipt{
		JobID: jobID, MatchID: job.MatchID,
		Bytes: stored.Bytes, SchemaVersion: stored.SchemaVersion,
	}, nil
}

// ClaimBuildJob prend le prochain job pour un ouvrier (protocole ouvrier).
func (r *ServiceRegistry) ClaimBuildJob(ctx context.Context, workerID, hostname, version string) (*domain.BuildQueueJob, error) {
	if r.monitoringStore == nil {
		return nil, fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	return r.monitoringStore.ClaimBuildJob(ctx, workerID, hostname, version)
}

// CompleteBuildJob enregistre le résultat d'un job (protocole ouvrier).
//
// LE COMPTE RENDU EST LE POINT FINAL, PAS LE POINT DE PASSAGE. Un job de
// construction de rejeu ne peut être déclaré RÉUSSI que si son artefact est
// arrivé et porte la version de schéma courante : c'est le fichier qui fait foi,
// pas la parole de l'ouvrier. Un succès annoncé sans artefact est REFUSÉ (409) ;
// le job reste `running`, son bail expire, et il repart en file — mécanique déjà
// livrée, rien à ajouter pour que le travail ne se perde pas.
func (r *ServiceRegistry) CompleteBuildJob(ctx context.Context, req ops.CompleteBuildJobRequest) error {
	if r.monitoringStore == nil {
		return fmt.Errorf("file de construction indisponible (store monitoring non ouvert)")
	}
	if err := r.requireArtifactBeforeSuccess(ctx, req); err != nil {
		return err
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

// requireArtifactBeforeSuccess refuse un compte rendu de SUCCÈS dont l'artefact
// n'est pas là. Ne concerne que les jobs de construction de rejeu — les autres
// types de travail n'ont pas de fichier à produire.
//
// La preuve est le FICHIER à la place canonique, avec la bonne version de schéma
// (replaybuild.ArtifactUpToDate) : un artefact d'une version antérieure ne vaut
// pas mieux qu'une absence, il faut le reconstruire.
func (r *ServiceRegistry) requireArtifactBeforeSuccess(ctx context.Context, req ops.CompleteBuildJobRequest) error {
	if !req.Succeeded {
		return nil
	}
	job, err := r.monitoringStore.ClaimedBuildJob(ctx, req.JobID, req.WorkerID)
	if err != nil {
		return err // ErrBuildJobNotClaimed → 409, même refus que le rendu périmé
	}
	if job.JobType != string(domain.JobTypeReplayBuild) {
		return nil
	}
	slug := job.TitleSlug
	if slug == "" {
		slug = titlePkg.DefaultSlug
	}
	path := titlePkg.NewPathResolver(r.cfg.RepoRoot).ReplayArtifactPath(slug, job.MatchID)
	if replaybuild.ArtifactUpToDate(path) {
		return nil
	}
	observability.IncCounter("build_queue_complete_without_artifact_total")
	monitoringLog.WarnContext(ctx, "build queue: succès annoncé sans artefact — refusé",
		"job_id", req.JobID, "worker_id", req.WorkerID, "match_id", job.MatchID, "path", path)
	return fmt.Errorf("%w (aucun artefact à jour reçu pour %s)", ops.ErrBuildJobNotClaimed, job.MatchID)
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
