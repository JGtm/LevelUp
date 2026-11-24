package livesync

// catchup_anchor.go — l'ancre du marque-page du rattrapage Halo 5 (historystate.Cursor.HeadID) : le
// nombre de matchs arrivés en tête de liste depuis la passe précédente se lit comme la position de
// l'ancienne tête dans la liste relue, toutes entrées comptées (Arena, mais aussi Warzone et Campagne,
// écartées et donc jamais connues). Sans ancre, N est le nombre d'entrées vues jusqu'à la dernière non connue (D-23).

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// ListIDsFunc : signature de halo5.ListMatchIDs côté livesync (injectable en test, comme CapturePageFunc).
type ListIDsFunc func(ctx context.Context, viewer canonical.PlayerIdentity, start, pageSize int) ([]string, error)

// listIDsFrom câble la lecture de liste sur la source live.
func listIDsFrom(src halo5.CaptureSource) ListIDsFunc {
	return func(ctx context.Context, viewer canonical.PlayerIdentity, start, pageSize int) ([]string, error) {
		if src == nil {
			return nil, errors.New("h5 list: source indisponible")
		}
		return halo5.ListMatchIDs(ctx, src, viewer.Gamertag, start, pageSize)
	}
}

// anchorKind est l'issue de la recherche de l'ancre d'un marque-page.
type anchorKind int

const (
	// anchorAbsent : pas d'ancre (marque-page rangé avant elle) ou liste illisible : règle sans ancre.
	anchorAbsent anchorKind = iota
	// anchorFound : l'ancienne tête est retrouvée, n entrées sont arrivées au-dessus d'elle.
	anchorFound
	// anchorLost : l'ancienne tête n'est pas dans les premières pages de la liste.
	anchorLost
)

// anchorResult est l'issue de locateAnchor ; n n'a de sens que si kind vaut anchorFound.
type anchorResult struct {
	kind anchorKind
	n    int
}

// locateAnchor cherche l'ancre du marque-page dans le début de la liste lu par le delta du dessus
// (stats.TopIDs). Retrouvée : N exact. Non vue par le delta (il s'est arrêté sur un match connu plus
// haut, ou au budget) : la liste est relue page à page jusqu'à historystate.AnchorScanLimit entrées.
// Introuvable dans cette fenêtre, ou liste épuisée : ancre perdue.
func (p *catchUpPass) locateAnchor(ctx context.Context, cur historystate.Cursor, stats halo5.CaptureStats) anchorResult {
	if !cur.HasAnchor() {
		return anchorResult{kind: anchorAbsent}
	}
	if n, ok := cur.ArrivedSince(stats.TopIDs); ok {
		return anchorResult{kind: anchorFound, n: n}
	}
	limit := historystate.AnchorScanLimit(backfillDefaultPageSize)
	if stats.HistoryEnd || len(stats.TopIDs) >= limit {
		return anchorResult{kind: anchorLost}
	}
	n, found, err := p.searchAnchor(ctx, cur, stats.TopIDs, limit)
	switch {
	case err != nil:
		p.r.logger.WarnContext(ctx, "h5 sync: lecture de la tete de liste impossible, marque-page decale sans ancre",
			"gamertag", p.r.deps.Viewer.Gamertag, "err", err)
		return anchorResult{kind: anchorAbsent}
	case found:
		return anchorResult{kind: anchorFound, n: n}
	}
	return anchorResult{kind: anchorLost}
}

// searchAnchor complète le début de liste déjà lu (top) page à page, jusqu'à retrouver l'ancre, limit
// entrées ou la fin de la liste : un appel par page, en reprenant à la page qui contient la première
// entrée non lue.
func (p *catchUpPass) searchAnchor(ctx context.Context, cur historystate.Cursor, top []string, limit int) (n int, found bool, err error) {
	list := p.r.deps.ListIDs
	if list == nil {
		list = listIDsFrom(p.src)
	}
	ids := append([]string(nil), top...)
	const pageSize = backfillDefaultPageSize
	for start := len(ids) / pageSize * pageSize; len(ids) < limit; start += pageSize {
		page, err := list(ctx, p.r.deps.Viewer, start, pageSize)
		if err != nil {
			return 0, false, err
		}
		if skip := len(ids) - start; skip < len(page) {
			ids = append(ids, page[skip:]...)
		}
		if n, ok := cur.ArrivedSince(ids); ok {
			return n, true, nil
		}
		if len(page) < pageSize {
			break
		}
	}
	return 0, false, nil
}

// noteAnchorLost journalise et compte une ancre perdue : plus de historystate.AnchorMaxPages pages de
// matchs sont arrivées en tête depuis le marque-page (ou la liste a perdu l'ancienne tête). Le
// compteur est celui des marque-pages perdus (metricCursorMissed).
func (p *catchUpPass) noteAnchorLost(ctx context.Context, cur historystate.Cursor) {
	slog.WarnContext(ctx, "h5 sync: ancre du marque-page introuvable en tête de liste — reprise depuis le début",
		"gamertag", p.r.deps.Viewer.Gamertag, "head_match_id", cur.HeadID, "offset", cur.Offset)
	observability.IncCounter(metricCursorMissed)
}
