// Package v2 — history.go : rattrapage de l'historique d'un joueur par le cycle V2.
//
// Règle (partagée avec le moteur V1, source unique `internal/sync/historystate`) : tant que la fin
// de l'historique n'a pas été vue pour un joueur, la découverte ne s'arrête plus au premier match
// connu. Elle saute les connus et garde les inconnus plus anciens par tranches de
// `historystate.PassBudget()` matchs ; un marque-page (`historystate.Cursor`) évite de reparcourir
// la liste de l'API depuis le début à chaque passe.
//
// Ce fichier porte les types échangés entre la découverte (Phase 1) et le cycle, et l'étape qui
// range, APRÈS la persistance du cycle, la fin d'historique ou le marque-page d'un joueur.
package v2

import (
	"context"
	"time"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/knownset"
)

// historyWriteWait borne l'attente du bail d'écriture du joueur pour ranger la fin d'historique ou
// le marque-page : au-delà, la passe suivante reprend (les matchs déjà persistés sont sautés).
const historyWriteWait = 30 * time.Second

// Compteurs expvar titrés du rattrapage V2.
const (
	// counterHistoryNotFound : matchs listés que l'API a rendus introuvables (404/410) à la
	// récupération ; ils ne bloquent pas la fin d'historique.
	counterHistoryNotFound = "sync_v2_history_not_found_total"
	// counterHistoryCursorFallback : reprises depuis le début de la liste faute de retrouver le
	// dernier match du marque-page dans la page de recouvrement.
	counterHistoryCursorFallback = "sync_v2_history_cursor_fallback_total"
	// counterHistoryComplete : fins d'historique rangées par le cycle.
	counterHistoryComplete = "sync_v2_history_complete_total"
	// counterHistoryCursorSaved : marque-pages rangés par le cycle.
	counterHistoryCursorSaved = "sync_v2_history_cursor_saved_total"
	// counterHistoryAbandoned : matchs abandonnés par le rattrapage après maxCatchUpAttempts passes
	// en échec, comptés au moment de l'abandon.
	counterHistoryAbandoned = "sync_v2_history_abandoned_total"
)

// maxCatchUpAttempts est le nombre de passes de rattrapage en échec au bout duquel un match est
// abandonné : il n'est plus retenté par le rattrapage, ne compte plus dans le budget et ne retient
// plus la fin d'historique. Un échec compte une passe où le match était listé pour le joueur et n'a
// été ni persisté ni rendu introuvable (404/410, qui ne comptent jamais comme échec).
const maxCatchUpAttempts = 3

// isAbandoned dit si un match en échec à `attempts` passes est abandonné.
func isAbandoned(attempts int) bool { return attempts >= maxCatchUpAttempts }

// HistoryState est l'état de l'historique d'un joueur au début du cycle, lu avec l'ensemble connu.
// La valeur zéro dit « historique complet » : delta ordinaire.
type HistoryState struct {
	// Incomplete : la fin de l'historique n'a pas encore été vue (clé `historystate.Key` absente).
	Incomplete bool
	// Cursor : marque-page du rattrapage en cours ; nil = aucun (premier passage, valeur illisible
	// ou marque-page effacé). N'a de sens que si Incomplete.
	Cursor *historystate.Cursor
	// Failures : les matchs que le rattrapage n'a pas réussi à ranger, avec leur nombre de passes en
	// échec (du plus ancien au plus récent). N'a de sens que si Incomplete.
	Failures []historystate.Failure
}

// abandoned rend l'ensemble des matchs que le rattrapage ne retente plus (échec à
// maxCatchUpAttempts passes ou plus).
func (h HistoryState) abandoned() map[string]bool {
	var out map[string]bool
	for _, f := range h.Failures {
		if isAbandoned(f.Attempts) {
			if out == nil {
				out = map[string]bool{}
			}
			out[f.MatchID] = true
		}
	}
	return out
}

// KnownState est ce que rend KnownLoader : l'ensemble connu du joueur et l'état de son historique,
// lus sur la même base joueur ouverte.
type KnownState struct {
	knownset.Set
	History HistoryState
}

