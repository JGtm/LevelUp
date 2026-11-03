package replayregen

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/LevelUp/libs/film/finalise"
)

// Compteurs expvar de la file.
const (
	MetricStarted      = "replay_regen_started_total"
	MetricDone         = "replay_regen_done_total"
	MetricSkipped      = "replay_regen_skipped_total"
	MetricFailed       = "replay_regen_failed_total"
	MetricDeferredGame = "replay_regen_deferred_game_total"
	// MetricStale : jauge, rejeux en retard au dernier décompte.
	MetricStale = "replay_regen_stale"
)

// Erreurs de la file.
var (
	// ErrNotBound : le service n'est pas relié au serveur (capacité absente, ou démarrage inachevé).
	ErrNotBound = errors.New("replayregen: service non relié")
	// ErrNotRunning : pause ou reprise demandée sans file active.
	ErrNotRunning = errors.New("replayregen: aucune file active")
	// ErrBusy : la reconstruction n'a pas pu démarrer parce qu'un autre décodage tient la machine ; la file
	// attend et recommence le même film. Enveloppée par la fonction de reconstruction.
	ErrBusy = errors.New("replayregen: un décodage de film est déjà en cours")
)

// PendingStore : le drapeau persisté « la file était active » (réglages, `replay_regen_pending`).
type PendingStore interface {
	ReplayRegenPending() (bool, error)
	SetReplayRegenPending(pending bool) error
}

// Deps : tout ce que la file demande au serveur.
type Deps struct {
	Paths *title.PathResolver
	// Titles : les titres dont les rejeux se régénèrent (ceux qui déclarent la capacité de rejeu).
	Titles func() []string
	// FilmPresent : le film du match est dans l'archive locale ; sans lui il n'y a rien à rejouer.
	FilmPresent func(titleSlug, matchID string) bool
	// Rebuild reconstruit et range le rejeu d'UN match, hors du processus (même chemin que l'action admin). Il
	// enveloppe ErrBusy quand un autre décodage tient la machine.
	Rebuild func(ctx context.Context, titleSlug, matchID string) error
	Pending PendingStore
	// HeavyAllowed : la porte du mode jeu (gameproc.Mode.HeavyWorkAllowed), consultée avant chaque film.
	HeavyAllowed func() bool
	// Rythmes ; zéro = valeur de production.
	GateEvery      time.Duration
	BusyRetryEvery time.Duration
	BusyRetries    int
	RescanEvery    time.Duration
}

const (
	defaultGateEvery      = 15 * time.Second
	defaultBusyRetryEvery = 20 * time.Second
	defaultBusyRetries    = 30
	defaultRescanEvery    = 10 * time.Minute
)

func orDefault(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}

func (d Deps) gateEvery() time.Duration   { return orDefault(d.GateEvery, defaultGateEvery) }
func (d Deps) busyEvery() time.Duration   { return orDefault(d.BusyRetryEvery, defaultBusyRetryEvery) }
func (d Deps) rescanEvery() time.Duration { return orDefault(d.RescanEvery, defaultRescanEvery) }
func (d Deps) busyRetries() int {
	if d.BusyRetries > 0 {
		return d.BusyRetries
	}
	return defaultBusyRetries
}

// Service : le décompte et la file. Un seul film à la fois, jamais deux en parallèle.
type Service struct {
	mu   sync.Mutex
	deps Deps
	ctx  context.Context
	wg   *sync.WaitGroup
	// bound : Bind a été appelé.
	bound bool

	// Décompte (hors file active).
	checked  bool
	stale    int
	scanAt   time.Time
	scanning bool
	blocked  map[string]struct{}

	// File.
	running     bool
	userPaused  bool
	waitingGame bool
	done        int
	total       int
	failed      int
	wake        chan struct{}
}

// New crée un service non relié.
func New() *Service { return &Service{blocked: map[string]struct{}{}, wake: make(chan struct{}, 1)} }

