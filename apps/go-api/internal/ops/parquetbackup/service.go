package parquetbackup

import (
	"context"
	"errors"
	"expvar"
	"log/slog"
	"sync"
	"time"
)

// Compteurs expvar (/debug/vars) : issues des passages, durée et poids du dernier jeu.
var (
	// metricRuns compte les passages par issue : ok, partial (jeu partiel), failed.
	metricRuns = expvar.NewMap("levelup.backup.parquet")
	// metricLastDurationMs : durée du dernier jeu écrit (ms).
	metricLastDurationMs = expvar.NewInt("levelup.backup.parquet_last_duration_ms")
	// metricLastBytes : poids du dernier jeu écrit (octets).
	metricLastBytes = expvar.NewInt("levelup.backup.parquet_last_bytes")
)

// Codes fixes de l'issue d'un passage, rendus à l'API (le texte de l'erreur va au journal).
const (
	CodeNothingExported = "nothing_exported"
	CodeFailed          = "backup_failed"
)

// ErrAlreadyRunning : un jeu est déjà en cours d'écriture.
var ErrAlreadyRunning = errors.New("parquetbackup: une sauvegarde est déjà en cours")

// ErrDisabled : aucun dossier de sauvegarde n'est choisi.
var ErrDisabled = errors.New("parquetbackup: aucun dossier de sauvegarde choisi")

// Summary : un jeu complet, tel que l'état le montre.
type Summary struct {
	Name       string
	CreatedAt  time.Time
	Bytes      int64
	Databases  int
	DurationMs int64
	Partial    bool
	AppVersion string
}

// Attempt : le dernier passage de ce processus.
type Attempt struct {
	At   time.Time
	OK   bool
	Code string // code fixe de l'échec, vide si OK
}

// Status : l'état de la sauvegarde pour la page des réglages.
type Status struct {
	Running bool
	// Last : le jeu complet le plus récent du dossier choisi (nil : aucun).
	Last *Summary
	// LastAttempt : le dernier passage de ce processus (nil : aucun depuis le démarrage).
	LastAttempt *Attempt
}

// Service porte l'état des passages : un seul à la fois, la dernière tentative, l'état lu dans le
// dossier choisi. Un *Service n'écrit jamais lui-même : il appelle [Run].
type Service struct {
	// options rend les options courantes (réglages relus à chaque passage) ou [ErrDisabled].
	options func() (Options, error)
	// runCtx et wg : le contexte et le groupe d'attente du serveur, pour les passages lancés à la demande.
	runCtx context.Context
	wg     *sync.WaitGroup

	mu      sync.Mutex // tient le passage en cours
	state   sync.Mutex // protège last
	last    *Attempt
	running bool
}

// NewService crée un service dont options relit les réglages à chaque appel (tests ; le serveur utilise
// [Default] et [Service.Bind]).
func NewService(options func() (Options, error)) *Service {
	return &Service{options: options}
}

// RunNow écrit un jeu maintenant. [ErrAlreadyRunning] si un passage est en cours, [ErrDisabled] sans
// dossier choisi. Le résultat est noté pour [Service.Status] et compté (expvar).
func (s *Service) RunNow(ctx context.Context) (*Manifest, error) {
	if !s.mu.TryLock() {
		return nil, ErrAlreadyRunning
	}
	defer s.mu.Unlock()
	s.state.Lock()
	options := s.options
	s.state.Unlock()
	if options == nil {
		return nil, ErrNotBound
	}
	opts, err := options()
	if err != nil {
		return nil, err
	}
	s.setRunning(true)
	defer s.setRunning(false)
	m, err := Run(ctx, opts)
	s.note(ctx, m, err)
	return m, err
}

func (s *Service) setRunning(v bool) {
	s.state.Lock()
	s.running = v
	s.state.Unlock()
}