// Listing est le résultat de la découverte d'un joueur par MatchListProvider.
type Listing struct {
	// Unknown : les matchs inconnus, en ordre de l'API (du plus récent au plus ancien), sans les
	// orphelins de l'ensemble connu (ajoutés ensuite par RunDiscovery).
	Unknown []string
	// EndSeen : la liste de l'API a été parcourue jusqu'à sa fin (page vide ou partielle) sans
	// s'arrêter sur le budget ni sur la borne de profondeur. Toujours faux en mode delta.
	EndSeen bool
	// Proposed : marque-page à ranger si tous les inconnus listés sont résolus ; nil = pas de
	// position fiable à proposer (mode delta, fin vue, ou arrêt avant le premier match connu).
	Proposed *historystate.Cursor
}

// PlayerHistory est ce que la découverte retient de l'historique d'un joueur pour le cycle.
// Présent dans DiscoveryResult.History pour tout joueur dont l'ensemble connu a été lu.
type PlayerHistory struct {
	Incomplete bool
	EndSeen    bool
	Proposed   *historystate.Cursor
	// Failures : les échecs de rattrapage connus au début du cycle (voir HistoryState.Failures).
	Failures []historystate.Failure
}

// historyTracker suit, pendant un cycle, les joueurs dont l'historique reste incomplet : valeur
// de PlayerOutcome.HistoryPending, y compris sur les sorties anticipées du cycle.
type historyTracker struct {
	pending map[string]bool // PlayerSlug -> historique incomplet en fin de cycle
	// queued : matchs soumis à la file d'écriture dont le rangement n'est pas encore prouvé
	// (attente expirée, persist_pending.go) ; ni rangés ni échecs, ils retiennent la fin et le
	// marque-page.
	queued map[string]bool
}

// newHistoryTracker part de l'état lu au début du cycle : incomplet = en attente.
func newHistoryTracker(disc DiscoveryResult) *historyTracker {
	t := &historyTracker{pending: make(map[string]bool, len(disc.History))}
	for slug, h := range disc.History {
		t.pending[slug] = h.Incomplete
	}
	return t
}

// apply recopie l'état sur les issues par joueur déjà créées.
func (t *historyTracker) apply(per map[string]PlayerOutcome) {
	for slug, pending := range t.pending {
		if out, ok := per[slug]; ok {
			out.HistoryPending = pending
			per[slug] = out
		}
	}
}

// WithHistoryWriter câble l'ouverture de la base joueur en ÉCRITURE sous le bail d'écriture du
// joueur (celle du post-sync : cmd/server `postSyncPlayerOpener`), par laquelle le cycle range la
// fin d'historique et le marque-page. Sans elle, aucun joueur ne sort du rattrapage : le câblage
// de production la pose toujours (garde `cmd/server/sync_v2_wiring_history_test.go`). Retourne
// l'orchestrator pour chaînage.
func (o *CycleOrchestratorImpl) WithHistoryWriter(open PlayerDBOpener) *CycleOrchestratorImpl {
	o.historyWriter = open
	return o
}

// settleHistory range, pour chaque joueur au rattrapage dont la découverte a abouti, la fin
// d'historique ou le marque-page. À appeler APRÈS une persistance du cycle réussie.
func (o *CycleOrchestratorImpl) settleHistory(
	ctx context.Context,
	tr *historyTracker,
	players []PlayerProfile,
	disc DiscoveryResult,
	fetched FetchSharedResult,
) {
	for _, p := range players {
		ph, ok := disc.History[p.PlayerSlug]
		if !ok || !ph.Incomplete {
			continue
		}
		if disc.Errors[p.PlayerSlug] != nil {
			continue // découverte en échec : rien n'a été listé, la passe suivante reprend
		}
		in := settleInput{listed: disc.PerPlayer[p.PlayerSlug], fetchErrs: fetched.Errors, queued: tr.queued}
		if o.settlePlayerHistory(ctx, p, ph, in) {
			tr.pending[p.PlayerSlug] = false
		}
	}
}
