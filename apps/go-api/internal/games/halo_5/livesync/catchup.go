package livesync

// catchup.go — rattrapage de l'historique Halo 5 par la synchronisation automatique.
//
// Un delta (CollectRecentMatches) s'arrête après domain.DeltaKnownRun matchs connus d'affilée : une première synchronisation
// interrompue laisserait un trou définitif derrière les matchs déjà rangés. Tant que la fin de
// l'historique n'a pas été vue pour le joueur (internal/sync/historystate), Runner.RunDelta fait donc
// une passe de rattrapage par la mécanique du backfill (RunBackfill : connus sautés, persistance page
// par page), bornée à historystate.PassBudget() matchs insérés. La fin se range seulement quand la
// passe a vu la fin de la liste sans erreur de page ni de persistance.
//
// Marque-page : la passe range la position où elle s'est arrêtée (historystate.Cursor : décalage dans la
// liste de l'API, dernier match traité et ancre, l'entrée en tête de liste à cette passe). La suivante
// (1) fait le delta du dessus, jusqu'à une suite de matchs connus (domain.DeltaKnownRun), et compte N, le nombre d'entrées arrivées
// en tête : la position de l'ancre dans la liste relue (catchup_anchor.go), toutes entrées comptées,
// écartées comprises ; sans ancre, le nombre d'entrées vues jusqu'à la dernière entrée non connue (CaptureStats.TopSpan) ; (2) reprend au
// décalage rangé + N moins une page de recouvrement ; (3) exige l'identifiant rangé sur la page de
// recouvrement, sinon repart du début (journalisé, compté) ; une ancre introuvable en tête de liste
// repart du début aussi. Une page qui échoue maxCatchUpAttempts passes de suite est abandonnée
// (historystate.Streak), pour qu'un échec durable ne garde pas le joueur en rattrapage pour toujours ; une
// panne de l'API (429, réseau, délai, passe de masse : historyfail) ne compte pas. Une page abandonnée est un trou
// noté (historystate.NoteHole) : passé historystate.AbandonRetryAfter, readHistoryState rouvre l'historique
// (historystate.Reopen) et la passe suivante relit la liste depuis son début, connus sautés, pour retenter la page.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/historystate"
)

// maxCatchUpAttempts est le nombre de passes de suite qu'une même page peut échouer avant d'être abandonnée.
const maxCatchUpAttempts = 3

// originSubject nomme le point d'échec d'une passe sans marque-page (la liste depuis son début).
const originSubject = "origin"

// historyState : ce que RunDelta sait de l'historique du joueur au début du cycle.
type historyState struct {
	catchUp bool // fin jamais vue, la passe est une passe de rattrapage
	pending bool // l'historique est (ou peut être) incomplet : vrai tant que la fin n'est pas rangée
}

// readHistoryState lit l'état de l'historique, après avoir rouvert celui d'un joueur dont une page abandonnée
// date de plus de historystate.AbandonRetryAfter. Sans magasin : complet (delta ordinaire, runner testable
// sans base joueur). Base joueur illisible : delta ordinaire mais en attente, pour que l'ordonnanceur
// replanifie un cycle court (la lecture de l'état ne bloque jamais le delta).
func (r *Runner) readHistoryState(ctx context.Context) historyState {
	if r.deps.History == nil {
		return historyState{}
	}
	if r.deps.History.HoleDue(ctx) && r.deps.History.Reopen(ctx) {
		slog.InfoContext(ctx, "h5 sync: page abandonnée depuis plus de 24 h — historique rouvert pour la retenter",
			"gamertag", r.deps.Viewer.Gamertag)
	}
	complete, err := r.deps.History.Complete(ctx)
	if err != nil {
		r.logger.WarnContext(ctx, "h5 sync: lecture de l'historique impossible, delta ordinaire",
			"gamertag", r.deps.Viewer.Gamertag, "err", err)
		return historyState{pending: true}
	}
	return historyState{catchUp: !complete, pending: !complete}
}