var defaultService = New()

// Default rend le service du processus : le même pour les routes et pour le démarrage.
func Default() *Service { return defaultService }

// Bind relie le service au serveur : ctx annule la file à l'arrêt, wg la fait attendre avant la fermeture des bases.
func (s *Service) Bind(ctx context.Context, wg *sync.WaitGroup, deps Deps) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx, s.wg, s.deps, s.bound = ctx, wg, deps, true
}

// Status rend l'état courant. Hors file, le décompte vient du cache ; un décompte absent ou périmé se relance en
// arrière-plan (jamais dans la requête : il lit chaque rejeu rangé).
func (s *Service) Status() domain.ReplayRegenStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.statusLocked()
	if s.bound && !s.running && !s.scanning && (!s.checked || time.Since(s.scanAt) > s.deps.rescanEvery()) {
		s.scanning = true
		s.wg.Add(1)
		go s.backgroundScan()
	}
	return st
}

func (s *Service) statusLocked() domain.ReplayRegenStatus {
	stale := s.stale
	if s.running {
		stale = max(s.total-s.done, 0)
	}
	return domain.ReplayRegenStatus{
		Checked: s.checked || s.running, Stale: stale, Running: s.running, Done: s.done, Total: s.total,
		Paused: s.running && s.userPaused, WaitingForGame: s.running && s.waitingGame, Failed: s.failed,
	}
}

func (s *Service) backgroundScan() {
	defer s.wg.Done()
	cands, err := s.scan(s.ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scanning = false
	if err != nil {
		slog.WarnContext(s.ctx, "replay_regen: décompte des rejeux en retard interrompu", "err", err)
		return
	}
	s.setCountLocked(len(cands))
}

func (s *Service) setCountLocked(n int) {
	s.checked, s.stale, s.scanAt = true, n, time.Now()
	observability.SetInt(MetricStale, int64(n))
}

// scan liste les rejeux régénérables de tous les titres, le plus récent d'abord.
func (s *Service) scan(ctx context.Context) ([]Candidate, error) {
	s.mu.Lock()
	d := s.deps
	s.mu.Unlock()
	var all []Candidate
	for _, slug := range d.Titles() {
		c, err := scanTitle(ctx, d.Paths.ReplayArtifactsDir(slug), slug, d.FilmPresent, s.isBlocked)
		if err != nil {
			return nil, err
		}
		all = append(all, c...)
	}
	sortRecentFirst(all)
	return all, nil
}

func blockKey(slug, matchID string) string { return slug + "/" + matchID }

func (s *Service) isBlocked(slug, matchID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.blocked[blockKey(slug, matchID)]
	return ok
}

func (s *Service) block(slug, matchID string) {
	s.mu.Lock()
	s.blocked[blockKey(slug, matchID)] = struct{}{}
	s.mu.Unlock()
}

// Start lance la file à la demande de l'utilisateur. Idempotent : une file déjà active n'est pas relancée.
func (s *Service) Start() (domain.ReplayRegenStatus, error) {
	return s.start(false)
}

// ResumeIfPending reprend, à l'ouverture de l'application, la file que l'utilisateur avait lancée et qu'un arrêt
// a interrompue (drapeau persisté). Sans drapeau, ne fait rien : la file ne part jamais seule.
func (s *Service) ResumeIfPending() {
	s.mu.Lock()
	bound, pending, ctx := s.bound, s.deps.Pending, s.ctx
	s.mu.Unlock()
	if !bound || pending == nil {
		return
	}
	on, err := pending.ReplayRegenPending()
	if err != nil {
		slog.WarnContext(ctx, "replay_regen: drapeau de reprise illisible, la file ne reprend pas", "err", err)
		return
	}
	if !on {
		return
	}
	slog.InfoContext(ctx, "replay_regen: reprise de la file lancée avant l'arrêt")
	if _, err := s.start(true); err != nil {
		slog.WarnContext(ctx, "replay_regen: reprise impossible", "err", err)
	}
}

func (s *Service) start(resumed bool) (domain.ReplayRegenStatus, error) {
	s.mu.Lock()
	if !s.bound {
		s.mu.Unlock()
		return domain.ReplayRegenStatus{}, ErrNotBound
	}
	if s.running {
		st := s.statusLocked()
		s.mu.Unlock()
		return st, nil
	}
	s.running, s.userPaused, s.waitingGame = true, false, false
	s.done, s.total, s.failed = 0, 0, 0
	st, ctx := s.statusLocked(), s.ctx
	s.wg.Add(1)
	s.mu.Unlock()

	observability.IncCounter(MetricStarted)
	if !resumed {
		s.persistPending(ctx, true)
	}
	go s.run(ctx)
	return st, nil
}

// Pause met la file en pause à la demande de l'utilisateur : le film en cours va à son terme, le suivant ne part
// pas. Une file en pause ne reprend pas toute seule à l'ouverture suivante (drapeau effacé).
func (s *Service) Pause() (domain.ReplayRegenStatus, error) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return domain.ReplayRegenStatus{}, ErrNotRunning
	}
	s.userPaused = true
	st, ctx := s.statusLocked(), s.ctx
	s.mu.Unlock()
	s.persistPending(ctx, false)
	return st, nil
}

