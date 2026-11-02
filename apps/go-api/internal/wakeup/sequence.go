package wakeup

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/clock"
)

// Raisons d'une séquence, journalisées et transmises au rattrapage.
const (
	// ReasonLaunch : la séquence du lancement de l'application.
	ReasonLaunch = "lancement"
	// ReasonResume : la séquence d'une sortie de veille détectée.
	ReasonResume = "sortie de veille"
)

// maxRetryEvery : le plus grand nombre de tics entre deux reprises après un réseau absent (20 tics
// de [SleepTick] = 10 minutes).
const maxRetryEvery = 20

// CatchUp est ce que la séquence demande à l'ordonnanceur d'auto-sync (*scheduler.AutoSyncScheduler).
type CatchUp interface {
	// CatchUpDue dit si la dernière synchronisation connue est plus vieille que l'intervalle à now.
	CatchUpDue(now time.Time) (due bool, last time.Time, interval time.Duration)
	// RequestCatchUp tire un tick immédiat dans la boucle de l'ordonnanceur ; false si une demande
	// attend déjà.
	RequestCatchUp(reason string) bool
}

// Sequence est la séquence de réveil : vérification des jetons, puis rattrapage d'un cycle de sync
// en retard. Deux déclencheurs, UNE seule fonction ([Sequence.Wake]) : le lancement et la sortie de veille
// que détecte [Sequence.Run].
//
// Un réseau absent (aucun joueur n'a obtenu de réponse) ne marque rien : la synchronisation attend
// elle aussi, et la vérification est reprise à un tic ultérieur, de plus en plus espacé. Une sortie de
// veille est justement le cas où le réseau n'est pas encore revenu.
type Sequence struct {
	// Tokens vérifie les jetons des profils suivis.
	Tokens TokenChecker
	// Sync rattrape le cycle de sync en retard ; nil : pas de rattrapage (mode sans ordonnanceur).
	Sync CatchUp
	// Clock rend l'heure du jour comparée à la dernière synchronisation connue.
	Clock clock.Clock
	// Sleep échantillonne les horloges à chaque tic ; nil : détecteur de production.
	Sleep *SleepDetector
	// Ticks : les tics de [SleepTick] ; nil : un ticker de production, arrêté avec Run.
	Ticks <-chan time.Time

	retry retryState
}

// retryState : ce qu'il reste à refaire après un réseau absent.
type retryState struct {
	only    []string // xuid dont le jeton reste à vérifier
	catchUp bool     // le rattrapage attend le retour du réseau
	reason  string   // raison de la séquence interrompue
	waited  int      // tics écoulés depuis la dernière tentative
	every   int      // tics à attendre entre deux tentatives
}

// active dit si une reprise est en attente.
func (r retryState) active() bool { return len(r.only) > 0 || r.catchUp }

// Run exécute la séquence du lancement, puis guette les sorties de veille jusqu'à l'annulation de ctx.
// Bloquant : à lancer dans une goroutine suivie.
func (s *Sequence) Run(ctx context.Context) {
	ticks := s.Ticks
	if ticks == nil {
		ticker := time.NewTicker(SleepTick)
		defer ticker.Stop()
		ticks = ticker.C
	}
	if s.Sleep == nil {
		s.Sleep = NewSleepDetector(NewSystemClocks())
	}
	s.Wake(ctx, ReasonLaunch)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			s.onTick(ctx)
		}
	}
}

// Wake est la séquence elle-même : vérifier les jetons, puis rattraper la synchronisation en retard.
func (s *Sequence) Wake(ctx context.Context, reason string) {
	slog.InfoContext(ctx, "wakeup: séquence de réveil", "raison", reason)
	report := s.Tokens.Check(ctx, nil)
	absent := report.NetworkAbsent()
	s.retry = retryState{only: report.Unreachable(), catchUp: absent, reason: reason, every: 1}
	if absent {
		slog.InfoContext(ctx, "wakeup: réseau absent — jetons non marqués, synchronisation différée",
			"raison", reason, "joueurs", len(s.retry.only))
		return
	}
	s.catchUpIfDue(ctx, reason)
}

// onTick traite un tic : sortie de veille détectée (séquence complète), sinon reprise éventuelle
// d'une vérification restée sans réseau.
func (s *Sequence) onTick(ctx context.Context) {
	if gap, woke := s.Sleep.Check(); woke {
		slog.InfoContext(ctx, "wakeup: sortie de veille détectée", "ecart", gap.Round(time.Second))
		s.Wake(ctx, ReasonResume)
		return
	}
	if !s.retry.active() {
		return
	}
	s.retry.waited++
	if s.retry.waited < s.retry.every {
		return
	}
	s.retry.waited = 0
	s.retry.every = min(s.retry.every*2, maxRetryEvery)
	s.resume(ctx)
}

// resume refait la vérification des jetons restés sans réponse et, si le réseau est revenu, le
// rattrapage qui l'attendait.
func (s *Sequence) resume(ctx context.Context) {
	report := s.Tokens.Check(ctx, s.retry.only)
	s.retry.only = report.Unreachable()
	if !s.retry.catchUp || report.NetworkAbsent() {
		return
	}
	s.retry.catchUp = false
	slog.InfoContext(ctx, "wakeup: réseau revenu — reprise de la séquence", "raison", s.retry.reason)
	s.catchUpIfDue(ctx, s.retry.reason)
}

// catchUpIfDue demande un cycle de sync immédiat si la dernière synchronisation connue est plus vieille
// que l'intervalle ; sinon le prochain tick reste à l'heure prévue.
func (s *Sequence) catchUpIfDue(ctx context.Context, reason string) {
	if s.Sync == nil {
		return
	}
	clk := s.Clock
	if clk == nil {
		clk = clock.System{}
	}
	due, last, interval := s.Sync.CatchUpDue(clk.Now())
	if !due {
		slog.InfoContext(ctx, "wakeup: dernière synchronisation à l'heure — prochain tick à l'heure prévue",
			"derniere", last, "intervalle", interval)
		return
	}
	slog.InfoContext(ctx, "wakeup: dernière synchronisation plus vieille que l'intervalle — cycle immédiat",
		"derniere", last, "intervalle", interval, "raison", reason)
	if !s.Sync.RequestCatchUp(reason) {
		slog.InfoContext(ctx, "wakeup: un rattrapage attend déjà dans l'ordonnanceur")
	}
}
