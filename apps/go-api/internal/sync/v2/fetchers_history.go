// Package v2 — fetchers_history.go : découverte d'un joueur dont l'historique est incomplet.
//
// Mode rattrapage de matchListProviderV2.ListUnknownMatches (règle : history.go). Quand un
// marque-page existe, sa position dans la liste, qui a gagné des entrées par le haut depuis la passe
// précédente, se retrouve par l'ancre (`historystate.Cursor.HeadID`) : lecture du haut de la liste
// jusqu'à retrouver l'ancienne tête ; sa position N est le nombre d'entrées arrivées, TOUTES
// entrées comptées (connues, inconnues, introuvables). La reprise est à `cursor.Offset + N`, contrôlée par
// l'identifiant rangé en première entrée de la page lue : aucune page de recouvrement. Ancienne tête
// introuvable dans les `historystate.AnchorMaxPages` premières pages, ou identifiant rangé qui n'est pas à sa
// place : reprise depuis le début, journalisée et comptée (`sync_v2_history_cursor_fallback_total`).
// Un marque-page sans ancre vaut pas de marque-page (le cycle V2 range la tête avec chacun des siens) :
// reprise depuis le début, journalisée et comptée de même.
//
// Un match que la découverte ne demande plus (HistoryState.Skip : introuvable 404/410 ; abandonné par
// maxCatchUpAttempts, HistoryState.Failures) est neutre : sauté, sans compter comme nouveau, ni dans
// le budget. Ensuite la lecture saute les connus, garde les inconnus dans l'ordre de l'API jusqu'au budget
// du joueur (HistoryState.Budget, au plus `historystate.PassBudget()`, N compris), et s'arrête sur une
// page VIDE (fin vue : une page partielle ne dit rien, l'API peut en rendre une avant la fin), sur le
// budget, ou sur `historystate.MaxDepth` (avertissement).
package v2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

// catchUpScan est l'état d'une passe de rattrapage d'un joueur.
type catchUpScan struct {
	p        *matchListProviderV2
	client   HaloClient
	histArg  string
	gamertag string
	known    map[string]bool
	skip     map[string]bool // matchs que la découverte ne demande plus (introuvables, abandonnés) : ni retentés, ni comptés, ni « connus »
	budget   int

	unknown []string
	seen    map[string]bool

	// Dernière entrée de la liste lue (position absolue, identifiant) et dernier inconnu listé :
	// de quoi proposer un marque-page à l'arrêt.
	lastReadPos    int
	lastReadID     string
	lastUnknownPos int
	lastUnknownID  string
	// headID : identifiant de l'entrée en position 0 de la liste, quand la passe l'a lue ; il
	// devient l'ancre du marque-page proposé.
	headID string

	// arrived : les inconnus listés au-dessus de l'ancre du marque-page (les nouveaux matchs), une fois l'ancre retrouvée.
	arrived []string

	endSeen   bool // fin de la liste vue (page vide)
	budgetHit bool // budget atteint : la lecture s'est arrêtée avec des inconnus possibles derrière
	depthHit  bool // borne de profondeur atteinte
}

// listCatchUp est la découverte d'un joueur à l'historique incomplet.
func (p *matchListProviderV2) listCatchUp(
	ctx context.Context,
	client HaloClient,
	prof PlayerProfile,
	known map[string]bool,
	hist HistoryState,
) (Listing, error) {
	gamertag := prof.Gamertag
	sc := &catchUpScan{
		p: p, client: client, histArg: fmt.Sprintf("xuid(%s)", prof.XUID), gamertag: gamertag, known: known,
		skip: hist.skipped(), budget: hist.catchUpBudget(), seen: map[string]bool{},
		lastReadPos: -1, lastUnknownPos: -1,
	}
	at := resumeAt{}
	if cur := hist.Cursor; cur != nil && !cur.HasAnchor() {
		sc.noteCursorFallback(ctx, *cur, "marque-page sans ancre")
		hist.Cursor = nil
	}
	if hist.Cursor != nil {
		resume, proceed, err := sc.resumePoint(ctx, *hist.Cursor)
		if err != nil {
			return Listing{}, err
		}
		if !proceed {
			// Budget, fin ou borne atteints avant d'avoir situé le marque-page : son décalage
			// n'est pas translatable, aucune position n'est proposée.
			return sc.listing(false), nil
		}
		at = resume
	}
	if err := sc.scanFrom(ctx, at.start, at.guard); err != nil {
		if !errors.Is(err, errAnchorMismatch) {
			return Listing{}, err
		}
		sc.noteCursorFallback(ctx, *hist.Cursor, "identifiant du marque-page absent à sa position")
		at = resumeAt{}
		if err := sc.scanFrom(ctx, 0, ""); err != nil {
			return Listing{}, err
		}
	}
	start := at.start
	slog.DebugContext(ctx, "v2 ListUnknownMatches (rattrapage)",
		"gamertag", gamertag, "unknown_count", len(sc.unknown), "end_seen", sc.endSeen,
		"budget_hit", sc.budgetHit, "resume_from", start)
	return sc.listing(true), nil
}

