// Package service — media_index_service.go : réinitialisation de l'index médias.
//
// Sprint 53 Volet A : implémentation réelle de POST /settings/media/reset-index.
// Remplace le stub "Terminé (stub)" de handlers/settings.go.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/jobs"
)

// stepNoPlayerFound est le label de step renvoye quand aucun joueur n'a ete trouve.
const stepNoPlayerFound = "Aucun joueur trouvé"

// MediaIndexer est l'interface d'abstraction pour le reset + reindex des médias.
// Facilite les mocks dans les tests.
type MediaIndexer interface {
	// ResetAndReindex supprime toutes les entrées media_files + media_match_associations
	// pour tous les joueurs sous repoRoot, puis réindexe si reindexAfter=true.
	// capturesBaseDir : si non vide, les captures sont lues depuis {capturesBaseDir}/{gamertag}/
	// plutôt que depuis le chemin interne data/titles/.../captures/.
	// timezone : IANA (ex: "Europe/Paris") — requis pour que parseCaptureTimeFromFilename
	// extraie correctement la datetime depuis les noms de fichier OBS/Xbox (sinon
	// la regex est skip → capture_start_utc=NULL → 0 associations match).
	// Rapporte la progression via jobStore pour le job jobID.
	ResetAndReindex(ctx context.Context, repoRoot string, capturesBaseDir string, timezone string, reindexAfter bool, jobStore *jobs.Store, jobID string) error

	// ScanAllMedia indexe les médias de tous les joueurs sans supprimer les entrées
	// existantes (opération non-destructive). ForceRescan=false : seuls les nouveaux
	// fichiers sont insérés. Cf. ResetAndReindex pour le param timezone.
	ScanAllMedia(ctx context.Context, repoRoot string, capturesBaseDir string, timezone string, jobStore *jobs.Store, jobID string) error
}

// DirMediaIndexer est l'implémentation par défaut de MediaIndexer.
// Parcourt data/players/ et opère sur chaque stats.duckdb trouvé.
type DirMediaIndexer struct{}

// NewDirMediaIndexer crée un DirMediaIndexer.
func NewDirMediaIndexer() *DirMediaIndexer {
	return &DirMediaIndexer{}
}

// ResetAndReindex supprime l'index médias de tous les joueurs, puis le reconstruit.
func (d *DirMediaIndexer) ResetAndReindex(
	ctx context.Context,
	repoRoot string,
	capturesBaseDir string,
	timezone string,
	reindexAfter bool,
	jobStore *jobs.Store,
	jobID string,
) error {
	pr := titlePkg.NewPathResolver(repoRoot)
	titleSlug := ctxkeys.TitleSlug(ctx) // titre courant (sync/admin) ; repli halo_infinite si absent
	capturesBaseDir = effectiveMediaBase(ctx, pr, capturesBaseDir)
	playersDir := pr.PlayersRootDir(titleSlug)

	entries, err := os.ReadDir(playersDir)
	if err != nil {
		// Si le répertoire n'existe pas, il n'y a rien à réinitialiser.
		if os.IsNotExist(err) {
			step := stepNoPlayerFound
			jobStore.SetStatus(jobID, domain.JobStatusSucceeded, &step)
			return nil
		}
		return fmt.Errorf("ResetAndReindex: lecture %s: %w", playersDir, err)
	}

	// Filtrer pour ne garder que les répertoires joueur.
	var playerDirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			playerDirs = append(playerDirs, e)
		}
	}

	total := len(playerDirs)
	if total == 0 {
		step := stepNoPlayerFound
		jobStore.SetStatus(jobID, domain.JobStatusSucceeded, &step)
		return nil
	}

	for i, entry := range playerDirs {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		gamertag := entry.Name()
		dbPath := pr.PlayerDBPath(titleSlug, gamertag)

		// Mettre à jour la progression (0–50% pour reset, 50–100% pour reindex).
		pct := (i * 50) / total
		step := fmt.Sprintf("Reset médias : %s (%d/%d)", gamertag, i+1, total)
		jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.CurrentStep = &step
			j.ProgressPct = &pct
		})

		if err := resetPlayerMediaIndex(ctx, dbPath); err != nil {
			// Avertissement non-fatal : continuer avec les autres joueurs.
			jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
				w := fmt.Sprintf("WARN reset %s: %v", gamertag, err)
				j.Warnings = append(j.Warnings, w)
			})
			continue
		}

		if reindexAfter {
			capturesDir := pr.ResolveCapturesDir(titleSlug, gamertag, capturesBaseDir)
			if _, err := os.Stat(capturesDir); err == nil {
				reindexPct := 50 + (i*50)/total
				reindexStep := fmt.Sprintf("Réindexation : %s (%d/%d)", gamertag, i+1, total)
				jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
					j.CurrentStep = &reindexStep
					j.ProgressPct = &reindexPct
				})

				if _, err := ops.IndexMedia(ctx, ops.MediaIndexOptions{
					PlayerDBPath:        dbPath,
					SharedSocialDBPath:  pr.SharedSocialDBPath(titleSlug),
					SharedMatchesDBPath: pr.SharedDBPath(titleSlug),
					CapturesDir:         capturesDir,
					CapturesBase:        capturesBaseDir,
					ForceRescan:         true,
					Gamertag:            gamertag,
					Timezone:            timezone,
				}); err != nil {
					jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
						w := fmt.Sprintf("WARN reindex %s: %v", gamertag, err)
						j.Warnings = append(j.Warnings, w)
					})
				}
			}
		}
	}

	// Symétrique du transcoding à l'upload : transcoder en HLS, en arrière-plan,
	// les vidéos réindexées encore sans HLS (HEVC/AVI/multipiste). Supprime
	// l'asymétrie upload/scan à l'origine de "media remux failed" sur HEVC.
	if reindexAfter {
		triggerHLSSweep(capturesBaseDir, pr.SharedSocialDBPath(titleSlug), "")
	}
	return nil
}