// catchUpBudget est le budget de la passe : celui de l'historique, ou le plafond plus serré que
// l'appelant demande (le tick du suivi en direct passe un petit MaxMatches).
func catchUpBudget(opts domain.SyncOptions) int {
	budget := historystate.PassBudget()
	if opts.MaxMatches > 0 && opts.MaxMatches < budget {
		return opts.MaxMatches
	}
	return budget
}

// catchUpPass porte l'état d'une passe de rattrapage.
type catchUpPass struct {
	r       *Runner
	store   HistoryStore
	src     halo5.CaptureSource
	resolve func(string) string
	known   map[string]bool
	res     *domain.SyncResult
	stats   halo5.CaptureStats   // cumul du delta du dessus et de la reprise
	saved   *historystate.Cursor // dernier marque-page rangé pendant la passe
	head    string               // entrée en tête de liste lue par le delta du dessus : l'ancre à ranger
}

// runCatchUp exécute une passe de rattrapage et range ses compteurs dans res. known est l'ensemble
// des matchs déjà rangés, lu par RunDelta (la lecture unique de loadKnownMatchIDs) et complété au
// fil des pages. Rend les compteurs de capture de la passe et vrai quand la fin de l'historique est rangée.
func (r *Runner) runCatchUp(ctx context.Context, src halo5.CaptureSource, resolveXUID func(string) string,
	known map[string]bool, budget int, res *domain.SyncResult) (halo5.CaptureStats, bool) {
	p := &catchUpPass{r: r, store: r.deps.History, src: src, resolve: resolveXUID, known: known, res: res}
	marked := p.run(ctx, budget)
	tallySkipped(res, p.stats.MatchesSeen)
	return p.stats, marked
}

// run enchaîne le delta du dessus (avec un marque-page) puis la reprise, et range la fin quand elle est vue.
func (p *catchUpPass) run(ctx context.Context, budget int) bool {
	resume := &BackfillResume{MaxAttempts: maxCatchUpAttempts, Save: p.saveCursor, OnAbandon: func(ctx context.Context) { p.store.NoteHole(ctx) }}
	subject, remaining, topClean, restart := originSubject, budget, true, false
	if cur, ok := p.store.LoadCursor(ctx); ok {
		subject = cur.LastMatchID
		top := p.topDelta(ctx, cur, budget, resume)
		if top.stop {
			return top.marked
		}
		remaining, topClean, restart = budget-top.inserted, top.clean, top.restart
	}
	if remaining <= 0 {
		return false
	}
	if !restart { // reprise depuis 0 : les pages abandonnées et les échecs comptés visaient l'ancienne position
		streak := p.store.LoadStreak(ctx, subject)
		resume.Skip, resume.Attempts = streak.Skipped, streak.Attempts
	}
	bs, err := RunBackfill(ctx, BackfillDeps{
		CapturePage: p.capturePage(),
		Viewer:      p.r.deps.Viewer,
		ResolveXUID: p.resolve,
		LoadKnown:   func(context.Context) (map[string]bool, error) { return p.known, nil },
		PersistPage: p.persistPage,
		Budget:      remaining,
		Resume:      resume,
	}, 0, p.r.logger)
	addCapture(&p.stats, bs.Capture)
	p.res.CatchUpInserted += bs.Inserted // le delta du dessus (topDelta) n'y entre pas : ce sont les nouveaux matchs
	return p.finish(ctx, subject, bs, err, topClean)
}

// capturePage rend la capture d'une page de la reprise (celle des Deps en test, sinon la source du cycle).
func (p *catchUpPass) capturePage() CapturePageFunc {
	if p.r.deps.CapturePage != nil {
		return p.r.deps.CapturePage
	}
	return capturePageFrom(p.src)
}

