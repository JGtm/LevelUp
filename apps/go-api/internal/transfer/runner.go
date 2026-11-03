package transfer

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Phase : l'étape d'un import depuis le site.
type Phase string

// Les phases d'un import.
const (
	PhaseIdle        Phase = "idle"
	PhaseDownloading Phase = "downloading"
	PhaseImporting   Phase = "importing"
	PhaseDone        Phase = "done"
	PhaseFailed      Phase = "failed"
)

// Codes fixes de l'échec d'un import, rendus à l'API (le texte de l'erreur va au journal).
const (
	CodeBadURL          = "site_url_invalid"
	CodeTokenInvalid    = "site_token_invalid"
	CodeRateLimited     = "site_rate_limited"
	CodeSiteRefused     = "site_refused"
	CodeSiteTooBig      = "site_archive_too_big"
	CodeSiteUnreachable = "site_unreachable"
	CodeSiteTLS         = "site_tls"
	CodeSiteNotFound    = "site_not_found"
	CodeBadArchive      = "archive_invalid"
	CodeXuidMismatch    = "archive_other_player"
	CodeUnknownTable    = "archive_unknown_table"
	CodeFormatTooNewer  = "archive_format_unsupported"
	CodeFailed          = "import_failed"
)

// ErrImportRunning : un import est déjà en cours.
var ErrImportRunning = errors.New("transfer: un import est déjà en cours")

// CodeOf traduit l'erreur d'un import en son code fixe.
func CodeOf(err error) string {
	switch {
	case errors.Is(err, ErrBadURL):
		return CodeBadURL
	case errors.Is(err, ErrTokenInvalid):
		return CodeTokenInvalid
	case errors.Is(err, ErrRateLimited):
		return CodeRateLimited
	case errors.Is(err, ErrSiteRefused):
		return CodeSiteRefused
	case errors.Is(err, ErrSiteUnreachable):
		return CodeSiteUnreachable
	case errors.Is(err, ErrSiteTLS):
		return CodeSiteTLS
	case errors.Is(err, ErrSiteNotFound):
		return CodeSiteNotFound
	case errors.Is(err, ErrSiteTooBig), errors.Is(err, ErrTooLarge):
		return CodeSiteTooBig
	case errors.Is(err, ErrXuidMismatch):
		return CodeXuidMismatch
	case errors.Is(err, ErrUnknownTable):
		return CodeUnknownTable
	case errors.Is(err, ErrFormatTooNewer):
		return CodeFormatTooNewer
	case errors.Is(err, ErrBadArchive), errors.Is(err, ErrInvalidXuid):
		return CodeBadArchive
	}
	return CodeFailed
}

// Status : l'état de l'import courant ou du dernier.
type Status struct {
	Phase     Phase
	ErrorCode string
	// Inserted : lignes insérées par table ("shared/match_registry"), une fois l'import fini.
	Inserted  map[string]int64
	UpdatedAt time.Time
}

// RunInput : un import depuis le site, du téléchargement à l'écriture.
type RunInput struct {
	Fetch  FetchInput
	Import ImportInput
	// WorkDir : où l'archive est téléchargée (vide = dossier temporaire du système).
	WorkDir string
	// AfterImport : appelé une fois l'import ÉCRIT (pas sur un échec), avant que l'état passe à « terminé » : c'est là que
	// l'application invalide ses caches de lecture, périmés par les lignes importées. Nil : rien.
	AfterImport func(ctx context.Context)
}

// Runner exécute un import à la fois et en tient l'état. Un *Runner sert une route de démarrage et une route
// d'état.
type Runner struct {
	mu sync.Mutex
	st Status
}

// NewRunner crée un exécuteur au repos.
func NewRunner() *Runner { return &Runner{st: Status{Phase: PhaseIdle}} }

// Status rend l'état courant (copie).
func (r *Runner) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := r.st
	if r.st.Inserted != nil {
		cp.Inserted = make(map[string]int64, len(r.st.Inserted))
		for k, v := range r.st.Inserted {
			cp.Inserted[k] = v
		}
	}
	return cp
}

func (r *Runner) set(p Phase, code string, inserted map[string]int64) {
	r.mu.Lock()
	r.st = Status{Phase: p, ErrorCode: code, Inserted: inserted, UpdatedAt: time.Now()}
	r.mu.Unlock()
}

// Start lance l'import en arrière-plan sous ctx (annulé à l'arrêt du serveur) et wg (attendu avant la
// fermeture des bases). [ErrImportRunning] si un import est en cours.
func (r *Runner) Start(ctx context.Context, wg *sync.WaitGroup, in RunInput) error {
	r.mu.Lock()
	if r.st.Phase == PhaseDownloading || r.st.Phase == PhaseImporting {
		r.mu.Unlock()
		return ErrImportRunning
	}
	r.st = Status{Phase: PhaseDownloading, UpdatedAt: time.Now()}
	r.mu.Unlock()
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		r.run(ctx, in)
	}()
	return nil
}

// run enchaîne téléchargement et import ; l'archive téléchargée est supprimée à la fin, réussie ou non.
func (r *Runner) run(ctx context.Context, in RunInput) {
	dir, err := os.MkdirTemp(in.WorkDir, "levelup-site-import-")
	if err != nil {
		slog.ErrorContext(ctx, "import depuis le site: dossier de travail impossible", "err", err)
		r.set(PhaseFailed, CodeFailed, nil)
		return
	}
	defer func() { _ = os.RemoveAll(dir) }()
	zipPath := filepath.Join(dir, "export.zip")
	in.Fetch.Dest, in.Import.ZipPath, in.Import.WorkDir = zipPath, zipPath, dir
	if err := Fetch(ctx, in.Fetch); err != nil {
		slog.WarnContext(ctx, "import depuis le site: téléchargement échoué", "err", err)
		r.set(PhaseFailed, CodeOf(err), nil)
		return
	}
	r.set(PhaseImporting, "", nil)
	res, err := Import(ctx, in.Import)
	if err != nil {
		slog.WarnContext(ctx, "import depuis le site: import échoué", "err", err)
		r.set(PhaseFailed, CodeOf(err), nil)
		return
	}
	if in.AfterImport != nil {
		in.AfterImport(ctx)
	}
	r.set(PhaseDone, "", res.Inserted)
}
