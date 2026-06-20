package ingest

import (
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// CollectMatchBatch assemble le MatchBatch (PUR) d'un match Halo 5 à partir de son
// résumé + sa timeline canonique. UN SEUL batch par match (médailles + kills +
// arme), car match_registry est l'ancre d'idempotence : un 2e batch sur le même
// match serait skippé par SharedPersister → ses lignes ne seraient jamais écrites.
//
//   - match_registry (ancre) depuis le résumé,
//   - medals_earned (agrégat) + highlight_events (médailles horodatées),
//   - killer_victim_pairs (kill-feed par-kill) + weapon_kills (arme par kill).
//
// resolveXUID(gamertag) est injecté (le câblage live fournit le CachingResolver).
// viewer = joueur dont la consultation déclenche la capture (owner du batch +
// first_sync_by). Le batch est persisté par SharedPersister sur le shared du titre.
func CollectMatchBatch(
	titleSlug, source string,
	viewer canonical.PlayerIdentity,
	summary canonical.MatchSummary,
	timeline []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) *persist.MatchBatch {
	registry := MatchRegistryRowFromSummary(summary, viewer.Gamertag)
	medals, medalEvents := MapMedalEvents(summary.MatchID, timeline, resolveXUID)
	pairs, weapons := MapKillEvents(summary.MatchID, timeline, resolveXUID)
	positions := MapKillPositions(summary.MatchID, timeline, resolveXUID)

	return persist.NewBatchBuilder(titleSlug, viewer.Gamertag, viewer.XUID, source).
		SetMatch(&registry).
		AddMedals(medals).
		AddHighlightEvents(medalEvents).
		AddKillerVictim(pairs).
		AddWeaponKills(weapons).
		AddKillPositions(positions).
		Build()
}