// persistPage range une page de la reprise et en tient les compteurs du résultat.
func (p *catchUpPass) persistPage(ctx context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
	done, errs := p.r.deps.PersistAll(ctx, batches)
	p.res.Errors = append(p.res.Errors, errs...)
	for _, b := range done {
		tallyBatch(p.res, b)
	}
	return done, errs
}

// saveCursor range un marque-page (bail joueur) et retient le dernier rangé pour nommer le point d'échec.
func (p *catchUpPass) saveCursor(ctx context.Context, c historystate.Cursor) {
	if p.store.SaveCursor(ctx, c) {
		p.saved = &c
	}
}

// finish tire les conclusions de la reprise (règle D-5) : une erreur de page ou de persistance non
// abandonnée laisse la fin non rangée et compte l'échec du point ; une passe propre efface ce compte et
// range la fin quand elle a vu la fin de la liste.
func (p *catchUpPass) finish(ctx context.Context, subject string, bs BackfillStats, err error, topClean bool) bool {
	if err != nil {
		p.res.AddError("h5 backfill: " + err.Error())
	}
	if err != nil || bs.PersistErrors > bs.AbandonedErrors {
		p.recordFailure(ctx, subject, bs)
		p.r.logger.InfoContext(ctx, "h5 sync: passe de rattrapage — historique encore incomplet",
			"gamertag", p.r.deps.Viewer.Gamertag, "inserted", bs.Inserted, "page_error", err != nil,
			"persist_errors", bs.PersistErrors, "abandoned_pages", bs.AbandonedPages)
		return false
	}
	p.store.ClearStreak(ctx)
	if !bs.HistoryEnd || !topClean {
		p.r.logger.InfoContext(ctx, "h5 sync: passe de rattrapage — historique encore incomplet",
			"gamertag", p.r.deps.Viewer.Gamertag, "inserted", bs.Inserted, "budget_reached", bs.BudgetReached,
			"history_end", bs.HistoryEnd, "top_clean", topClean, "resume_missed", bs.ResumeMissed)
		return false
	}
	return p.store.MarkComplete(ctx)
}

// recordFailure range le compte d'échecs du point où la passe s'est arrêtée : le marque-page rangé
// pendant la passe, sinon celui d'avant (le point reste le même tant que le marque-page ne bouge pas).
func (p *catchUpPass) recordFailure(ctx context.Context, subject string, bs BackfillStats) {
	if p.saved != nil {
		subject = p.saved.LastMatchID
	}
	p.store.SaveStreak(ctx, historystate.Streak{
		Subject: subject, Attempts: bs.FrontierAttempts, Skipped: bs.FrontierSkipped, LastError: bs.LastError,
	})
}

// topDeltaResult est l'issue du delta du dessus d'une passe avec marque-page.
type topDeltaResult struct {
	inserted int
	clean    bool // aucune erreur de page ni de persistance
	stop     bool // la passe s'arrête là (marked : la fin est rangée)
	marked   bool
	restart  bool // ancre perdue : la reprise repart du début de la liste
}

