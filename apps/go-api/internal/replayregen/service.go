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
	// ErrNotRunning : pause ou reprise demandée sans file active (file jamais lancée, ou déjà finie).
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
	// RefusedByRule rend, pour un titre, l'ensemble des matchs (forme courte, celle du nom du fichier) que la cuisson
	// refuse PAR RÈGLE : un mode que le rejeu ne gère pas (replaybuild.ErrModeNonGere, lu au registre par le même
	// prédicat que la cuisson). Le décompte les écarte d'emblée au lieu d'attendre leur premier refus de cuisson ; une
	// erreur est journalisée et le décompte retombe sur le comportement d'avant (refus au premier essai). nil = aucun.
	RefusedByRule func(ctx context.Context, titleSlug string) (map[string]struct{}, error)
	// Rebuild reconstruit et range le rejeu d'UN match, hors du processus (même chemin que l'action admin). Il
	// enveloppe ErrBusy quand un autre décodage tient la machine.
	Rebuild func(ctx context.Context, titleSlug, matchID string) error
	Pending PendingStore
	// HeavyAllowed : la porte du mode jeu (gameproc.Mode.HeavyWorkAllowed), consultée avant chaque film, avant
	// chaque nouvel essai d'un film occupé et avant chaque décompte.
	HeavyAllowed func() bool
	// SyncRunning : un cycle de synchronisation est en cours. La file lui CÈDE LA PLACE : entre deux films elle
	// attend la fin du cycle (le décodage de l'étape kill source et la cuisson après synchro prennent le même
	// verrou solo, et se reportent au cycle suivant quand la file le tient). nil = jamais.
	SyncRunning func() bool
	// Rythmes ; zéro = valeur de production.
	GateEvery      time.Duration
	BusyRetryEvery time.Duration
	BusyRetries    int
}

const (
	defaultGateEvery      = 15 * time.Second
	defaultBusyRetryEvery = 20 * time.Second
	defaultBusyRetries    = 30
)

func orDefault(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}

func (d Deps) gateEvery() time.Duration { return orDefault(d.GateEvery, defaultGateEvery) }
func (d Deps) busyEvery() time.Duration { return orDefault(d.BusyRetryEvery, defaultBusyRetryEvery) }
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

	// Décompte (hors file active). Il ne se lance QUE : au démarrage du serveur, à la fin de la file et sur
	// demande explicite ([Service.Rescan]) ; Status ne le relance jamais.
	checked  bool
	stale    int
	scanning bool
	blocked  map[string]struct{}
	// countSeq numérote les décomptes à leur lancement ; countApplied est le numéro du dernier publié. Un
	// décompte plus ancien que le dernier publié est jeté : il décrit un état d'avant la file.
	countSeq     int64
	countApplied int64

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

// Status rend l'état courant, sans rien lancer : hors file, le décompte vient du cache (fait au démarrage, à la
// fin de la file ou sur [Service.Rescan]).
func (s *Service) Status() domain.ReplayRegenStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked()
}

// Rescan lance un décompte en arrière-plan, sous la porte du mode jeu (démarrage du serveur, demande explicite de
// l'utilisateur). Sans effet pendant la file (elle refait le décompte à sa fin) ou si un décompte est déjà en
// cours. Rend l'état courant : `checked` vaut faux jusqu'à la fin du décompte.
func (s *Service) Rescan() (domain.ReplayRegenStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.bound {
		return domain.ReplayRegenStatus{}, ErrNotBound
	}
	if !s.running && !s.scanning {
		s.scanning, s.checked = true, false
		seq := s.beginCountLocked()
		s.wg.Add(1)
		go s.backgroundScan(seq)
	}
	return s.statusLocked(), nil
}

// beginCountLocked numérote un décompte qui démarre.
func (s *Service) beginCountLocked() int64 {
	s.countSeq++
	return s.countSeq
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
	if !resumed {
		s.persistPendingLocked(ctx, true)
	}
	s.mu.Unlock()

	observability.IncCounter(MetricStarted)
	go s.run(ctx)
	return st, nil
}

// Pause met la file en pause à la demande de l'utilisateur : le film en cours va à son terme, le suivant ne part
// pas. Une file en pause ne reprend pas toute seule à l'ouverture suivante (drapeau effacé).
func (s *Service) Pause() (domain.ReplayRegenStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return domain.ReplayRegenStatus{}, ErrNotRunning
	}
	s.userPaused = true
	// Le drapeau s'écrit sous le verrou, dans le même geste que l'état : `finish` ne peut pas s'intercaler entre la
	// vérification « file active » et l'écriture (une reprise ne repose jamais le drapeau d'une file finie).
	s.persistPendingLocked(s.ctx, false)
	return s.statusLocked(), nil
}

