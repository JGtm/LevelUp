// Package v2 — history_settle.go : règlement, après la persistance du cycle, de l'état de
// rattrapage d'un joueur (fin d'historique, marque-page, matchs en échec).
//
// Règle d'un joueur au rattrapage dont la découverte a abouti :
//
//   - chaque match listé est RÉSOLU (introuvable 404/410, ou présent dans l'ensemble connu relu après
//     la persistance) ou en ÉCHEC ; un échec compte une passe dans la liste des échecs rangée avec le
//     marque-page ; un match résolu sort de cette liste ;
//   - à maxCatchUpAttempts passes en échec, le match est ABANDONNÉ : journalisé et compté une fois, il
//     ne bloque plus rien (la découverte ne le liste plus : history.go) ; il est retenté après
//     historystate.AbandonRetryAfter (retry.go), tentatives remises à zéro ;
//   - une panne de l'API ne compte aucune tentative : un échec passager (429, réseau, délai) ne compte
//     jamais, et dans une passe de masse (historyfail.Wide, un 5xx général compris) aucun échec ne compte. Le match retient alors
//     la passe sans entrer dans la liste des échecs : la passe suivante le retente ;
//   - un match encore dans la file d'écriture (attente expirée, persist_pending.go) n'est ni résolu ni un
//     échec : il ne compte aucune passe, et retient la fin et le marque-page jusqu'à la passe suivante ;
//   - la fin d'historique se range quand la fin est vue et qu'il ne reste aucun échec non abandonné ;
//   - sinon le marque-page avance (position proposée par la découverte) quand il ne reste aucun échec
//     non abandonné, et la liste des échecs est rangée si elle a changé ;
//   - les matchs introuvables et les matchs abandonnés entrent dans l'ensemble des matchs à ne plus
//     demander (history_skip.go), rangé AVANT la fin d'historique : la fin efface la liste des échecs,
//     pas cet ensemble.
package v2

import (
	"context"
	"log/slog"
	"slices"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historyfail"
	"levelup/go-api/internal/sync/historystate"
)

// notPersistedReason est la raison journalisée d'un match listé, récupéré sans erreur, que la
// relecture de l'ensemble connu ne retrouve pas.
const notPersistedReason = "non persisté (absent de l'ensemble connu relu après la persistance)"

// settleInput est ce que le cycle sait d'un joueur au rattrapage après la persistance : ses matchs
// listés, les erreurs de récupération par match, et les matchs encore dans la file d'écriture.
type settleInput struct {
	listed    []string
	fetchErrs map[string]error
	queued    map[string]bool
	stored    playerStored // rangement prouvé par la relecture de l'ensemble connu (cycle_stored.go)
}

// withoutQueued rend les matchs listés qui ne sont pas dans la file d'écriture, et le nombre de ceux
// qui y sont.
func (in settleInput) withoutQueued() (listed []string, queued int) {
	if len(in.queued) == 0 {
		return in.listed, 0
	}
	listed = make([]string, 0, len(in.listed))
	for _, id := range in.listed {
		if in.queued[id] {
			queued++
			continue
		}
		listed = append(listed, id)
	}
	return listed, queued
}

// settlePlayerHistory range l'état de rattrapage d'un joueur (règle du fichier). Rend vrai quand la
// fin d'historique est rangée.
func (o *CycleOrchestratorImpl) settlePlayerHistory(
	ctx context.Context,
	p PlayerProfile,
	ph PlayerHistory,
	in settleInput,
) bool {
	listed, queued := in.withoutQueued()
	unresolved, ok := unresolvedListed(ctx, p, listed, in)
	if !ok {
		return false
	}
	failures, blocking := trackFailures(ctx, p, ph.Failures, passOutcome{
		listed: listed, unresolved: unresolved, fetchErrs: in.fetchErrs,
		due: toSet(ph.Retry), wide: wideFailure(unresolved, len(listed), ph.catchUpBudget()),
	})
	canAdvance := blocking == 0 && queued == 0
	complete := canAdvance && ph.EndSeen
	saveCursor := !complete && canAdvance && ph.Proposed != nil
	saveFailures := !complete && !slices.Equal(failures, ph.Failures)
	skips := freshSkips(ph.Skip, notFoundListed(listed, in.fetchErrs), abandonedIDs(failures))
	if !complete && !saveCursor && !saveFailures && len(skips) == 0 {
		return false
	}
	wctx, db, done, ok := o.openHistoryDB(ctx, p)
	if !ok {
		return false
	}
	defer done()

	// Les matchs à ne plus demander d'abord : la fin d'historique efface la liste des échecs (dont les
	// abandonnés), l'ensemble rangé à part les garde. Rangement en échec : la fin n'est pas rangée, les
	// abandonnés restent dans la liste des échecs et la passe suivante recommence.
	if len(skips) > 0 && !addSkips(wctx, db, p, skips) && complete {
		return false
	}
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
			"offset", ph.Proposed.Offset, "listed", len(in.listed))
	}
	return false
}