// Resume relance une file en pause.
func (s *Service) Resume() (domain.ReplayRegenStatus, error) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return domain.ReplayRegenStatus{}, ErrNotRunning
	}
	s.userPaused = false
	st, ctx := s.statusLocked(), s.ctx
	s.mu.Unlock()
	s.persistPending(ctx, true)
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return st, nil
}

func (s *Service) persistPending(ctx context.Context, on bool) {
	s.mu.Lock()
	p := s.deps.Pending
	s.mu.Unlock()
	if p == nil {
		return
	}
	if err := p.SetReplayRegenPending(on); err != nil {
		slog.WarnContext(ctx, "replay_regen: drapeau de reprise non écrit", "pending", on, "err", err)
	}
}

// run : la file. Prépare la liste, puis traite un film à la fois.
func (s *Service) run(ctx context.Context) {
	defer s.wg.Done()
	cands, err := s.scan(ctx)
	if err != nil {
		slog.WarnContext(ctx, "replay_regen: liste des rejeux impossible, file arrêtée", "err", err)
		s.finish(ctx, ctx.Err() != nil)
		return
	}
	s.mu.Lock()
	s.total = len(cands)
	s.mu.Unlock()
	slog.InfoContext(ctx, "replay_regen: file démarrée", "total", len(cands))
	for _, c := range cands {
		if !s.waitTurn(ctx) {
			s.finish(ctx, true)
			return
		}
		s.process(ctx, c)
		if ctx.Err() != nil {
			s.finish(ctx, true)
			return
		}
	}
	s.finish(ctx, false)
}

// finish clôt la file. interrupted : l'arrêt du serveur l'a coupée, le drapeau reste posé pour la reprise.
//
// Une file terminée refait le décompte AVANT de se dire arrêtée : l'interface ne voit jamais « file finie » avec
// l'ancien décompte (ce qui a échoué reste en retard et se voit).
func (s *Service) finish(ctx context.Context, interrupted bool) {
	if interrupted {
		s.mu.Lock()
		done, total := s.done, s.total
		s.running, s.waitingGame, s.userPaused = false, false, false
		s.mu.Unlock()
		slog.InfoContext(ctx, "replay_regen: file interrompue par l'arrêt, reprise à la prochaine ouverture",
			"done", done, "total", total)
		return
	}
	s.persistPending(ctx, false)
	cands, err := s.scan(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.setCountLocked(len(cands))
	} else {
		slog.WarnContext(ctx, "replay_regen: décompte final impossible", "err", err)
	}
	s.running, s.waitingGame, s.userPaused = false, false, false
	slog.InfoContext(ctx, "replay_regen: file terminée", "done", s.done, "total", s.total, "failed", s.failed)
}

