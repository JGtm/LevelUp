package service

import (
	"context"
	"fmt"
	"levelup/go-api/internal/domain"
	mediapkg "levelup/go-api/internal/media"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func (s *MediaService) UploadMedia(ctx context.Context, req domain.UploadRequest) (*domain.UploadResult, error) {
	result := &domain.UploadResult{}

	if len(req.Files) == 0 {
		return result, nil
	}

	if err := os.MkdirAll(req.CapturesDir, 0o755); err != nil {
		return nil, fmt.Errorf("créer captures dir %q: %w", req.CapturesDir, err)
	}

	var savedVideos []string // chemins abs des vidéos sauvées (candidats transcoding HLS)
	for _, f := range req.Files {
		// Dédup à la source : si un fichier du même nom ET du même contenu existe
		// déjà sur disque, c'est un ré-upload du même média. On skippe l'écriture
		// pour ne pas créer une copie disque redondante — l'indexation l'ignorerait
		// ensuite par hash, en laissant un fichier orphelin.
		sameName := filepath.Join(req.CapturesDir, filepath.Base(f.OriginalName))
		if existingHash, err := ops.HashFile(sameName); err == nil && ops.HashBytes(f.Data) == existingHash {
			result.Skipped++
			slog.DebugContext(ctx, "upload: doublon ignoré (même nom + même contenu)",
				"file", f.OriginalName)
			continue
		}

		dest := safeDestPath(req.CapturesDir, f.OriginalName)
		if err := os.WriteFile(dest, f.Data, 0o644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", f.OriginalName, err))
			slog.WarnContext(ctx, "upload: écriture fichier échouée",
				"file", f.OriginalName, "err", err)
			continue
		}
		result.Saved++
		slog.DebugContext(ctx, "upload: fichier sauvegardé",
			"file", f.OriginalName, "dest", dest)
		if mediapkg.IsVideoExt(filepath.Ext(dest)) {
			savedVideos = append(savedVideos, dest)
		}
	}

	if result.Saved == 0 && result.Skipped == 0 {
		slog.WarnContext(ctx, "upload: aucun fichier sauvegardé", "attempted", len(req.Files))
		return result, nil
	}
	if result.Saved == 0 {
		// Tous les fichiers étaient déjà présents (ré-upload du même média). On NE
		// court-circuite PAS l'indexation : re-uploader est le geste naturel pour
		// "réparer" un média, et IndexMedia relance l'association (idempotente).
		// Indispensable depuis la dédup à la source — sinon un ré-upload ne pourrait
		// plus jamais rattraper une association précédemment échouée (incident
		// 2026-06-03 : médias non-associés suite à un conflit de lock).
		slog.InfoContext(ctx, "upload: aucun nouveau fichier, ré-indexation pour (re)association",
			"attempted", len(req.Files), "skipped", result.Skipped)
	}

	tol := req.Tolerance
	if tol <= 0 {
		tol = 2
	}

	// Construction de la map basename → unix ts client (pour le filename parser)
	captureTimes := make(map[string]*int64, len(req.Files))
	for _, f := range req.Files {
		if f.CaptureTimeUnix != nil {
			captureTimeUnix := f.CaptureTimeUnix
			captureTimes[f.OriginalName] = captureTimeUnix
		}
	}

	slog.InfoContext(ctx, "upload: démarrage indexation",
		"captures_dir", req.CapturesDir, "buffer_min", tol, "timezone", s.timezone)

	idxResult, err := ops.IndexMedia(ctx, ops.MediaIndexOptions{
		PlayerDBPath:        req.DBPath,
		SharedSocialDBPath:  req.SharedSocialDBPath,
		SharedMatchesDBPath: req.SharedMatchesDBPath,
		CapturesDir:         req.CapturesDir,
		CapturesBase:        req.CapturesBase,
		Gamertag:            req.Gamertag,
		BufferMin:           tol,
		Timezone:            s.timezone,
		TitleSlug:           req.TitleSlug,
		CaptureTimes:        captureTimes,
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("indexation: %v", err))
		slog.ErrorContext(ctx, "upload: indexation échouée", "err", err)
	}
	result.NewIndexed = idxResult.NewFiles
	result.Associated = idxResult.Associated
	result.Thumbnails = idxResult.Thumbnails
	result.Errors = append(result.Errors, idxResult.Errors...)

	// Transcoding HLS asynchrone des vidéos éligibles. Les thumbnails ont déjà
	// été générés par IndexMedia (donc disponibles même pendant le 'processing').
	if s.jobStore != nil {
		s.launchHLSTranscoding(ctx, req, savedVideos)
	}

	slog.InfoContext(ctx, "upload: terminé",
		"saved", result.Saved,
		"skipped", result.Skipped,
		"new_indexed", result.NewIndexed,
		"associated", result.Associated,
		"thumbnails", result.Thumbnails,
		"errors", len(result.Errors),
	)
	return result, nil
}

// launchHLSTranscoding détecte les vidéos éligibles au HLS (MKV/AVI ou
// multipiste), les marque 'processing' et lance un worker async par clip. La
// détection (probe) est synchrone et rapide ; le transcoding tourne en goroutine
// détachée pour ne pas bloquer la réponse upload.
func (s *MediaService) launchHLSTranscoding(ctx context.Context, req domain.UploadRequest, videoPaths []string) {
	log := slog.With("module", "media") // logs/media.log
	store := ops.MediaPathStore{CapturesBase: req.CapturesBase}
	for _, dest := range videoPaths {
		needed, err := ops.DetectHLSNeeded(ctx, dest)
		if err != nil {
			log.WarnContext(ctx, "hls: détection échouée", "file", dest, "err", err)
			continue
		}
		fileRel := store.ToRel(dest, req.Gamertag)
		if fileRel == "" {
			fileRel = dest
		}
		if !needed {
			// Média web-natif servi en direct : on persiste 'direct' pour que le
			// balayage post-sync ne le re-probe jamais.
			if err := ops.MarkTranscodeStatus(ctx, req.SharedSocialDBPath, fileRel, ops.TranscodeDirect); err != nil {
				log.WarnContext(ctx, "hls: mark direct échoué", "file", fileRel, "err", err)
			}
			continue
		}
		outDir, hlsRel := ops.HLSPathsFor(req.CapturesDir, req.CapturesBase, req.Gamertag, dest)
		// Compare-and-set : verrou refusé = un balayage post-sync (ou un autre
		// upload) transcode déjà ce fichier — ne pas lancer un second ffmpeg.
		acquired, err := ops.MarkTranscodeProcessing(ctx, req.SharedSocialDBPath, fileRel)
		if err != nil {
			log.WarnContext(ctx, "hls: mark processing échoué", "file", fileRel, "err", err)
			continue
		}
		if !acquired {
			log.InfoContext(ctx, "hls: transcodage déjà en cours (verrou non acquis) — sauté", "file", fileRel)
			continue
		}
		job := s.jobStore.Create(domain.JobTypeTranscodeMedia, req.Gamertag)
		log.InfoContext(ctx, "hls: transcoding lancé",
			"job", job.JobID, "file", fileRel, "out_dir", outDir)
		go s.runTranscodeJob(ops.HLSTranscodeParams{
			SourceAbs:        dest,
			OutDir:           outDir,
			DBPath:           req.SharedSocialDBPath,
			FileRel:          fileRel,
			HLSRel:           hlsRel,
			DeleteSource:     req.DeleteSource,
			ManualAudioRoles: req.ManualAudioRoles,
		}, job.JobID)
	}
}

// runTranscodeJob exécute le transcoding en arrière-plan (context.Background :
// survit à la requête HTTP), met à jour le job, et notifie la galerie au succès.
func (s *MediaService) runTranscodeJob(p ops.HLSTranscodeParams, jobID string) {
	observability.Heartbeat("media_pipeline") // A6/DC-5 : pipeline media vu vivant
	ctx := context.Background()
	log := slog.With("module", "media") // logs/media.log
	step := "transcoding"
	s.jobStore.SetStatus(jobID, domain.JobStatusRunning, &step)

	if err := ops.RunHLSTranscode(ctx, p); err != nil {
		log.ErrorContext(ctx, "hls: job de transcoding échoué",
			"job", jobID, "file", p.FileRel, "err", err)
		s.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusFailed
			now := time.Now().UTC()
			j.FinishedAt = &now
			j.Error = &domain.JobErrorDetail{Code: "transcode_failed", Message: err.Error()}
		})
		return
	}

	s.jobStore.SetStatus(jobID, domain.JobStatusSucceeded, nil)
	log.InfoContext(ctx, "hls: job de transcoding terminé", "job", jobID, "file", p.HLSRel)
	if s.feedBump != nil {
		s.feedBump()
	}
}