// ScanAllMedia indexe les médias de tous les joueurs (non-destructif, ForceRescan=false).
func (d *DirMediaIndexer) ScanAllMedia(
	ctx context.Context,
	repoRoot string,
	capturesBaseDir string,
	timezone string,
	jobStore *jobs.Store,
	jobID string,
) error {
	pr := titlePkg.NewPathResolver(repoRoot)
	titleSlug := ctxkeys.TitleSlug(ctx) // titre courant (sync/admin) ; repli halo_infinite si absent
	capturesBaseDir = effectiveMediaBase(ctx, pr, capturesBaseDir)
	playersDir := pr.PlayersRootDir(titleSlug)

	entries, err := os.ReadDir(playersDir)
	if err != nil {
		if os.IsNotExist(err) {
			step := stepNoPlayerFound
			jobStore.SetStatus(jobID, domain.JobStatusSucceeded, &step)
			return nil
		}
		return fmt.Errorf("ScanAllMedia: lecture %s: %w", playersDir, err)
	}

	var playerDirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			playerDirs = append(playerDirs, e)
		}
	}

	total := len(playerDirs)
	if total == 0 {
		step := stepNoPlayerFound
		jobStore.SetStatus(jobID, domain.JobStatusSucceeded, &step)
		return nil
	}

	for i, entry := range playerDirs {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		gamertag := entry.Name()
		dbPath := pr.PlayerDBPath(titleSlug, gamertag)

		capturesDir := pr.ResolveCapturesDir(titleSlug, gamertag, capturesBaseDir)

		if _, err := os.Stat(capturesDir); err != nil {
			continue // dossier absent, on passe
		}

		pct := (i * 100) / total
		step := fmt.Sprintf("Scan médias : %s (%d/%d)", gamertag, i+1, total)
		jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.CurrentStep = &step
			j.ProgressPct = &pct
		})

		if _, err := ops.IndexMedia(ctx, ops.MediaIndexOptions{
			PlayerDBPath:        dbPath,
			SharedSocialDBPath:  pr.SharedSocialDBPath(titleSlug),
			SharedMatchesDBPath: pr.SharedDBPath(titleSlug),
			CapturesDir:         capturesDir,
			CapturesBase:        capturesBaseDir,
			ForceRescan:         false,
			Gamertag:            gamertag,
			Timezone:            timezone,
		}); err != nil {
			jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
				w := fmt.Sprintf("WARN scan %s: %v", gamertag, err)
				j.Warnings = append(j.Warnings, w)
			})
		}
	}

	// Transcoder en HLS, en arrière-plan, les vidéos scannées encore sans HLS
	// (HEVC/AVI/multipiste) — symétrique du transcoding déclenché à l'upload.
	triggerHLSSweep(capturesBaseDir, pr.SharedSocialDBPath(titleSlug), "")
	return nil
}

