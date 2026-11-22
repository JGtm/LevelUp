// Package v2 — history_settle.go : règlement, après la persistance du cycle, de l'état de
// rattrapage d'un joueur (fin d'historique, marque-page, matchs en échec).
//
// Règle d'un joueur au rattrapage dont la découverte a abouti :
//
//   - chaque match listé est RÉSOLU (introuvable 404/410, ou présent dans l'ensemble connu relu après
//     la persistance) ou en ÉCHEC ; un échec compte une passe dans la liste des échecs rangée avec le
//     marque-page ; un match résolu sort de cette liste ;
//   - à maxCatchUpAttempts passes en échec, le match est ABANDONNÉ : journalisé et compté une fois, il
//     ne bloque plus rien (la découverte ne le liste plus : history.go) ;
//   - la fin d'historique se range quand la fin est vue et qu'il ne reste aucun échec non abandonné ;
//   - sinon le marque-page avance (position proposée par la découverte) quand il ne reste aucun échec
//     non abandonné, et la liste des échecs est rangée si elle a changé.
package v2

import (
	"context"
	"log/slog"
	"slices"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

// notPersistedReason est la raison journalisée d'un match listé, récupéré sans erreur, que la
// relecture de l'ensemble connu ne retrouve pas.
const notPersistedReason = "non persisté (absent de l'ensemble connu relu après la persistance)"

// settlePlayerHistory range l'état de rattrapage d'un joueur (règle du fichier). Rend vrai quand la
// fin d'historique est rangée.
func (o *CycleOrchestratorImpl) settlePlayerHistory(
	ctx context.Context,
	p PlayerProfile,
	ph PlayerHistory,
	listed []string,
	fetchErrs map[string]error,
) bool {
	unresolved, ok := o.unresolvedListed(ctx, p, listed, fetchErrs)
	if !ok {
		return false
	}
	failures, blocking := trackFailures(ctx, p, ph.Failures, passOutcome{listed: listed, unresolved: unresolved, fetchErrs: fetchErrs})
	canAdvance := blocking == 0
	complete := canAdvance && ph.EndSeen
	saveCursor := !complete && canAdvance && ph.Proposed != nil
	saveFailures := !complete && !slices.Equal(failures, ph.Failures)
	if !complete && !saveCursor && !saveFailures {
		return false
	}
	if o.historyWriter == nil {
		slog.WarnContext(ctx, "sync.v2: rattrapage d'historique sans écrivain câblé — rien n'est rangé",
			"event", "sync.v2.history.no_writer", "player", p.PlayerSlug)
		return false
	}
	wctx, cancel := context.WithTimeout(ctx, historyWriteWait)
	defer cancel()
	db, release, err := o.historyWriter(wctx, p.Gamertag)
	if err != nil {
		slog.WarnContext(ctx, "sync.v2: base joueur indisponible en écriture — état de l'historique non rangé",
			"event", "sync.v2.history.write_unavailable", "player", p.PlayerSlug, "err", err)
		return false
	}
	defer release()

	title := ctxkeys.TitleSlug(ctx)
	if complete {
		slog.InfoContext(ctx, "sync.v2: fin de l'historique vue — rangement",
			"event", "sync.v2.history.complete", "player", p.PlayerSlug, "abandoned", countAbandoned(failures))
		if !historystate.MarkComplete(wctx, db, p.Gamertag) {
			return false
		}
		observability.IncCounterT(title, counterHistoryComplete)
		return true
	}
	if saveFailures {
		historystate.SaveFailures(wctx, db, failures)
	}
	if saveCursor && historystate.SaveCursor(wctx, db, *ph.Proposed) {
		observability.IncCounterT(title, counterHistoryCursorSaved)
		slog.InfoContext(ctx, "sync.v2: rattrapage d'historique — marque-page rangé",
			"event", "sync.v2.history.cursor_saved", "player", p.PlayerSlug,
			"offset", ph.Proposed.Offset, "listed", len(listed))
	}
	return false
}

// unresolvedListed rend l'ensemble des matchs listés pour le joueur qui ne sont NI introuvables côté
// API (404/410, comptés et journalisés : ils ne comptent jamais comme échec) NI présents dans
// l'ensemble connu RELU après la persistance (preuve que le match est bien rangé : une erreur de
// lecture du match, de décodage ou de file d'écriture ne passe pas pour un succès). ok=false : ensemble
// connu illisible, rien ne peut être conclu et l'état n'est pas touché.
func (o *CycleOrchestratorImpl) unresolvedListed(
	ctx context.Context,
	p PlayerProfile,
	listed []string,
	fetchErrs map[string]error,
) (unresolved map[string]bool, ok bool) {
	toCheck := make([]string, 0, len(listed))
	notFound := 0
	for _, id := range listed {
		if err, failed := fetchErrs[id]; failed && syncpkg.IsNotFound(err) {
			notFound++
			continue
		}
		toCheck = append(toCheck, id)
	}
	if notFound > 0 {
		observability.AddIntT(ctxkeys.TitleSlug(ctx), counterHistoryNotFound, int64(notFound))
		slog.InfoContext(ctx, "sync.v2: rattrapage d'historique — matchs introuvables côté API (404/410), non bloquants",
			"event", "sync.v2.history.not_found", "player", p.PlayerSlug, "not_found", notFound)
	}
	unresolved = map[string]bool{}
	if len(toCheck) == 0 {
		return unresolved, true
	}
	state, err := o.loader.LoadKnown(ctx, p)
	if err != nil {
		slog.WarnContext(ctx, "sync.v2: rattrapage d'historique — ensemble connu illisible après la persistance, rien n'est rangé",
			"event", "sync.v2.history.reload_failed", "player", p.PlayerSlug, "err", err)
		return nil, false
	}
	for _, id := range toCheck {
		if !state.Known[id] {
			unresolved[id] = true
		}
	}
	return unresolved, true
}

// passOutcome est le bilan d'une passe pour un joueur : matchs listés, ceux qui ne sont ni résolus ni
// introuvables, et les erreurs de récupération par match.
type passOutcome struct {
	listed     []string
	unresolved map[string]bool
	fetchErrs  map[string]error
}

// trackFailures met à jour la liste des échecs d'un joueur après une passe et rend la nouvelle liste
// (du plus ancien échec au plus récent) et le nombre de matchs en échec NON abandonnés (ceux qui
// retiennent la fin d'historique et le marque-page).
//
//   - un échec dont le match n'était pas listé cette passe reste tel quel ;
//   - un échec dont le match est résolu sort de la liste ;
//   - un échec dont le match est encore non résolu gagne une passe (sauf s'il est déjà abandonné : un
//     orphelin de l'ensemble connu peut être listé de nouveau sans pour autant bloquer) ;
//   - un match listé non résolu et jamais échoué entre dans la liste à une passe.
//
// Le passage au seuil maxCatchUpAttempts abandonne le match : journalisé (dernière erreur) et compté.
func trackFailures(
	ctx context.Context,
	p PlayerProfile,
	prev []historystate.Failure,
	pass passOutcome,
) (next []historystate.Failure, blocking int) {
	listed, unresolved, fetchErrs := pass.listed, pass.unresolved, pass.fetchErrs
	listedSet := make(map[string]bool, len(listed))
	for _, id := range listed {
		listedSet[id] = true
	}
	known := make(map[string]bool, len(prev))
	next = make([]historystate.Failure, 0, len(prev)+len(unresolved))
	for _, f := range prev {
		known[f.MatchID] = true
		switch {
		case !listedSet[f.MatchID]:
			next = append(next, f)
		case !unresolved[f.MatchID]:
			// résolu : sort de la liste
		case isAbandoned(f.Attempts):
			next = append(next, f)
		default:
			f.Attempts++
			next = append(next, f)
			if !noteAttempt(ctx, p, f, fetchErrs) {
				blocking++
			}
		}
	}
	for _, id := range listed {
		if unresolved[id] && !known[id] {
			known[id] = true
			f := historystate.Failure{MatchID: id, Attempts: 1}
			next = append(next, f)
			if !noteAttempt(ctx, p, f, fetchErrs) {
				blocking++
			}
		}
	}
	return next, blocking
}

// noteAttempt journalise et compte l'abandon d'un match qui atteint le seuil à cette passe, et dit
// s'il est abandonné (vrai : il ne bloque plus).
func noteAttempt(ctx context.Context, p PlayerProfile, f historystate.Failure, fetchErrs map[string]error) bool {
	if !isAbandoned(f.Attempts) {
		return false
	}
	reason := notPersistedReason
	if err := fetchErrs[f.MatchID]; err != nil {
		reason = err.Error()
	}
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), counterHistoryAbandoned)
	slog.WarnContext(ctx, "sync.v2: rattrapage d'historique — match abandonné après plusieurs passes en échec",
		"event", "sync.v2.history.abandoned", "player", p.PlayerSlug, "match_id", f.MatchID,
		"attempts", f.Attempts, "err", reason)
	return true
}

// countAbandoned compte les matchs abandonnés d'une liste d'échecs.
func countAbandoned(failures []historystate.Failure) int {
	n := 0
	for _, f := range failures {
		if isAbandoned(f.Attempts) {
			n++
		}
	}
	return n
}
