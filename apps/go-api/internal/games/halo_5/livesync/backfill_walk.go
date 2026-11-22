package livesync

// backfill_walk.go — la boucle de RunBackfill : une page à la fois, capturée puis persistée, avec le
// marque-page et l'abandon des pages en échec durable quand deps.Resume est posé (voir BackfillResume).

import (
	"context"
	"log/slog"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// Compteurs expvar du rattrapage Halo 5 (namespace "levelup" sur /debug/vars, clés fixes, ADR 0009).
const (
	// metricAbandonedPages compte les pages abandonnées après maxCatchUpAttempts passes en échec : un
	// trou connu et journalisé dans l'historique rattrapé. Une valeur qui grimpe = une page durablement
	// illisible chez l'API ou un match que la base refuse.
	metricAbandonedPages = "h5_history_abandoned_pages_total"
	// metricCursorMissed compte les reprises où le marque-page n'était plus sur la page de recouvrement :
	// la passe est repartie du début de la liste (lent, sûr).
	metricCursorMissed = "h5_history_cursor_missed_total"
)

// backfillRun porte l'état d'une passe de RunBackfill.
type backfillRun struct {
	deps     BackfillDeps
	pageSize int
	logger   *slog.Logger
	known    map[string]bool
	isKnown  func(matchID string) bool
	cum      halo5.CaptureStats   // cumulé sur toutes les pages
	seen     map[string]struct{}  // anti-boucle partagé entre pages
	stats    BackfillStats        // compteurs de la passe
	pending  *historystate.Cursor // dernière position propre pas encore rangée
	blocked  bool                 // une page en échec non abandonnée : le marque-page n'avance plus
	verify   bool                 // la page en cours est la page de recouvrement
	attempts int                  // échecs consécutifs de la première page non abandonnée
	skipped  int                  // pages abandonnées juste après le marque-page
}

func newBackfillRun(deps BackfillDeps, pageSize int, logger *slog.Logger, known map[string]bool) *backfillRun {
	return &backfillRun{
		deps: deps, pageSize: pageSize, logger: logger, known: known,
		isKnown: func(id string) bool { return known[id] },
		seen:    make(map[string]struct{}, 64),
	}
}

// walk parcourt la liste page par page jusqu'à la fin, au budget, à la profondeur maximale ou à une
// erreur de page non abandonnée.
func (r *backfillRun) walk(ctx context.Context) error {
	start, done, err := r.begin(ctx)
	if done || err != nil {
		return err
	}
	for ; start < historystate.MaxDepth; start += r.pageSize {
		more, err := r.readPage(ctx, start)
		if err != nil {
			if r.failFrontier(ctx, start, err.Error(), 0) {
				continue // page abandonnée : la suivante
			}
			return err
		}
		if !more || r.budgetReached() {
			return nil
		}
	}
	return nil
}

// begin règle la reprise : rend le décalage de la première page à lire dans la boucle, ou done quand la
// page de recouvrement a déjà tout dit (fin vue, budget atteint).
func (r *backfillRun) begin(ctx context.Context) (start int, done bool, err error) {
	res := r.deps.Resume
	if res == nil {
		return 0, false, nil
	}
	r.attempts, r.skipped = res.Attempts, res.Skip
	start = max(res.Start, 0)
	if res.ExpectID == "" {
		return start + res.Skip*r.pageSize, false, nil
	}
	r.verify = true
	more, err := r.readPage(ctx, start)
	r.verify = false
	if err != nil {
		return 0, true, err
	}
	if _, ok := r.seen[res.ExpectID]; !ok {
		r.restart(ctx, start)
		return 0, false, nil
	}
	if !more || r.budgetReached() {
		return 0, true, nil
	}
	return start + (1+res.Skip)*r.pageSize, false, nil
}

// restart reprend la passe au début de la liste : le marque-page n'est pas sur la page de recouvrement
// (la liste a changé sous lui). Rien de ce qui a été lu de la page de recouvrement ne sert de preuve.
func (r *backfillRun) restart(ctx context.Context, start int) {
	slog.WarnContext(ctx, "h5 backfill: marque-page absent de la page de recouvrement — reprise depuis le début",
		"gamertag", r.deps.Viewer.Gamertag, "overlap_start", start, "expected_match_id", r.deps.Resume.ExpectID)
	observability.IncCounter(metricCursorMissed)
	r.stats.ResumeMissed = true
	r.seen = make(map[string]struct{}, 64)
	r.cum.HistoryEnd = false
	r.pending, r.blocked = nil, false
	r.attempts, r.skipped = 0, 0
}

// readPage capture et persiste UNE page à `start`. Rend vrai quand il reste probablement des pages.
// Une erreur de capture remonte telle quelle ; une erreur de persistance ne l'arrête pas (le match
// reste inconnu, la passe suivante le retente) mais bloque le marque-page avant cette page.
func (r *backfillRun) readPage(ctx context.Context, start int) (bool, error) {
	before := r.cum
	batches, hasMore, err := r.deps.CapturePage(ctx, r.deps.Viewer, r.deps.ResolveXUID, r.isKnown, start, r.pageSize, r.seen, &r.cum)
	if err != nil {
		slog.ErrorContext(ctx, "h5 backfill: capture page échouée (arrêt)", "start", start, "err", err)
		return false, err
	}
	r.stats.Pages++

	inserted, errs := persistAndTrack(ctx, r.deps, batches, r.known, &r.stats, r.logger)
	r.logger.InfoContext(ctx, "h5 backfill: page",
		"page", r.stats.Pages-1, "start", start,
		"page_seen", r.cum.MatchesSeen-before.MatchesSeen,
		"inserted", inserted, "cumulated", r.stats.Inserted,
		"skipped_cum", r.cum.MatchesSkipped, "oldest_match_date", oldestStart(batches))

	clean := len(errs) == 0 || r.failFrontier(ctx, start, errs[0], len(errs))
	if clean && hasMore {
		r.advance(ctx, start, r.cum.MatchesSeen-before.MatchesSeen, inserted)
	}
	return hasMore, nil
}

// failFrontier note l'échec de la page à `start` (cause : le texte de l'erreur, nErrs : ses erreurs de
// persistance). Rend vrai quand la page est abandonnée : elle était la première après le marque-page
// et c'est son MaxAttempts-ième échec de suite ; elle ne bloque alors plus le marque-page ni la fin.
// Sinon elle le bloque (les pages suivantes ne le font plus avancer).
func (r *backfillRun) failFrontier(ctx context.Context, start int, cause string, nErrs int) bool {
	r.stats.LastError = cause
	res := r.deps.Resume
	if r.blocked || r.verify || res == nil {
		r.blocked = true
		return false
	}
	r.attempts++
	if res.MaxAttempts > 0 && r.attempts >= res.MaxAttempts {
		slog.WarnContext(ctx, "h5 backfill: page abandonnée après des échecs répétés — historique incomplet à cet endroit",
			"gamertag", r.deps.Viewer.Gamertag, "offset", start, "attempts", r.attempts, "last_error", cause)
		observability.IncCounter(metricAbandonedPages)
		r.stats.AbandonedPages++
		r.stats.AbandonedErrors += nErrs
		r.attempts = 0
		r.skipped++
		return true
	}
	r.blocked = true
	return false
}

// advance fait avancer le marque-page après une page pleine propre : il pointe la fin de la page, preuve
// de position = son dernier match. Rangé tout de suite quand la page a inséré des matchs, sinon à la
// prochaine occasion (flushCursor) : une page de matchs déjà connus ne coûte que de la relecture.
func (r *backfillRun) advance(ctx context.Context, start, entries, inserted int) {
	if r.blocked || r.deps.Resume == nil {
		return
	}
	if !r.verify { // la page de recouvrement est derrière la frontière : elle ne remet pas les échecs à zéro
		r.attempts, r.skipped = 0, 0
	}
	c := historystate.Cursor{Offset: start + entries, LastMatchID: r.cum.LastMatchID}
	if !c.Valid() {
		return
	}
	r.pending = &c
	if inserted > 0 {
		r.flushCursor(ctx)
	}
}

// flushCursor range la dernière position propre qui ne l'est pas encore.
func (r *backfillRun) flushCursor(ctx context.Context) {
	if r.pending == nil || r.deps.Resume == nil || r.deps.Resume.Save == nil {
		return
	}
	r.deps.Resume.Save(ctx, *r.pending)
	r.pending = nil
}

// budgetReached dit si la passe a inséré son budget de matchs (0 = sans borne).
func (r *backfillRun) budgetReached() bool {
	if r.deps.Budget > 0 && r.stats.Inserted >= r.deps.Budget {
		r.stats.BudgetReached = true
		return true
	}
	return false
}