// Resume relance une file en pause. Une file finie (même si son décompte final est en cours) n'est plus active :
// ErrNotRunning.
func (s *Service) Resume() (domain.ReplayRegenStatus, error) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return domain.ReplayRegenStatus{}, ErrNotRunning
	}
	s.userPaused = false
	s.persistPendingLocked(s.ctx, true)
	st := s.statusLocked()
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return st, nil
}

// persistPendingLocked écrit le drapeau de reprise ; s.mu est tenu (l'écriture est un petit fichier).
func (s *Service) persistPendingLocked(ctx context.Context, on bool) {
	p := s.deps.Pending
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
// LA FILE EST FINIE AVANT TOUT DÉCOMPTE : `running` et la pause tombent d'abord, et le drapeau est effacé dans le
// même geste. Une pause ou une reprise qui arrive pendant le décompte final trouve une file finie (409) et ne
// repose jamais le drapeau. Pendant le décompte, `checked` vaut faux : l'interface n'affiche pas « file finie »
// avec l'ancien décompte (ce qui a échoué reste en retard et se voit une fois le décompte fait).
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
	s.mu.Lock()
	s.running, s.waitingGame, s.userPaused = false, false, false
	wasChecked := s.checked
	s.checked = false
	s.persistPendingLocked(ctx, false)
	seq := s.beginCountLocked()
	done, total, failed := s.done, s.total, s.failed
	s.mu.Unlock()

	cands, err := s.scanUnderGate(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.checked = wasChecked
		slog.WarnContext(ctx, "replay_regen: décompte final impossible", "err", err)
	} else {
		s.publishCountLocked(seq, len(cands))
	}
	slog.InfoContext(ctx, "replay_regen: file terminée", "done", done, "total", total, "failed", failed)
}

// turnBlock : ce qui retient le film suivant.
type turnBlock struct{ paused, game, syncing bool }

func (b turnBlock) any() bool { return b.paused || b.game || b.syncing }

// blockNow lit ce qui retient le film suivant : la pause, la partie en cours, un cycle de synchronisation.
func (s *Service) blockNow() turnBlock {
	s.mu.Lock()
	paused := s.userPaused
	s.mu.Unlock()
	return turnBlock{
		paused:  paused,
		game:    s.deps.HeavyAllowed != nil && !s.deps.HeavyAllowed(),
		syncing: s.deps.SyncRunning != nil && s.deps.SyncRunning(),
	}
}

// announceBlock journalise (et compte) chaque cause d'attente UNE fois par attente ; seen porte ce qui l'a déjà été.
func announceBlock(ctx context.Context, b turnBlock, seen *turnBlock) {
	if b.game && !b.paused && !seen.game {
		seen.game = true
		observability.IncCounter(MetricDeferredGame)
		slog.InfoContext(ctx, "game_mode: régénération des rejeux différée jusqu'à la fin de la partie")
	}
	if b.syncing && !b.paused && !b.game && !seen.syncing {
		seen.syncing = true
		slog.InfoContext(ctx, "replay_regen: file en attente de la fin du cycle de synchronisation")
	}
}

// waitTurn attend que le film suivant puisse partir : pas en pause, pas pendant une partie, pas pendant un cycle
// de synchronisation (la file lui cède la place). Rend faux si ctx est annulé en attendant.
func (s *Service) waitTurn(ctx context.Context) bool {
	every := s.deps.gateEvery()
	var seen turnBlock
	for {
		if ctx.Err() != nil {
			return false
		}
		b := s.blockNow()
		if !b.any() {
			s.setWaitingGame(false)
			return true
		}
		s.setWaitingGame(b.game && !b.paused)
		announceBlock(ctx, b, &seen)
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
	if !artifactStale(ctx, s.deps.Paths.ReplayArtifactPath(c.TitleSlug, c.MatchID)) {
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
// passer : la file attend et recommence le même film, borné. CHAQUE nouvel essai repasse par [Service.waitTurn] :
// la porte du mode jeu, la pause et la synchronisation valent pour un essai comme pour un premier lancement.
func (s *Service) rebuildWithRetry(ctx context.Context, c Candidate) error {
	var err error
	for attempt := 0; attempt <= s.deps.busyRetries(); attempt++ {
		if attempt > 0 && !s.waitTurn(ctx) {
			return ctx.Err()
		}
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
