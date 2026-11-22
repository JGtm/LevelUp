// Package v2 — fetchers_history_anchor.go : situer le marque-page d'un rattrapage dans la liste
// d'aujourd'hui (règle : fetchers_history.go), par l'ancre `historystate.Cursor.HeadID` ou, pour un
// marque-page rangé sans elle, par le delta du dessus et la page de recouvrement.
package v2

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// resumeAt est le point de reprise de la lecture : `start` est la position de la première page à
// lire ; `guard`, quand il n'est pas vide, est l'identifiant que l'entrée à `start` doit porter
// (contrôle de position : c'est le dernier match traité du marque-page, déjà rangé, et la lecture
// ordinaire commence à l'entrée suivante).
type resumeAt struct {
	start int
	guard string
}

// errAnchorMismatch : l'entrée lue à la position du marque-page n'est pas celle qu'il a rangée.
var errAnchorMismatch = errors.New("sync.v2: marque-page d'historique : identifiant absent à sa position")

// resumePoint situe le marque-page dans la liste d'aujourd'hui. proceed=false : la lecture du haut
// a épuisé le budget, ou vu la fin ou la borne de profondeur, avant d'avoir pu le situer.
func (sc *catchUpScan) resumePoint(ctx context.Context, cursor historystate.Cursor) (r resumeAt, proceed bool, err error) {
	if !cursor.HasAnchor() {
		if err := sc.scanNew(ctx); err != nil {
			return resumeAt{}, false, err
		}
		if !sc.reachedKnown {
			return resumeAt{}, false, nil
		}
		start, err := sc.resumeStart(ctx, cursor)
		return resumeAt{start: start}, err == nil, err
	}
	arrived, found, err := sc.scanTop(ctx, cursor)
	if err != nil {
		return resumeAt{}, false, err
	}
	if sc.budgetHit || sc.depthHit {
		return resumeAt{}, false, nil
	}
	if !found {
		sc.noteCursorFallback(ctx, cursor, "ancre introuvable dans les premières pages")
		return resumeAt{}, true, nil
	}
	resume := cursor.ResumeOffset(arrived)
	slog.DebugContext(ctx, "v2 ListUnknownMatches : marque-page situé par son ancre",
		"gamertag", sc.gamertag, "arrived", arrived, "offset", cursor.Offset, "resume_offset", resume)
	return resumeAt{start: resume - 1, guard: cursor.LastMatchID}, true, nil
}

// scanTop lit le haut de la liste jusqu'à l'ancre du marque-page, au plus
// historystate.AnchorMaxPages pages : les entrées qui la précèdent sont les matchs arrivés depuis la
// passe précédente, toutes entrées comptées. Les inconnus parmi elles sont listés (budget compris),
// les connus et les abandonnés sautés. found=false : ancre absente des pages lues (liste finie ou
// limite atteinte), ou lecture arrêtée sur le budget (sc.budgetHit).
func (sc *catchUpScan) scanTop(ctx context.Context, cursor historystate.Cursor) (arrived int, found bool, err error) {
	for pos := 0; pos < historystate.AnchorScanLimit(sc.p.pageSize); pos += sc.p.pageSize {
		entries, err := sc.page(ctx, pos)
		if err != nil {
			return 0, false, err
		}
		if len(entries) == 0 {
			sc.endSeen = true
			return 0, false, nil
		}
		for i, e := range entries {
			sc.noteRead(pos+i, e.MatchID)
			if e.MatchID == cursor.HeadID {
				return pos + i, true, nil
			}
			if sc.known[e.MatchID] || sc.skip[e.MatchID] {
				continue
			}
			if sc.addUnknown(pos+i, e.MatchID) {
				return 0, false, nil
			}
		}
		if len(entries) < sc.p.pageSize {
			sc.endSeen = true
			return 0, false, nil
		}
	}
	return 0, false, nil
}

// noteCursorFallback compte et journalise une reprise depuis le début de la liste, faute d'avoir pu
// situer le marque-page.
func (sc *catchUpScan) noteCursorFallback(ctx context.Context, cursor historystate.Cursor, reason string) {
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), counterHistoryCursorFallback)
	slog.WarnContext(ctx, "sync.v2: marque-page d'historique non situé — reprise depuis le début de la liste",
		"event", "sync.v2.history.cursor_fallback", "gamertag", sc.gamertag, "reason", reason,
		"offset", cursor.Offset, "new_matches", len(sc.unknown))
}

// resumeStart est l'étape (b) sans ancre : décalage de la page de recouvrement, vérifié par lecture.
// Rend 0 (reprise depuis le début) quand la page ne contient pas le dernier match du marque-page.
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
	sc.noteCursorFallback(ctx, cursor, "marque-page introuvable dans la page de recouvrement")
	return 0, nil
}
