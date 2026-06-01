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

// MatchCallback est appelé quand de nouveaux matchs sont détectés. Retourne true
// si les matchs ont été ACCEPTÉS (état Watching → sync lancé) ; false s'ils ont
// été ignorés (état busy Syncing/Cooling). Le poller ne marque connus QUE les
// matchs acceptés → les matchs détectés pendant un sync sont re-signalés au
// prochain poll au lieu d'être perdus (W4, revue 2026-06-01).
type MatchCallback func(matchIDs []string) bool

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

	// Poll immédiat = baseline seul (AS-4) : marque l'historique existant comme
	// connu SANS déclencher de sync. Sinon, chaque entrée en Watching re-syncait
	// les 25 derniers matchs déjà en DB (faux "nouveaux matchs").
	p.poll(ctx, true)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "match_poller: arrêté", "gamertag", p.gamertag)
			return
		case <-ticker.C:
			p.poll(ctx, false)
		}
	}
}

// poll effectue un appel à l'API et détecte les nouveaux matchs.
//
// seedOnly=true : 1er poll — marque les matchs détectés comme connus SANS les
// signaler (établit le baseline pré-Watching, AS-4).
func (p *MatchPoller) poll(ctx context.Context, seedOnly bool) {
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
		}
	}
	if len(newIDs) == 0 {
		return
	}

	if seedOnly {
		for _, id := range newIDs {
			p.knownIDs[id] = true
		}
		slog.InfoContext(ctx, "match_poller: baseline seedé",
			"gamertag", p.gamertag, "count", len(newIDs))
		return
	}

	slog.InfoContext(ctx, "match_poller: nouveaux matchs détectés",
		"gamertag", p.gamertag,
		"count", len(newIDs),
		"ids", fmt.Sprintf("%v", newIDs),
	)

	// W4 : ne marquer connus QUE si acceptés. Si onNewMatches retourne false
	// (état busy), les IDs restent inconnus → re-signalés au prochain poll une
	// fois revenu en Watching → aucun match détecté pendant un sync n'est perdu.
	accepted := true
	if p.onNewMatches != nil {
		accepted = p.onNewMatches(newIDs)
	}
	if accepted {
		for _, id := range newIDs {
			p.knownIDs[id] = true
		}
	} else {
		slog.DebugContext(ctx, "match_poller: matchs non acceptés (busy), re-signal au prochain poll",
			"gamertag", p.gamertag, "count", len(newIDs))
	}
}
