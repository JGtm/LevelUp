package service

// demo_replay_allowlist.go — QUELS rejeux la démo sert (décision D-2 du plan des
// recommandations du 2026-10-09) : ceux de l'index des rejeux figés, et eux seuls.

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// DemoReplayAllowlist répond, en démo, « ce match a-t-il un rejeu figé servi ? ».
type DemoReplayAllowlist struct {
	layout title.DemoLayout
}

// NewDemoReplayAllowlist construit la liste blanche sur la disposition de la démo.
func NewDemoReplayAllowlist(layout title.DemoLayout) *DemoReplayAllowlist {
	return &DemoReplayAllowlist{layout: layout}
}

// Allows lit l'index du titre (quelques Ko, relu à chaque appel : le seed le réécrit à chaque
// régénération de la démo) et compare les formes COURTES des matchs. Un index absent vaut
// « aucun rejeu » ; un index illisible aussi, journalisé.
func (a *DemoReplayAllowlist) Allows(ctx context.Context, titleSlug, matchID string) bool {
	raw, err := os.ReadFile(a.layout.ReplayIndexPath(titleSlug))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		slog.ErrorContext(ctx, "rejeu démo : index illisible", "err", err, "titleSlug", titleSlug)
		return false
	}
	var index domain.DemoReplayIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		slog.ErrorContext(ctx, "rejeu démo : index invalide", "err", err, "titleSlug", titleSlug)
		return false
	}
	want := title.FilmShortMatchID(matchID)
	for _, m := range index.Matches {
		if title.FilmShortMatchID(m.MatchID) == want {
			return true
		}
	}
	return false
}
