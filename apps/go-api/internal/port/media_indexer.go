package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// MediaJobProgress : les deux opérations du magasin de jobs que l'indexeur des médias appelle pour
// rapporter sa progression. Implémenté par platform/jobs.Store.
type MediaJobProgress interface {
	// Update modifie le job en place ; faux si le job n'existe pas.
	Update(jobID string, fn func(*domain.AsyncJobStatus)) bool
	// SetStatus pose le statut du job et, si step n'est pas nil, son étape courante ; faux si le job n'existe pas.
	SetStatus(jobID string, status domain.JobStatus, step *string) bool
}

// MediaIndexParams : ce qu'une indexation de tous les joueurs reçoit de l'appelant.
type MediaIndexParams struct {
	// RepoRoot : racine du dépôt sous laquelle vivent les joueurs.
	RepoRoot string
	// CapturesBaseDir : si non vide, les captures sont lues depuis {CapturesBaseDir}/{gamertag}/
	// plutôt que depuis le chemin interne data/titles/.../captures/.
	CapturesBaseDir string
	// Timezone : IANA (ex: "Europe/Paris") — requis pour que captureInstantFromName extraie
	// correctement la datetime depuis les noms de fichier OBS/Xbox (sinon la regex est skip →
	// capture_start_utc=NULL → 0 associations match).
	Timezone string
	// DeleteSource : politique de rétention du source après transcodage HLS (résolue par l'appelant via
	// config.AppConfig.MediaDeleteSource), propagée au sweep post-reindex.
	DeleteSource bool
	// Jobs et JobID : le job dont l'indexation rapporte la progression.
	Jobs  MediaJobProgress
	JobID string
}

// MediaResetParams : les paramètres d'une réinitialisation de l'index.
type MediaResetParams struct {
	MediaIndexParams
	// ReindexAfter : réindexer après la réinitialisation.
	ReindexAfter bool
}

// MediaIndexer reset et réindexe les médias de tous les joueurs ; abstraction qui permet de
// substituer l'indexeur dans les tests. Implémenté par service.DirMediaIndexer.
type MediaIndexer interface {
	// ResetAndReindex supprime toutes les entrées media_files + media_match_associations pour tous les
	// joueurs sous RepoRoot, puis réindexe si ReindexAfter. Rapporte la progression sur le job de p.
	ResetAndReindex(ctx context.Context, p MediaResetParams) error

	// ScanAllMedia indexe les médias de tous les joueurs sans supprimer les entrées existantes
	// (opération non-destructive ; seuls les nouveaux fichiers sont insérés). Cf. MediaIndexParams pour
	// les paramètres.
	ScanAllMedia(ctx context.Context, p MediaIndexParams) error
}