// BuildMediaScanHook construit la closure `func(ctx)` à injecter dans SyncEngine
// via WithMediaScanHook. Elle scanne le répertoire captures/ du joueur et indexe
// les nouveaux fichiers médias (ForceRescan=false), puis les associe aux matchs.
//
// capturesBaseDirFn et timezoneFn sont des chargeurs paresseux : appelés à chaque
// sync pour lire les settings courants (évite de capturer une valeur périmée au
// boot). timezoneFn est CRITIQUE — sans timezone, parseCaptureTimeFromFilename
// retourne nil systématiquement → capture_start_utc=NULL → 0 associations match.
//
// Usage typique :
//
//	hook := service.BuildMediaScanHook(cfg.RepoRoot, gamertag,
//	    func() string {
//	        s, _ := settingsStore.Load()
//	        if s != nil { return s.MediaCapturesBaseDir }
//	        return ""
//	    },
//	    func() string {
//	        s, _ := settingsStore.Load()
//	        if s != nil { return s.UserTimezone }
//	        return ""
//	    },
//	)
//	engine.WithMediaScanHook(hook)
func BuildMediaScanHook(repoRoot, gamertag string, capturesBaseDirFn, timezoneFn func() string) func(ctx context.Context) {
	return func(ctx context.Context) {
		capturesBaseDir := ""
		if capturesBaseDirFn != nil {
			capturesBaseDir = capturesBaseDirFn()
		}
		timezone := ""
		if timezoneFn != nil {
			timezone = timezoneFn()
		}
		pr := titlePkg.NewPathResolver(repoRoot)
		titleSlug := ctxkeys.TitleSlug(ctx) // titre courant (sync/admin) ; repli halo_infinite si absent
		capturesBaseDir = effectiveMediaBase(ctx, pr, capturesBaseDir)
		capturesDir := pr.ResolveCapturesDir(titleSlug, gamertag, capturesBaseDir)
		if _, err := os.Stat(capturesDir); err != nil {
			return // répertoire absent → rien à indexer
		}
		if _, err := ops.IndexMedia(ctx, ops.MediaIndexOptions{
			PlayerDBPath:        pr.PlayerDBPath(titleSlug, gamertag),
			SharedSocialDBPath:  pr.SharedSocialDBPath(titleSlug),
			SharedMatchesDBPath: pr.SharedDBPath(titleSlug),
			CapturesDir:         capturesDir,
			CapturesBase:        capturesBaseDir,
			ForceRescan:         false,
			Gamertag:            gamertag,
			Timezone:            timezone,
		}); err != nil {
			slog.WarnContext(ctx, "post-sync: media scan échoué (non-fatal)", "gamertag", gamertag, "err", err)
		}
		// Transcoder en HLS, en arrière-plan, les captures fraîchement scannées
		// encore sans HLS (HEVC/AVI/multipiste) — comme à l'upload.
		triggerHLSSweep(capturesBaseDir, pr.SharedSocialDBPath(titleSlug), gamertag)
	}
}

