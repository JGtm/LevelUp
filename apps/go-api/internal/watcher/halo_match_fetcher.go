// Package watcher — halo_match_fetcher.go : adaptateur HaloAPI → MatchFetcher.
//
// Le MatchPoller consomme l'interface MatchFetcher (FetchRecentMatchIDs).
// Ce fichier la satisfait en s'appuyant sur PooledHaloClient (sync/), qui
// gère le pool de tokens partagé avec l'auto-sync (round-robin PolicyAnyPublic).
//
// Format xuid : GetMatchHistory exige fmt.Sprintf("xuid(%s)", xuid). Passer
// un gamertag textuel renvoie une réponse stale figée silencieusement
// (cf. mémoire reference_halo_api_xuid_format.md, incident mai 2026).
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	syncpkg "levelup/go-api/internal/sync"
)

// matchHistoryClient est l'interface narrow attendue par HaloMatchFetcher.
// *syncpkg.PooledHaloClient la satisfait structurellement. Une interface
// dédiée (1 méthode) plutôt que le type concret pour : (1) testabilité —
// mocker 1 méthode au lieu de 8 ; (2) ISP — l'adaptateur n'a besoin que
// de GetMatchHistory.
type matchHistoryClient interface {
	GetMatchHistory(
		ctx context.Context,
		gamertag, matchType string,
		start, count int,
	) ([]syncpkg.MatchHistoryEntry, error)
}

// HaloMatchFetcher implémente MatchFetcher en interrogeant l'API Halo
// (/hi/players/xuid(N)/matches) via un client HTTP poolé.
type HaloMatchFetcher struct {
	client matchHistoryClient
}

// NewHaloMatchFetcher crée un fetcher branché sur le client poolé fourni.
func NewHaloMatchFetcher(client matchHistoryClient) *HaloMatchFetcher {
	return &HaloMatchFetcher{client: client}
}

// FetchRecentMatchIDs retourne les `count` derniers match_ids d'un joueur
// identifié par son xuid (numérique, sans préfixe).
func (f *HaloMatchFetcher) FetchRecentMatchIDs(
	ctx context.Context,
	xuid string,
	count int,
) ([]string, error) {
	xuid = strings.TrimSpace(xuid)
	if xuid == "" {
		return nil, fmt.Errorf("halo_match_fetcher: xuid vide")
	}
	if count < 1 {
		count = 1
	}
	gamertag := fmt.Sprintf("xuid(%s)", xuid)
	entries, err := f.client.GetMatchHistory(ctx, gamertag, "all", 0, count)
	if err != nil {
		return nil, fmt.Errorf("halo_match_fetcher: fetch xuid=%s: %w", xuid, err)
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.MatchID)
	}
	slog.DebugContext(ctx, "halo_match_fetcher: matchs récupérés",
		"xuid", xuid,
		"count", len(ids),
	)
	return ids, nil
}

// Vérification statique que HaloMatchFetcher satisfait MatchFetcher.
var _ MatchFetcher = (*HaloMatchFetcher)(nil)
