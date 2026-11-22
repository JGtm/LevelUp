// Package v2 — fetchers_history.go : découverte d'un joueur dont l'historique est incomplet.
//
// Mode rattrapage de matchListProviderV2.ListUnknownMatches (règle : history.go). Les deux étapes
// quand un marque-page existe, une seule sinon :
//
//	(a) delta du dessus : page 0 jusqu'au premier match connu ; les N inconnus rencontrés sont les
//	    matchs apparus depuis la dernière passe ;
//	(b) reprise : lecture à `cursor.Offset + N - pageSize` (borné à 0), soit la page de
//	    recouvrement qui doit contenir `cursor.LastMatchID` (la liste a gagné N entrées par le haut) ;
//	    sinon la position n'est plus fiable et la lecture repart du début, ce que le journal et le
//	    compteur `sync_v2_history_cursor_fallback_total` disent.
//
// Ensuite la lecture saute les connus, garde les inconnus dans l'ordre de l'API jusqu'au budget
// (`historystate.PassBudget()`, N compris), et s'arrête sur une page vide ou partielle (fin vue),
// sur le budget, ou sur `historystate.MaxDepth` (avertissement).
package v2

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
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
	budget   int

	unknown []string
	seen    map[string]bool

	// Dernière entrée de la liste lue (position absolue, identifiant) et dernier inconnu listé :
	// de quoi proposer un marque-page à l'arrêt.
	lastReadPos    int
	lastReadID     string
	lastUnknownPos int
	lastUnknownID  string

	endSeen      bool // fin de la liste vue (page vide ou partielle)
	budgetHit    bool // budget atteint : la lecture s'est arrêtée avec des inconnus possibles derrière
	depthHit     bool // borne de profondeur atteinte
	reachedKnown bool // l'étape (a) a rencontré un match connu
}

// listCatchUp est la découverte d'un joueur à l'historique incomplet.
func (p *matchListProviderV2) listCatchUp(
	ctx context.Context,
	client HaloClient,
	gamertag, histArg string,
	known map[string]bool,
	cursor *historystate.Cursor,
) (Listing, error) {
	sc := &catchUpScan{
		p: p, client: client, histArg: histArg, gamertag: gamertag, known: known,
		budget: historystate.PassBudget(), seen: map[string]bool{},
		lastReadPos: -1, lastUnknownPos: -1,
	}
	start := 0
	if cursor != nil {
		if err := sc.scanNew(ctx); err != nil {
			return Listing{}, err
		}
		if !sc.reachedKnown {
			// Budget, fin ou borne atteints avant le premier connu : le décalage du marque-page
			// n'est pas translatable, aucune position n'est proposée.
			return sc.listing(false), nil
		}
		resume, err := sc.resumeStart(ctx, *cursor)
		if err != nil {
			return Listing{}, err
		}
		start = resume
	}
	if err := sc.scanFrom(ctx, start); err != nil {
		return Listing{}, err
	}
	slog.DebugContext(ctx, "v2 ListUnknownMatches (rattrapage)",
		"gamertag", gamertag, "unknown_count", len(sc.unknown), "end_seen", sc.endSeen,
		"budget_hit", sc.budgetHit, "resume_from", start)
	return sc.listing(true), nil
}

// scanNew est l'étape (a) : du début de la liste jusqu'au premier match connu. Les inconnus
// rencontrés comptent dans le budget.
func (sc *catchUpScan) scanNew(ctx context.Context) error {
	for pos := 0; ; pos += sc.p.pageSize {
		if pos >= historystate.MaxDepth {
			sc.warnDepth(ctx)
			return nil
		}
		entries, err := sc.page(ctx, pos)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			sc.endSeen = true
			return nil
		}
		for i, e := range entries {
			sc.noteRead(pos+i, e.MatchID)
			if sc.known[e.MatchID] {
				sc.reachedKnown = true
				return nil
			}
			if sc.addUnknown(pos+i, e.MatchID) {
				return nil
			}
		}
		if len(entries) < sc.p.pageSize {
			sc.endSeen = true
			return nil
		}
	}
}

// resumeStart est l'étape (b) : décalage de la page de recouvrement, vérifié par lecture. Rend 0
// (reprise depuis le début) quand la page ne contient pas le dernier match du marque-page.
func (sc *catchUpScan) resumeStart(ctx context.Context, cursor historystate.Cursor) (int, error) {
	start := cursor.Offset + len(sc.unknown) - sc.p.pageSize
	if start <= 0 {
		return 0, nil // la page de recouvrement couvre le début : la lecture ordinaire la verra
	}
	entries, err := sc.page(ctx, start)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.MatchID == cursor.LastMatchID {
			return start, nil
		}
	}
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), counterHistoryCursorFallback)
	slog.WarnContext(ctx, "sync.v2: marque-page d'historique introuvable dans la page de recouvrement — reprise depuis le début de la liste",
		"event", "sync.v2.history.cursor_fallback", "gamertag", sc.gamertag,
		"offset", cursor.Offset, "new_matches", len(sc.unknown), "resume_from", start)
	return 0, nil
}

// scanFrom lit la liste à partir de `start`, saute les connus et garde les inconnus jusqu'au
// budget, à la fin de la liste ou à la borne de profondeur.
func (sc *catchUpScan) scanFrom(ctx context.Context, start int) error {
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
		if len(entries) == 0 {
			sc.endSeen = true
			return nil
		}
		for i, e := range entries {
			sc.noteRead(pos+i, e.MatchID)
			if sc.known[e.MatchID] || sc.seen[e.MatchID] {
				continue
			}
			if sc.addUnknown(pos+i, e.MatchID) {
				return nil
			}
		}
		if len(entries) < sc.p.pageSize {
			sc.endSeen = true
			return nil
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
	out := Listing{Unknown: sc.unknown, EndSeen: sc.endSeen && !sc.budgetHit && !sc.depthHit}
	if !withProposal || out.EndSeen {
		return out
	}
	switch {
	case sc.budgetHit && sc.lastUnknownPos >= 0:
		// Le budget a coupé la lecture : la position fiable est juste après le dernier inconnu listé.
		out.Proposed = &historystate.Cursor{Offset: sc.lastUnknownPos + 1, LastMatchID: sc.lastUnknownID}
	case sc.depthHit && sc.lastReadPos >= 0:
		out.Proposed = &historystate.Cursor{Offset: sc.lastReadPos + 1, LastMatchID: sc.lastReadID}
	}
	return out
}