// unresolvedListed rend l'ensemble des matchs listés pour le joueur qui ne sont NI introuvables côté
// API (404/410, comptés et journalisés : ils ne comptent jamais comme échec) NI présents dans
// l'ensemble connu RELU après la persistance (in.stored, cycle_stored.go : preuve que le match est
// bien rangé ; une erreur de lecture du match, de décodage ou de file d'écriture ne passe pas pour un
// succès). ok=false : ensemble connu illisible, rien ne peut être conclu et l'état n'est pas touché.
func unresolvedListed(ctx context.Context, p PlayerProfile, listed []string, in settleInput) (unresolved map[string]bool, ok bool) {
	if notFound := len(notFoundListed(listed, in.fetchErrs)); notFound > 0 {
		observability.AddIntT(ctxkeys.TitleSlug(ctx), counterHistoryNotFound, int64(notFound))
		slog.InfoContext(ctx, "sync.v2: rattrapage d'historique — matchs introuvables côté API (404/410), non bloquants",
			"event", "sync.v2.history.not_found", "player", p.PlayerSlug, "not_found", notFound)
	}
	return storedMissing(listed, in)
}

// storedMissing rend les matchs listés qui ne sont ni introuvables (404/410) ni prouvés rangés par la
// relecture de l'ensemble connu. ok=false : relecture en échec (journalisée par proveStored), rien
// ne peut être conclu.
func storedMissing(listed []string, in settleInput) (missing map[string]bool, ok bool) {
	toCheck := make([]string, 0, len(listed))
	for _, id := range listed {
		if !isNotFoundFetch(in.fetchErrs, id) {
			toCheck = append(toCheck, id)
		}
	}
	missing = map[string]bool{}
	if len(toCheck) == 0 {
		return missing, true
	}
	if !in.stored.read {
		return nil, false
	}
	for _, id := range toCheck {
		if !in.stored.known[id] {
			missing[id] = true
		}
	}
	return missing, true
}

// wideFailure dit si les matchs en échec d'une passe (`failed`, parmi `listed` matchs listés, pour un budget
// de passe `budget`) sont ceux d'une panne de l'API plutôt que de matchs précis (historyfail.Wide).
func wideFailure(failed map[string]bool, listed, budget int) bool {
	return historyfail.Wide(len(failed), listed, budget)
}

// passOutcome est le bilan d'une passe pour un joueur : matchs listés, ceux qui ne sont ni résolus ni
// introuvables, et les erreurs de récupération par match.
type passOutcome struct {
	listed     []string
	unresolved map[string]bool
	fetchErrs  map[string]error
	// due : les abandons échus (HistoryState.Retry) : leurs tentatives repartent de zéro.
	due map[string]bool
	// wide : passe de masse (wideFailure) : aucun échec ne compte une tentative.
	wide bool
}

// counts dit si l'échec du match compte une tentative vers l'abandon (historyfail.Counts).
func (o passOutcome) counts(id string) bool { return historyfail.Counts(o.wide, o.fetchErrs[id]) }

// trackFailures met à jour la liste des échecs d'un joueur après une passe et rend la nouvelle liste
// (du plus ancien échec au plus récent) et le nombre de matchs en échec NON abandonnés (ceux qui
// retiennent la fin d'historique et le marque-page).
//
//   - un échec dont le match n'était pas listé cette passe reste tel quel ;
//   - un échec dont le match est résolu sort de la liste ;
//   - un échec dont le match est encore non résolu gagne une passe (sauf s'il est déjà abandonné : un
//     orphelin de l'ensemble connu peut être listé de nouveau sans pour autant bloquer) ;
//   - un match listé non résolu et jamais échoué entre dans la liste à une passe ;
//   - un échec qui ne compte pas (pass.counts : panne de l'API) ne change pas la liste mais retient la passe ;
//   - un match abandonné dont l'abandon est échu (pass.due) sort de la liste : ses tentatives repartent
//     de zéro, il redevient un inconnu ordinaire.
//
// Le passage au seuil maxCatchUpAttempts abandonne le match : journalisé (dernière erreur) et compté.
func trackFailures(
	ctx context.Context,
	p PlayerProfile,
	prev []historystate.Failure,
	pass passOutcome,
) (next []historystate.Failure, blocking int) {
	listedSet := toSet(pass.listed)
	known := make(map[string]bool, len(prev))
	next = make([]historystate.Failure, 0, len(prev)+len(pass.unresolved))
	for _, f := range prev {
		if isAbandoned(f.Attempts) && pass.due[f.MatchID] {
			continue
		}
		known[f.MatchID] = true
		switch {
		case !listedSet[f.MatchID]:
			next = append(next, f)
		case !pass.unresolved[f.MatchID]:
			// résolu : sort de la liste
		case isAbandoned(f.Attempts):
			next = append(next, f)
		case !pass.counts(f.MatchID):
			next = append(next, f)
			blocking++
		default:
			f.Attempts++
			next = append(next, f)
			if !noteAttempt(ctx, p, f, pass.fetchErrs) {
				blocking++
			}
		}
	}
	for _, id := range pass.listed {
		if !pass.unresolved[id] || known[id] {
			continue
		}
		known[id] = true
		if !pass.counts(id) {
			blocking++
			continue
		}
		f := historystate.Failure{MatchID: id, Attempts: 1}
		next = append(next, f)
		if !noteAttempt(ctx, p, f, pass.fetchErrs) {
			blocking++
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
