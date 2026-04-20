// Package watcher — match_poller.go : polling de nouveaux matchs via l'API Halo.
//
// Quand la FSM est en état Watching, le match_poller vérifie périodiquement
// l'API Halo pour détecter de nouveaux match_ids.
// Quand un nouveau match est trouvé, il le signale au watcher pour déclencher un sync.
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	// defaultPollInterval est l'intervalle par défaut entre deux polls.
	defaultPollInterval = 30 * time.Second
)

// MatchFetcher est l'interface pour récupérer les derniers match_ids d'un joueur.
type MatchFetcher interface {
	// FetchRecentMatchIDs retourne les N derniers match_ids d'un joueur.
	FetchRecentMatchIDs(ctx context.Context, xuid string, count int) ([]string, error)
}

// MatchCallback est appelé quand de nouveaux matchs sont détectés.
type MatchCallback func(matchIDs []string)

// MatchPoller poll l'API pour détecter les nouveaux matchs d'un joueur.
type MatchPoller struct {
	xuid         string
	gamertag     string
	fetcher      MatchFetcher
	interval     time.Duration
	onNewMatches MatchCallback
	knownIDs     map[string]bool
}

// NewMatchPoller crée un poller de matchs.
func NewMatchPoller(xuid, gamertag string, fetcher MatchFetcher, onNewMatches MatchCallback) *MatchPoller {
	return &MatchPoller{
		xuid:         xuid,
		gamertag:     gamertag,
		fetcher:      fetcher,
		interval:     defaultPollInterval,
		onNewMatches: onNewMatches,
		knownIDs:     make(map[string]bool),
	}
}

// SetInterval change l'intervalle de polling.
func (p *MatchPoller) SetInterval(d time.Duration) {
	p.interval = d
}

// SeedKnownIDs initialise les IDs déjà connus (matchs récents en DB).
func (p *MatchPoller) SeedKnownIDs(ids []string) {
	for _, id := range ids {
		p.knownIDs[id] = true
	}
	slog.Info("match_poller: seed IDs connus",
		"gamertag", p.gamertag,
		"count", len(ids),
	)
}

// Run démarre le polling. Bloquant — à lancer dans une goroutine.
func (p *MatchPoller) Run(ctx context.Context) {
	slog.InfoContext(ctx, "match_poller: démarré",
		"gamertag", p.gamertag,
		"interval", p.interval,
	)

	// Poll immédiat pour récupérer l'état initial
	p.poll(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "match_poller: arrêté", "gamertag", p.gamertag)
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll effectue un appel à l'API et détecte les nouveaux matchs.
func (p *MatchPoller) poll(ctx context.Context) {
	recentIDs, err := p.fetcher.FetchRecentMatchIDs(ctx, p.xuid, 25)
	if err != nil {
		slog.WarnContext(ctx, "match_poller: erreur fetch",
			"gamertag", p.gamertag,
			"err", err,
		)
		return
	}

	var newIDs []string
	for _, id := range recentIDs {
		if !p.knownIDs[id] {
			newIDs = append(newIDs, id)
			p.knownIDs[id] = true
		}
	}

	if len(newIDs) > 0 {
		slog.InfoContext(ctx, "match_poller: nouveaux matchs détectés",
			"gamertag", p.gamertag,
			"count", len(newIDs),
			"ids", fmt.Sprintf("%v", newIDs),
		)
		if p.onNewMatches != nil {
			p.onNewMatches(newIDs)
		}
	}
}