// note enregistre l'issue d'un passage : tentative, compteurs, journal.
func (s *Service) note(ctx context.Context, m *Manifest, err error) {
	a := &Attempt{At: time.Now(), OK: err == nil}
	switch {
	case err == nil:
		issue := "ok"
		if m.Partial {
			issue = "partial"
		}
		metricRuns.Add(issue, 1)
		metricLastDurationMs.Set(m.DurationMs)
		metricLastBytes.Set(m.TotalBytes)
	case errors.Is(err, ErrNothingExported):
		a.Code = CodeNothingExported
		metricRuns.Add("failed", 1)
		slog.ErrorContext(ctx, "backup parquet: passage échoué", "err", err)
	default:
		a.Code = CodeFailed
		metricRuns.Add("failed", 1)
		slog.ErrorContext(ctx, "backup parquet: passage échoué", "err", err)
	}
	s.state.Lock()
	s.last = a
	s.state.Unlock()
}

// Status rend l'état : passage en cours, dernier jeu complet de dest (lu dans le dossier, il survit au
// redémarrage) et dernière tentative de ce processus. dest vide : seulement la tentative.
func (s *Service) Status(dest string) Status {
	s.state.Lock()
	st := Status{Running: s.running}
	if s.last != nil {
		cp := *s.last
		st.LastAttempt = &cp
	}
	s.state.Unlock()
	if dest != "" {
		st.Last = latestSummary(dest)
	}
	return st
}

// Due dit si un passage est dû : jamais de jeu ni de tentative, ou le dernier plus vieux que interval.
// Une tentative échouée repart au bout de retryAfter (au plus interval) plutôt qu'au bout d'un intervalle plein.
func (s *Service) Due(dest string, interval, retryAfter time.Duration, now time.Time) bool {
	st := s.Status(dest)
	if st.Running {
		return false
	}
	var ref time.Time
	if st.Last != nil {
		ref = st.Last.CreatedAt
	}
	if a := st.LastAttempt; a != nil && a.At.After(ref) {
		ref = a.At
		if !a.OK && retryAfter < interval {
			return now.Sub(ref) >= retryAfter
		}
	}
	return ref.IsZero() || now.Sub(ref) >= interval
}

// latestSummary lit le manifeste du jeu complet le plus récent de dest.
func latestSummary(dest string) *Summary {
	sets, err := SetDirs(dest)
	if err != nil {
		return nil
	}
	for _, dir := range sets { // le plus récent dont le manifeste se lit
		m, err := LoadManifest(dir)
		if err != nil {
			slog.Warn("backup parquet: manifeste illisible", "set", dir, "err", err)
			continue
		}
		return &Summary{Name: m.Name, CreatedAt: m.CreatedAt, Bytes: m.TotalBytes, Databases: len(m.Databases),
			DurationMs: m.DurationMs, Partial: m.Partial, AppVersion: m.AppVersion}
	}
	return nil
}

// ErrNotBound : le service n'a pas été relié aux réglages et au contexte du serveur (profil sans la
// capacité de sauvegarde, tests).
var ErrNotBound = errors.New("parquetbackup: service non relié")

var defaultService = &Service{}

// Default rend le service du processus : le même pour la boucle périodique du serveur et pour la route
// « sauvegarder maintenant », comme gameproc.Default.
func Default() *Service { return defaultService }

// Bind relie le service aux réglages vivants (options relit app_settings.json à chaque passage) et au
// contexte du serveur : ctx annule un passage en cours à l'arrêt, wg le fait attendre avant la fermeture des
// bases (le scheduler du serveur).
func (s *Service) Bind(ctx context.Context, wg *sync.WaitGroup, options func() (Options, error)) {
	s.state.Lock()
	defer s.state.Unlock()
	s.options, s.runCtx, s.wg = options, ctx, wg
}

// Start lance un passage en arrière-plan, sous le contexte relié par [Service.Bind]. [ErrNotBound] sans
// liaison, [ErrAlreadyRunning] si un passage est en cours, [ErrDisabled] sans dossier choisi (relu ici pour
// que la route réponde tout de suite).
func (s *Service) Start() error {
	s.state.Lock()
	options, ctx, wg, running := s.options, s.runCtx, s.wg, s.running
	s.state.Unlock()
	if options == nil || ctx == nil {
		return ErrNotBound
	}
	if running {
		return ErrAlreadyRunning
	}
	if _, err := options(); err != nil {
		return err
	}
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		if _, err := s.RunNow(ctx); err != nil && !errors.Is(err, ErrAlreadyRunning) {
			slog.WarnContext(ctx, "backup parquet: passage lancé à la demande échoué", "err", err)
		}
	}()
	return nil
}