// topDelta ramasse les matchs plus récents qu'une suite de matchs connus, domain.DeltaKnownRun (le delta ordinaire, borné au
// budget), compte N, le nombre d'entrées arrivées en tête depuis le marque-page, et décale le
// marque-page de N : ce sont autant d'entrées de plus entre le haut de la liste et lui. N est la
// position de l'ancre du marque-page dans la liste (locateAnchor), exacte quelles que soient les
// entrées arrivées, écartées (Warzone, campagne) comprises ; sans ancre, le nombre d'entrées vues
// jusqu'à la dernière entrée non connue, et la page de recouvrement absorbe le surcompte des écartées. Une ancre
// perdue (plus d'AnchorMaxPages pages arrivées) renvoie la reprise au début de la liste.
// Le marque-page n'est décalé qu'après un delta propre : un match du dessus resté en échec ne doit pas
// être déclaré traité.
func (p *catchUpPass) topDelta(ctx context.Context, cur historystate.Cursor, budget int, resume *BackfillResume) topDeltaResult {
	batches, stats, err := p.r.deps.Capture(ctx, p.src, p.r.deps.Viewer, p.resolve,
		mapContains(p.known), halo5.CaptureOptions{
			MaxMatches: budget, TopLimit: historystate.AnchorScanLimit(backfillDefaultPageSize)})
	if err != nil {
		p.res.AddError("h5 collect: " + err.Error())
	}
	done, errs := p.r.deps.PersistAll(ctx, batches)
	p.res.Errors = append(p.res.Errors, errs...)
	for _, b := range done {
		tallyBatch(p.res, b)
		if b.Shared.Match != nil {
			p.known[b.Shared.Match.MatchID] = true
		}
	}
	addCapture(&p.stats, stats)
	p.head = stats.HeadID
	top := topDeltaResult{inserted: len(done), clean: err == nil && len(errs) == 0}
	switch {
	case err != nil:
		top.stop = true // la liste est illisible : sans N, pas de reprise
	case stats.HistoryEnd && !stats.StoppedOnKnown:
		// Tout l'historique tenait dans le delta : la fin est vue.
		top.stop = true
		top.marked = top.clean && p.store.MarkComplete(ctx)
	case !stats.StoppedOnKnown:
		// Budget atteint avant la suite de connus : la suite au prochain passage.
		top.stop = true
		if a := p.locateAnchor(ctx, cur, stats); a.kind == anchorLost {
			p.noteAnchorLost(ctx, cur) // marque-page laissé tel quel : le prochain passage le jugera
		} else {
			p.shiftCursor(ctx, cur, shiftOf(a, stats.MatchesSeen), top.clean)
		}
	default:
		a := p.locateAnchor(ctx, cur, stats)
		if a.kind == anchorLost {
			p.noteAnchorLost(ctx, cur)
			resume.Start, resume.ExpectID, top.restart = 0, "", true
		} else {
			shift := shiftOf(a, stats.TopSpan)
			p.shiftCursor(ctx, cur, shift, top.clean)
			resume.Start = max(0, cur.ResumeOffset(shift)-backfillDefaultPageSize)
			resume.ExpectID = cur.LastMatchID
		}
		top.stop = top.inserted >= budget
	}
	resume.HeadID = p.head
	return top
}

// shiftOf rend le décalage du marque-page : N exact quand l'ancre est retrouvée, sinon fallback, le
// nombre d'entrées vues au-dessus du marque-page selon la règle sans ancre.
func shiftOf(a anchorResult, fallback int) int {
	if a.kind == anchorFound {
		return a.n
	}
	return fallback
}

// shiftCursor décale le marque-page de n entrées nouvelles au-dessus de lui, si le delta est propre, et
// y range l'ancre lue à cette passe (la tête de liste actuelle).
func (p *catchUpPass) shiftCursor(ctx context.Context, cur historystate.Cursor, n int, clean bool) {
	if n <= 0 || !clean {
		return
	}
	p.saveCursor(ctx, historystate.Cursor{Offset: cur.ResumeOffset(n), LastMatchID: cur.LastMatchID, HeadID: p.head})
}

// addCapture ajoute les compteurs d'une capture à un cumul.
func addCapture(dst *halo5.CaptureStats, s halo5.CaptureStats) {
	dst.MatchesSeen += s.MatchesSeen
	dst.MatchesCollected += s.MatchesCollected
	dst.MatchesSkipped += s.MatchesSkipped
	dst.EventsFailed += s.EventsFailed
	dst.CarnageFailed += s.CarnageFailed
	dst.ExcludedCampaign += s.ExcludedCampaign
	dst.ExcludedWarzone += s.ExcludedWarzone
	dst.RosterDropped += s.RosterDropped
	dst.StoppedOnKnown = dst.StoppedOnKnown || s.StoppedOnKnown
	dst.HistoryEnd = s.HistoryEnd
}
