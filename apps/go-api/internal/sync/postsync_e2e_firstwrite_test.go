//go:build integration

package sync

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/domain"
)

// passePremierEcrit joue le PREMIER écrit d'une passe de sync par le chemin de production
// (fetchAndPersistMatches : fetch, résolution des noms, persistance INSERT-only), avec la
// résolution des amis une fois par passe comme run() (le branchement de run() lui-même est tenu
// par TestRunHistoryInputsCarryFriends). Rend le résultat de la passe et le nombre de matchs persistés.
func passePremierEcrit(ctx context.Context, e *SyncEngine, client HaloClient, shared, player *sql.DB, ids []string) (*domain.SyncResult, int) {
	in := historyPaginationInputs{
		client: client,
		opts: domain.SyncOptions{
			MatchType: "matchmaking", MaxMatches: 10,
			WithParticipants: true, WithMedals: true, WithHighlightEvents: true,
		},
		known:       map[string]bool{},
		sharedDB:    shared,
		playerDB:    player,
		friendXUIDs: ResolveFriendXUIDs(ctx, shared, e.friendsLoader, e.xuid),
	}
	result := &domain.SyncResult{}
	n := e.fetchAndPersistMatches(ctx, in, ids, result)
	return result, n
}
