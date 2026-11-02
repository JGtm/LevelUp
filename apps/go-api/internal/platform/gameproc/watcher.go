package gameproc

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability"
)

const (
	// DefaultInterval : période du sondage des processus.
	DefaultInterval = 15 * time.Second
	// DefaultDebounce : délai minimal entre deux synchronisations déclenchées par une fermeture du
	// jeu (un jeu relancé aussitôt qui plante ne doit pas enchaîner les cycles).
	DefaultDebounce = 60 * time.Second
)

// Watcher sonde les processus et tient le [Mode] à jour.
type Watcher struct {
	// Lister énumère les processus ; obligatoire.
	Lister Lister
	// Mode reçoit l'état ; obligatoire.
	Mode *Mode
	// Names rend les noms d'exécutables du jeu (union des titres), relus à chaque sondage.
	Names func() []string
	// Enabled rend le réglage `game_process_watch`, relu à chaque sondage ; nil = actif. Réglage
	// coupé : le mode retombe à « pas de jeu » sans déclencher de synchronisation.
	Enabled func() bool
	// OnGameClosed est appelé à la fermeture du jeu, hors anti-rebond : il demande une
	// synchronisation et ne bloque pas. reason alimente le journal de la synchronisation.
	OnGameClosed func(reason string)
	// Interval : période des sondages ; zéro = [DefaultInterval].
	Interval time.Duration
	// Debounce : anti-rebond de la synchronisation ; zéro = [DefaultDebounce].
	Debounce time.Duration
	// Ticks : les tics de sondage ; nil = un ticker de [Watcher.Interval], arrêté avec Run.
	Ticks <-chan time.Time
	// Now : l'horloge de l'anti-rebond ; nil = time.Now.
	Now func() time.Time

	lastSync time.Time
	lastErr  string
}

// ReasonGameClosed : la raison transmise à [Watcher.OnGameClosed].
const ReasonGameClosed = "jeu fermé"

// Run sonde une première fois puis à chaque tic, jusqu'à l'annulation de ctx. Bloquant : à lancer
// dans une goroutine suivie.
func (w *Watcher) Run(ctx context.Context) {
	ticks := w.Ticks
	if ticks == nil {
		every := w.Interval
		if every <= 0 {
			every = DefaultInterval
		}
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		ticks = ticker.C
	}
	w.Poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			w.Poll(ctx)
		}
	}
}

// Poll fait un sondage : relit le réglage, énumère les processus et applique la transition.
func (w *Watcher) Poll(ctx context.Context) {
	if w.Enabled != nil && !w.Enabled() {
		if w.Mode.set(false) {
			slog.InfoContext(ctx, "game_mode: leave", "raison", "détection désactivée dans les réglages")
		}
		return
	}
	running, err := w.Lister.Names()
	if err != nil && len(running) == 0 {
		observability.IncCounter(MetricPollErrors)
		// Une panne qui dure ne se journalise qu'une fois : l'état précédent est conservé.
		if msg := err.Error(); msg != w.lastErr {
			w.lastErr = msg
			slog.WarnContext(ctx, "game_mode: énumération des processus impossible, état conservé", "err", err)
		}
		return
	}
	w.lastErr = ""
	var wanted []string
	if w.Names != nil {
		wanted = w.Names()
	}
	w.apply(ctx, AnyRunning(running, wanted))
}

// apply pose l'état observé et déclenche ce que la transition appelle.
func (w *Watcher) apply(ctx context.Context, running bool) {
	if !w.Mode.set(running) {
		return
	}
	if running {
		observability.IncCounter(MetricEnter)
		slog.InfoContext(ctx, "game_mode: enter", "effet", "cuisson des rejeux et tâches lourdes suspendues")
		return
	}
	observability.IncCounter(MetricLeave)
	slog.InfoContext(ctx, "game_mode: leave", "effet", "tâches lourdes reprises")
	w.requestSync(ctx)
}

// requestSync demande la synchronisation de fin de partie, sauf si la précédente date de moins de
// l'anti-rebond.
func (w *Watcher) requestSync(ctx context.Context) {
	if w.OnGameClosed == nil {
		return
	}
	now := time.Now()
	if w.Now != nil {
		now = w.Now()
	}
	debounce := w.Debounce
	if debounce <= 0 {
		debounce = DefaultDebounce
	}
	if !w.lastSync.IsZero() && now.Sub(w.lastSync) < debounce {
		observability.IncCounter(MetricSyncDebounced)
		slog.InfoContext(ctx, "game_mode: synchronisation de fin de partie absorbée par l'anti-rebond",
			"depuis", now.Sub(w.lastSync).Round(time.Second), "anti_rebond", debounce)
		return
	}
	w.lastSync = now
	observability.IncCounter(MetricSyncTriggered)
	slog.InfoContext(ctx, "game_mode: synchronisation de fin de partie demandée")
	w.OnGameClosed(ReasonGameClosed)
}
