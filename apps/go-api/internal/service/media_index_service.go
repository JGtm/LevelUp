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
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

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
	titleSlug := titlePkg.DefaultSlug
	playersDir := filepath.Join(pr.TitleDataDir(titleSlug), "players")

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
	titleSlug := titlePkg.DefaultSlug
	playersDir := filepath.Join(pr.TitleDataDir(titleSlug), "players")

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
		titleSlug := titlePkg.DefaultSlug
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
	}
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

	if _, err := db.ExecContext(ctx, `DELETE FROM media_match_associations`); err != nil {
		return fmt.Errorf("DELETE media_match_associations: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM media_files`); err != nil {
		return fmt.Errorf("DELETE media_files: %w", err)
	}
	return nil
}
