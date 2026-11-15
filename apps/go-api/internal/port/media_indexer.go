package port

import (
	"context"

	"levelup/go-api/internal/platform/jobs"
)

// MediaIndexer reset et réindexe les médias de tous les joueurs ; abstraction qui permet de
// substituer l'indexeur dans les tests. Implémenté par service.DirMediaIndexer.
type MediaIndexer interface {
	// ResetAndReindex supprime toutes les entrées media_files + media_match_associations
	// pour tous les joueurs sous repoRoot, puis réindexe si reindexAfter=true.
	// capturesBaseDir : si non vide, les captures sont lues depuis {capturesBaseDir}/{gamertag}/
	// plutôt que depuis le chemin interne data/titles/.../captures/.
	// timezone : IANA (ex: "Europe/Paris") — requis pour que captureInstantFromName
	// extraie correctement la datetime depuis les noms de fichier OBS/Xbox (sinon
	// la regex est skip → capture_start_utc=NULL → 0 associations match).
	// Rapporte la progression via jobStore pour le job jobID.
	// deleteSource : politique de rétention du source après transcodage HLS (résolue
	// par l'appelant via config.AppConfig.MediaDeleteSource), propagée au sweep post-reindex.
	ResetAndReindex(ctx context.Context, repoRoot string, capturesBaseDir string, timezone string, reindexAfter bool, deleteSource bool, jobStore *jobs.Store, jobID string) error

	// ScanAllMedia indexe les médias de tous les joueurs sans supprimer les entrées
	// existantes (opération non-destructive). ForceRescan=false : seuls les nouveaux
	// fichiers sont insérés. Cf. ResetAndReindex pour les params timezone/deleteSource.
	ScanAllMedia(ctx context.Context, repoRoot string, capturesBaseDir string, timezone string, deleteSource bool, jobStore *jobs.Store, jobID string) error
}
