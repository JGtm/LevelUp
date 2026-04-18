// Package service — media_index_service.go : réinitialisation de l'index médias.
//
// Sprint 53 Volet A : implémentation réelle de POST /settings/media/reset-index.
// Remplace le stub "Terminé (stub)" de handlers/settings.go.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/jobs"
)

// MediaIndexer est l'interface d'abstraction pour le reset + reindex des médias.
// Facilite les mocks dans les tests.
type MediaIndexer interface {
	// ResetAndReindex supprime toutes les entrées media_files + media_match_associations
	// pour tous les joueurs sous repoRoot, puis réindexe si reindexAfter=true.
	// Rapporte la progression via jobStore pour le job jobID.
	ResetAndReindex(ctx context.Context, repoRoot string, reindexAfter bool, jobStore *jobs.Store, jobID string) error
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
	reindexAfter bool,
	jobStore *jobs.Store,
	jobID string,
) error {
	playersDir := filepath.Join(repoRoot, "data", "players")

	entries, err := os.ReadDir(playersDir)
	if err != nil {
		// Si le répertoire n'existe pas, il n'y a rien à réinitialiser.
		if os.IsNotExist(err) {
			step := "Aucun joueur trouvé"
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
		step := "Aucun joueur trouvé"
		jobStore.SetStatus(jobID, domain.JobStatusSucceeded, &step)
		return nil
	}

	for i, entry := range playerDirs {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		gamertag := entry.Name()
		dbPath := filepath.Join(playersDir, gamertag, "stats.duckdb")

		// Mettre à jour la progression (0–50% pour reset, 50–100% pour reindex).
		pct := (i * 50) / total
		step := fmt.Sprintf("Reset médias : %s (%d/%d)", gamertag, i+1, total)
		jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.CurrentStep = &step
			j.ProgressPct = &pct
		})

		if err := resetPlayerMediaIndex(dbPath); err != nil {
			// Avertissement non-fatal : continuer avec les autres joueurs.
			jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
				w := fmt.Sprintf("WARN reset %s: %v", gamertag, err)
				j.Warnings = append(j.Warnings, w)
			})
			continue
		}

		if reindexAfter {
			capturesDir := filepath.Join(playersDir, gamertag, "media")
			if _, err := os.Stat(capturesDir); err == nil {
				reindexPct := 50 + (i*50)/total
				reindexStep := fmt.Sprintf("Réindexation : %s (%d/%d)", gamertag, i+1, total)
				jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
					j.CurrentStep = &reindexStep
					j.ProgressPct = &reindexPct
				})

				if _, err := ops.IndexMedia(ops.MediaIndexOptions{
					PlayerDBPath: dbPath,
					CapturesDir:  capturesDir,
					ForceRescan:  true,
					Gamertag:     gamertag,
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

// resetPlayerMediaIndex supprime toutes les entrées media_files et media_match_associations
// dans la player DB.
func resetPlayerMediaIndex(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil // pas de DB joueur, rien à réinitialiser
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("ouverture %s: %w", dbPath, err)
	}
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM media_match_associations`); err != nil {
		return fmt.Errorf("DELETE media_match_associations: %w", err)
	}
	if _, err := db.Exec(`DELETE FROM media_files`); err != nil {
		return fmt.Errorf("DELETE media_files: %w", err)
	}
	return nil
}