// effectiveMediaBase résout la base média effective pour l'indexation/scan/HLS :
// retourne `configured` (settings.MediaCapturesBaseDir) s'il existe sur disque ;
// sinon retombe sur la base interne canonique pr.MediaDataDir() ({root}/data/media)
// si elle existe ; sinon "" (comportement interne PlayerCapturesDir inchangé).
// Logge un WARN quand le fallback s'active.
//
// Garde-fou anti-régression (incident prod 2026-06-13) : un media_captures_base_dir
// invalide — ex: chemin Windows recopié sur le VPS Linux — ne doit plus rendre
// scan/HLS/thumbnails inopérants ni rediriger les écritures vers un dossier fantôme.
func effectiveMediaBase(ctx context.Context, pr *titlePkg.PathResolver, configured string) string {
	if configured == "" {
		return ""
	}
	if fi, err := os.Stat(configured); err == nil && fi.IsDir() {
		return configured
	}
	fallback := pr.MediaDataDir()
	if fi, err := os.Stat(fallback); err == nil && fi.IsDir() {
		slog.WarnContext(ctx, "media: media_captures_base_dir introuvable, fallback data/media",
			"configured", configured, "effective", fallback)
		return fallback
	}
	return ""
}

// triggerHLSSweep lance EN ARRIÈRE-PLAN (détaché du ctx scan/sync, qui peut être
// annulé bien avant la fin d'un transcodage) un balayage EnsurePendingHLS : il
// transcode en HLS les vidéos scannées encore sans HLS, exactement comme le fait
// l'upload. C'est ce qui supprime l'asymétrie upload/scan responsable de l'erreur
// "media remux failed" sur les captures HEVC. No-op si capturesBaseDir est vide
// (mode legacy interne : HLSPathsFor exige une base multi-player). Le single-flight
// est géré dans EnsurePendingHLS (un seul balayage à la fois dans le process).
func triggerHLSSweep(capturesBaseDir, sharedSocialDBPath, onlySlug string) {
	if capturesBaseDir == "" || sharedSocialDBPath == "" {
		return
	}
	go func() {
		ctx := context.Background()
		log := slog.With("module", "media") // logs/media.log
		st, err := ops.EnsurePendingHLS(ctx, ops.EnsureHLSParams{
			DBPath:       sharedSocialDBPath,
			CapturesBase: capturesBaseDir,
			OnlySlug:     onlySlug,
		})
		if err != nil {
			log.ErrorContext(ctx, "post-scan hls sweep échoué", "err", err)
			return
		}
		if st.Transcoded > 0 || st.Failed > 0 {
			log.InfoContext(ctx, "post-scan hls sweep",
				"transcoded", st.Transcoded, "failed", st.Failed, "skipped_direct", st.SkippedDirect)
		}
	}()
}

// resetPlayerMediaIndex supprime toutes les entrées media_files et media_match_associations
// dans la player DB.
//
// Sprint B1 commit 14a : acquiert dblease.KindPlayer avant les DELETE pour
// se sérialiser avec auto_sync.RunDelta qui peut écrire sur ces mêmes tables
// (post-sync media reindex inline). Sans ce lease, race condition possible
// entre le DELETE et un INSERT/UPDATE sync — données incohérentes.
func resetPlayerMediaIndex(ctx context.Context, dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil // pas de DB joueur, rien à réinitialiser
	}

	lease, err := dblease.AcquireWriterCtx(ctx, nil, dbPath, dblease.KindPlayer)
	if err != nil {
		return fmt.Errorf("resetPlayerMediaIndex lease %s: %w", dbPath, err)
	}
	defer lease.Release()

	// Phase 2 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 (audit residuel 2026-05-25) :
	// passage par le cache duckdbpkg.OpenReadWrite pour DSN aligne avec sync.
	// Sans ce fix, resetPlayerMediaIndex peut entrer en conflit "different
	// configuration" avec un sync delta concurrent sur le meme player DB.
	handle, err := duckdbpkg.OpenReadWrite(dbPath)
	if err != nil {
		return fmt.Errorf("ouverture %s: %w", dbPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// APPEND-ONLY / topologie migrée : plus aucun DELETE. media_files +
	// media_match_associations ont été déplacées de la player DB vers shared_social
	// (migration drop_media_from_player_db) → ces DELETE frappaient des tables absentes
	// (chemin mort, WARN non-fatal). Les associations shared_social sont désormais
	// append-only ; le reindex (IndexMedia) est ADDITIF et idempotent (loadUnassociatedMedia
	// forward-only), aucun reset destructif n'est nécessaire.
	_ = db
	return nil
}
