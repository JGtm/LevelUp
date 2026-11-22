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
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	syncpkg "levelup/go-api/internal/sync"
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
)

// HistoryState est l'état de l'historique d'un joueur au début du cycle, lu avec l'ensemble connu.
// La valeur zéro dit « historique complet » : delta ordinaire.
type HistoryState struct {
	// Incomplete : la fin de l'historique n'a pas encore été vue (clé `historystate.Key` absente).
	Incomplete bool
	// Cursor : marque-page du rattrapage en cours ; nil = aucun (premier passage, valeur illisible
	// ou marque-page effacé). N'a de sens que si Incomplete.
	Cursor *historystate.Cursor
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
}

// historyTracker suit, pendant un cycle, les joueurs dont l'historique reste incomplet : valeur
// de PlayerOutcome.HistoryPending, y compris sur les sorties anticipées du cycle.
type historyTracker struct {
	pending map[string]bool // PlayerSlug -> historique incomplet en fin de cycle
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
		if o.settlePlayerHistory(ctx, p, ph, disc.PerPlayer[p.PlayerSlug], fetched.Errors) {
			tr.pending[p.PlayerSlug] = false
		}
	}
}

// settlePlayerHistory applique la règle d'un joueur : la fin d'historique se range seulement si la
// découverte l'a vue ET si chaque match listé est persisté ou introuvable ; sinon, si tous le sont
// et qu'une position est proposée, le marque-page avance ; sinon l'ancien marque-page reste. Rend
// vrai quand la fin d'historique est rangée.
func (o *CycleOrchestratorImpl) settlePlayerHistory(
	ctx context.Context,
	p PlayerProfile,
	ph PlayerHistory,
	listed []string,
	fetchErrs map[string]error,
) bool {
	if !ph.EndSeen && ph.Proposed == nil {
		return false // rien à ranger : budget atteint avant tout premier connu, ou borne de profondeur sans position
	}
	if o.historyWriter == nil {
		slog.WarnContext(ctx, "sync.v2: rattrapage d'historique sans écrivain câblé — rien n'est rangé",
			"event", "sync.v2.history.no_writer", "player", p.PlayerSlug)
		return false
	}
	if !o.listedResolved(ctx, p, listed, fetchErrs) {
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
	if ph.EndSeen {
		if !historystate.MarkComplete(wctx, db, p.Gamertag) {
			return false
		}
		observability.IncCounterT(title, counterHistoryComplete)
		return true
	}
	if historystate.SaveCursor(wctx, db, *ph.Proposed) {
		observability.IncCounterT(title, counterHistoryCursorSaved)
		slog.InfoContext(ctx, "sync.v2: rattrapage d'historique — marque-page rangé",
			"event", "sync.v2.history.cursor_saved", "player", p.PlayerSlug,
			"offset", ph.Proposed.Offset, "listed", len(listed))
	}
	return false
}

// listedResolved dit si chaque match listé pour le joueur est « résolu » : introuvable côté API
// (404/410, compté et journalisé) ou présent dans l'ensemble connu RELU après la persistance (preuve
// que le match est bien rangé : une erreur de lecture du match, de décodage ou de file d'écriture
// ne passe pas pour un succès). Un ensemble connu illisible vaut « non résolu ».
func (o *CycleOrchestratorImpl) listedResolved(
	ctx context.Context,
	p PlayerProfile,
	listed []string,
	fetchErrs map[string]error,
) bool {
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
	if len(toCheck) == 0 {
		return true
	}
	state, err := o.loader.LoadKnown(ctx, p)
	if err != nil {
		slog.WarnContext(ctx, "sync.v2: rattrapage d'historique — ensemble connu illisible après la persistance, rien n'est rangé",
			"event", "sync.v2.history.reload_failed", "player", p.PlayerSlug, "err", err)
		return false
	}
	unresolved := 0
	for _, id := range toCheck {
		if !state.Known[id] {
			unresolved++
		}
	}
	if unresolved > 0 {
		slog.WarnContext(ctx, "sync.v2: rattrapage d'historique — matchs listés non persistés, l'état n'avance pas (la passe suivante reprend)",
			"event", "sync.v2.history.unresolved", "player", p.PlayerSlug,
			"listed", len(listed), "unresolved", unresolved, "not_found", notFound)
		return false
	}
	return true
}
