package ingest

import (
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// CollectMedalsBatch assemble le MatchBatch (PUR) de la tranche médailles à partir
// d'un résumé de match + sa timeline canonique :
//   - match_registry (ancre) depuis le résumé,
//   - medals_earned (agrégat) + highlight_events (horodaté) depuis les events medal.
//
// resolveXUID(gamertag) est injecté (le câblage live fournit le CachingResolver).
// viewer = joueur dont la consultation déclenche la capture (owner du batch +
// first_sync_by). Le batch est ensuite persisté par SharedPersister sur le shared
// du titre (réutilisation à 100% : SetMatch rend Persist non-no-op).
func CollectMedalsBatch(
	titleSlug, source string,
	viewer canonical.PlayerIdentity,
	summary canonical.MatchSummary,
	timeline []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) *persist.MatchBatch {
	registry := MatchRegistryRowFromSummary(summary, viewer.Gamertag)
	medals, events := MapMedalEvents(summary.MatchID, timeline, resolveXUID)

	return persist.NewBatchBuilder(titleSlug, viewer.Gamertag, viewer.XUID, source).
		SetMatch(&registry).
		AddMedals(medals).
		AddHighlightEvents(events).
		Build()
}