// scanFrom lit la liste à partir de `start`, saute les connus et garde les inconnus jusqu'au
// budget, à la fin de la liste ou à la borne de profondeur. `guard` non vide : l'entrée à `start`
// doit porter cet identifiant (le dernier match traité du marque-page, déjà rangé) et la lecture
// commence à la suivante ; sinon errAnchorMismatch, sans rien avoir listé.
func (sc *catchUpScan) scanFrom(ctx context.Context, start int, guard string) error {
	if sc.endSeen || sc.budgetHit || sc.depthHit {
		return nil
	}
	for pos := start; ; pos += sc.p.pageSize {
		if pos >= historystate.MaxDepth {
			sc.warnDepth(ctx)
			return nil
		}
		entries, err := sc.page(ctx, pos)
		if err != nil {
			return err
		}
		pageLen, first := len(entries), pos
		if guard != "" {
			if pageLen == 0 || entries[0].MatchID != guard {
				return errAnchorMismatch
			}
			entries, first, guard = entries[1:], pos+1, ""
		}
		if pageLen == 0 {
			sc.endSeen = true
			return nil
		}
		for i, e := range entries {
			sc.noteRead(first+i, e.MatchID)
			if sc.known[e.MatchID] || sc.seen[e.MatchID] || sc.skip[e.MatchID] {
				continue
			}
			if sc.addUnknown(first+i, e.MatchID) {
				return nil
			}
		}
	}
}

// page lit une page de la liste de l'API à la position `pos`.
func (sc *catchUpScan) page(ctx context.Context, pos int) ([]syncpkg.MatchHistoryEntry, error) {
	entries, err := sc.client.GetMatchHistory(ctx, sc.histArg, sc.p.matchType, pos, sc.p.pageSize)
	if err != nil {
		return nil, fmt.Errorf("GetMatchHistory offset %d for %s: %w", pos, sc.gamertag, err)
	}
	return entries, nil
}

// noteRead retient la dernière entrée lue.
func (sc *catchUpScan) noteRead(pos int, id string) {
	sc.lastReadPos, sc.lastReadID = pos, id
	if pos == 0 {
		sc.headID = id
	}
}

// addUnknown ajoute un inconnu à la liste et rend vrai quand le budget est atteint (la lecture
// s'arrête alors).
func (sc *catchUpScan) addUnknown(pos int, id string) bool {
	sc.unknown = append(sc.unknown, id)
	sc.seen[id] = true
	sc.lastUnknownPos, sc.lastUnknownID = pos, id
	if len(sc.unknown) >= sc.budget {
		sc.budgetHit = true
		return true
	}
	return false
}

// warnDepth journalise l'arrêt sur la borne de profondeur.
func (sc *catchUpScan) warnDepth(ctx context.Context) {
	sc.depthHit = true
	slog.WarnContext(ctx, "sync.v2: rattrapage d'historique arrêté à la borne de profondeur — fin non rangée",
		"event", "sync.v2.history.max_depth", "gamertag", sc.gamertag, "max_depth", historystate.MaxDepth)
}

// listing assemble le résultat. withProposal=false : aucune position n'est proposée.
func (sc *catchUpScan) listing(withProposal bool) Listing {
	out := Listing{Unknown: sc.unknown, EndSeen: sc.endSeen && !sc.budgetHit && !sc.depthHit, Arrived: sc.arrived}
	if !withProposal || out.EndSeen {
		return out
	}
	switch {
	case sc.budgetHit && sc.lastUnknownPos >= 0:
		// Le budget a coupé la lecture : la position fiable est juste après le dernier inconnu listé.
		out.Proposed = &historystate.Cursor{Offset: sc.lastUnknownPos + 1, LastMatchID: sc.lastUnknownID, HeadID: sc.headID}
	case sc.depthHit && sc.lastReadPos >= 0:
		out.Proposed = &historystate.Cursor{Offset: sc.lastReadPos + 1, LastMatchID: sc.lastReadID, HeadID: sc.headID}
	}
	return out
}