// waitTurn attend que le film suivant puisse partir : pas en pause, et pas pendant une partie. Rend faux si ctx est
// annulé en attendant.
func (s *Service) waitTurn(ctx context.Context) bool {
	every := s.deps.gateEvery()
	announced := false
	for {
		if ctx.Err() != nil {
			return false
		}
		s.mu.Lock()
		paused := s.userPaused
		s.mu.Unlock()
		allowed := s.deps.HeavyAllowed == nil || s.deps.HeavyAllowed()
		if !paused && allowed {
			s.setWaitingGame(false)
			return true
		}
		s.setWaitingGame(!paused && !allowed)
		if !paused && !allowed && !announced {
			announced = true
			observability.IncCounter(MetricDeferredGame)
			slog.InfoContext(ctx, "game_mode: régénération des rejeux différée jusqu'à la fin de la partie")
		}
		select {
		case <-ctx.Done():
			return false
		case <-s.wake:
		case <-time.After(every):
		}
	}
}

func (s *Service) setWaitingGame(v bool) {
	s.mu.Lock()
	s.waitingGame = v
	s.mu.Unlock()
}

// benign : les refus voulus de la cuisson (mode non géré, carte hors catalogue, clé de film inconnue, film non
// finalisé). Ce ne sont pas des pannes : le film est écarté et ne compte plus comme « en retard » dans ce processus.
var benign = []error{
	replaybuild.ErrModeNonGere, replaybuild.ErrMapNotInCatalog, replaybuild.ErrUnknownFilmKey, finalise.ErrFilmNonFinalise,
}

func isBenign(err error) bool {
	for _, b := range benign {
		if errors.Is(err, b) {
			return true
		}
	}
	return false
}

// process régénère UN film : relit son état (un rejeu déjà refait à l'ouverture se saute), reconstruit, classe.
func (s *Service) process(ctx context.Context, c Candidate) {
	defer s.addDone()
	if !artifactStale(s.deps.Paths.ReplayArtifactPath(c.TitleSlug, c.MatchID)) {
		return // idempotence : déjà à jour (refait à l'ouverture, ou par un cycle de synchronisation)
	}
	err := s.rebuildWithRetry(ctx, c)
	switch {
	case err == nil:
		observability.IncCounter(MetricDone)
	case ctx.Err() != nil:
		// arrêt du serveur : ni réussi ni échoué, repris à l'ouverture suivante
	case isBenign(err):
		observability.IncCounter(MetricSkipped)
		s.block(c.TitleSlug, c.MatchID)
		slog.InfoContext(ctx, "replay_regen: rejeu écarté par la cuisson",
			"titleSlug", c.TitleSlug, "match_id", c.MatchID, "err", err)
	default:
		observability.IncCounter(MetricFailed)
		s.mu.Lock()
		s.failed++
		s.mu.Unlock()
		slog.WarnContext(ctx, "replay_regen: rejeu non régénéré",
			"titleSlug", c.TitleSlug, "match_id", c.MatchID, "err", err)
	}
}

func (s *Service) addDone() {
	s.mu.Lock()
	s.done++
	s.mu.Unlock()
}

// rebuildWithRetry reconstruit le film ; un décodage déjà en cours sur la machine (synchronisation) se laisse
// passer : la file attend et recommence le même film, borné.
func (s *Service) rebuildWithRetry(ctx context.Context, c Candidate) error {
	var err error
	for attempt := 0; attempt <= s.deps.busyRetries(); attempt++ {
		err = s.deps.Rebuild(ctx, c.TitleSlug, c.MatchID)
		if err == nil || !errors.Is(err, ErrBusy) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.deps.busyEvery()):
		}
	}
	return err
}
