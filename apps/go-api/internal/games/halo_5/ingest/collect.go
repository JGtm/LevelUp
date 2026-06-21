package ingest

import (
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// CollectMatchBatch assemble le MatchBatch (PUR) d'un match Halo 5 à partir de son
// résumé + sa timeline canonique. UN SEUL batch par match (médailles + kills +
// arme), car match_registry est l'ancre d'idempotence : un 2e batch sur le même
// match serait skippé par SharedPersister → ses lignes ne seraient jamais écrites.
//
//   - match_registry (ancre) depuis le résumé,
//   - match_participants (roster) depuis le carnage (mappé en amont, package halo_5),
//   - medals_earned (agrégat) + highlight_events (médailles horodatées),
//   - killer_victim_pairs (kill-feed par-kill) + weapon_kills (arme par kill).
//
// resolveXUID(gamertag) est injecté (le câblage live fournit le CachingResolver).
// viewer = joueur dont la consultation déclenche la capture (owner du batch +
// first_sync_by). participants = roster du carnage (nil si carnage indisponible →
// match collecté sans participants). Le batch est persisté par SharedPersister.
func CollectMatchBatch(
	titleSlug, source string,
	viewer canonical.PlayerIdentity,
	summary canonical.MatchSummary,
	timeline []canonical.MatchEvent,
	participants []domain.MatchParticipantRow,
	resolveXUID func(gamertag string) string,
) *persist.MatchBatch {
	registry := MatchRegistryRowFromSummary(summary, viewer.Gamertag)
	medals, medalEvents := MapMedalEvents(summary.MatchID, timeline, resolveXUID)
	pairs, weapons := MapKillEvents(summary.MatchID, timeline, resolveXUID)
	positions := MapKillPositions(summary.MatchID, timeline, resolveXUID)

	return persist.NewBatchBuilder(titleSlug, viewer.Gamertag, viewer.XUID, source).
		SetMatch(&registry).
		AddParticipants(participants).
		// xuid↔gamertag du roster → shared.xuid_aliases : source canonique UNIQUE du
		// mapping (xuid ET gamertag au même endroit) ET graine du resolver PeopleHub
		// (évite de re-résoudre un joueur déjà vu aux runs suivants → anti rate-limit).
		AddXUIDAliases(xuidAliasesFromParticipants(participants, summary.StartedAtUTC)).
		AddMedals(medals).
		AddHighlightEvents(medalEvents).
		AddKillerVictim(pairs).
		AddWeaponKills(weapons).
		AddKillPositions(positions).
		Build()
}

// xuidAliasesFromParticipants extrait le mapping xuid↔gamertag du roster résolu
// pour le persister dans shared.xuid_aliases (INSERT OR IGNORE côté persister).
func xuidAliasesFromParticipants(participants []domain.MatchParticipantRow, seen time.Time) []persist.XUIDAliasInsert {
	out := make([]persist.XUIDAliasInsert, 0, len(participants))
	for i := range participants {
		p := participants[i]
		if p.XUID == "" || p.Gamertag == nil || *p.Gamertag == "" {
			continue
		}
		out = append(out, persist.XUIDAliasInsert{XUID: p.XUID, Gamertag: *p.Gamertag, LastSeen: seen})
	}
	return out
}
