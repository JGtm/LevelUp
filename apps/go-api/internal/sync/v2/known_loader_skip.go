// Package v2 — known_loader_skip.go : lecture, par joueur, de l'ensemble des matchs que la découverte
// ne demande plus (introuvables 404/410, abandonnés du rattrapage), voir historystate.SkipSetKey.
package v2

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/sync/historystate"
)

var _ SkipSetLoader = (*knownLoaderV2)(nil)

// LoadSkipSet lit l'ensemble des matchs à ne plus demander sur la base joueur (même ouverture
// lecture seule que LoadKnown). Une base qui ne s'ouvre pas ou une lecture en échec rendent nil
// (journalisé) : la découverte se passe alors de l'ensemble, au pire elle redemande un match
// introuvable ce cycle ; le cycle ne s'arrête pas pour cela.
func (l *knownLoaderV2) LoadSkipSet(ctx context.Context, p PlayerProfile) map[string]bool {
	db, release, err := l.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		slog.WarnContext(ctx, "sync.v2: matchs à ne plus demander illisibles — base joueur indisponible",
			"event", "sync.v2.skipset.read_failed", "player", p.PlayerSlug, "err", err)
		return nil
	}
	defer release()
	return historystate.LoadSkipSet(ctx, db)
}
