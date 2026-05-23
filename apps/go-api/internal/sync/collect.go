// Package sync — collect.go : Phase 2.1 du refactor Collect→Persist.
//
// buildBatchFromFetchedMatch transforme un *fetchedMatch (data déjà fetchée
// + parsée depuis l'API Halo, sans I/O DB) en *persist.MatchBatch prêt à
// être Submit() dans la BatchQueue.
//
// **Coexistence avec insertFetchedMatch** : cette fonction est utilisée par
// le chemin Collect→Persist (Phase 2+) — le chemin direct insertFetchedMatch
// reste utilisé tant que le feature flag LEVELUP_PERSIST_BATCH n'est pas
// activé. Les deux chemins partagent fetchMatchData() en entrée.
//
// **Logique pure** : pas d'I/O DB, pas d'I/O API. Les seules erreurs
// possibles viennent du parsing du chunk highlight_events (analysis.
// ParseHighlightEvents). Une erreur de parse N'EST PAS fatale pour le batch :
// le batch est quand même retourné avec les autres données, l'erreur est
// retournée pour que le caller logge un warning.

package sync

import (
	"fmt"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/persist"
)

// buildBatchFromFetchedMatch convertit un fetchedMatch en MatchBatch.
//
// Couvre Phase 2.1 :
//   - SharedBatch.Match, .Participants, .Medals, .XUIDAliases
//   - SharedBatch.HighlightEvents, .KillerVictim (depuis le chunk highlight)
//   - PlayerData.PersonalScoreAwards
//   - PlayerData.SkillRank (si match ranked → CSR)
//   - PlayerData.Enrichment (placeholder MatchID seulement — Phase 2.2 le
//     remplira avec performance_score/sessions/friends/etc.)
//
// Hors scope Phase 2.1 (Phase 2.2+) :
//   - shared.match_csrs (CSR lobby de tous les participants)
//   - weapon_kills (backfill film séparé, pas dans fetchedMatch)
//   - pve_match_stats (firefight, fetchedMatch ne l'extrait pas)
//   - Tous les enrichments post-sync compute (perf, session, friends, etc.)
//   - Bitmasks backfill_completed / backfill_bits (à calculer côté caller
//     en fonction des données effectivement fetched + skill réussi)
func buildBatchFromFetchedMatch(
	fm *fetchedMatch,
	titleSlug, gamertag, xuid string,
) (*persist.MatchBatch, error) {
	if fm == nil || fm.Registry == nil {
		return nil, fmt.Errorf("buildBatchFromFetchedMatch: fm nil ou Registry absent")
	}

	builder := persist.NewBatchBuilder(titleSlug, gamertag, xuid, "sync_delta")
	builder.SetMatch(fm.Registry)
	if len(fm.Participants) > 0 {
		builder.AddParticipants(fm.Participants)
	}
	if len(fm.Medals) > 0 {
		builder.AddMedals(fm.Medals)
	}

	// XUID aliases — extraits des participants ayant un gamertag non-vide.
	now := time.Now().UTC()
	var aliases []persist.XUIDAliasInsert
	for _, p := range fm.Participants {
		if p.Gamertag != nil && *p.Gamertag != "" {
			aliases = append(aliases, persist.XUIDAliasInsert{
				XUID: p.XUID, Gamertag: *p.Gamertag, LastSeen: now,
			})
		}
	}

	// Highlight events + killer/victim pairs (parsing pur du chunk binaire).
	// L'erreur de parse NE rend PAS le batch invalide : on continue sans
	// events. Le caller logge un warning via l'erreur retournée.
	var parseErr error
	if fm.HasHighlights && len(fm.HighlightData) > 0 {
		events, err := analysis.ParseHighlightEvents(fm.HighlightData, fm.FilmMajorVer)
		if err != nil {
			parseErr = fmt.Errorf("ParseHighlightEvents: %w", err)
		} else if len(events) > 0 {
			heInserts := make([]persist.HighlightEventInsert, 0, len(events))
			for _, ev := range events {
				xuidStr := strconv.FormatUint(ev.XUID, 10)
				heInserts = append(heInserts, persist.HighlightEventInsert{
					MatchID:   fm.MatchID,
					XUID:      &xuidStr,
					EventType: ev.EventType,
					TimeMS:    ev.TimeMS,
				})
				// Alias additionnels depuis events (joueurs hors participants).
				if ev.Gamertag != "" && ev.XUID != 0 {
					aliases = append(aliases, persist.XUIDAliasInsert{
						XUID: xuidStr, Gamertag: ev.Gamertag, LastSeen: now,
					})
				}
			}
			builder.AddHighlightEvents(heInserts)

			// killer_victim_pairs : conversion HighlightEvent → RawEvent puis
			// délégation à analysis.ComputeKillerVictimPairs (mirror du chemin
			// legacy InsertKillerVictimPairsFromEvents pour comportement identique).
			raw := make([]analysis.RawEvent, 0, len(events))
			for _, ev := range events {
				if ev.EventType != analysis.EventTypeKill && ev.EventType != analysis.EventTypeDeath {
					continue
				}
				raw = append(raw, analysis.RawEvent{
					EventType: ev.EventType,
					XUID:      strconv.FormatUint(ev.XUID, 10),
					Gamertag:  ev.Gamertag,
					TimeMS:    int64(ev.TimeMS),
				})
			}
			const toleranceMS = int64(5)
			pairs := analysis.ComputeKillerVictimPairs(raw, toleranceMS)
			if len(pairs) > 0 {
				kvInserts := make([]persist.KillerVictimInsert, 0, len(pairs))
				for _, p := range pairs {
					kvInserts = append(kvInserts, persist.KillerVictimInsert{
						MatchID:    fm.MatchID,
						KillerXUID: p.KillerXUID,
						VictimXUID: p.VictimXUID,
						Count:      1, // 1 row par kill event (agg via SUM kill_count)
					})
				}
				builder.AddKillerVictim(kvInserts)
			}
		}
	}

	if len(aliases) > 0 {
		builder.AddXUIDAliases(aliases)
	}

	// PSA (PersonalScoreAwards) → player batch.
	if len(fm.PSA) > 0 {
		psaInserts := make([]persist.PersonalScoreAwardInsert, 0, len(fm.PSA))
		for _, a := range fm.PSA {
			psaInserts = append(psaInserts, persist.PersonalScoreAwardInsert{
				MatchID:       a.MatchID,
				XUID:          a.XUID,
				AwardName:     a.AwardName,
				AwardCategory: a.AwardCategory,
				AwardCount:    a.AwardCount,
				AwardScore:    a.AwardScore,
			})
		}
		builder.AddPersonalScoreAwards(psaInserts)
	}

	// CSR du joueur sync (match_skill_rank) si match ranked.
	if fm.CSRRow != nil {
		builder.SetSkillRank(matchCSRRowToSkillRankInsert(fm.CSRRow))
	}

	// Enrichment placeholder — Phase 2.2 enrichira (performance_score,
	// session_id, is_with_friends, dominance_flag, engagement_*, etc.).
	builder.SetEnrichment(&persist.EnrichmentRow{
		MatchID: fm.MatchID,
	})

	return builder.Build(), parseErr
}

// matchCSRRowToSkillRankInsert convertit le MatchCSRRow (struct interne sync)
// en persist.SkillRankInsert. RatingType forcé à "CSR" car fetchedMatch ne
// produit que des rows de matchs ranked.
func matchCSRRowToSkillRankInsert(r *MatchCSRRow) *persist.SkillRankInsert {
	tier := r.Tier
	tierFR := r.TierFR
	subTier := r.SubTier
	tierLabel := r.TierLabel
	playlistGroup := r.PlaylistGroup
	return &persist.SkillRankInsert{
		MatchID:       r.MatchID,
		RatingType:    "CSR",
		RatingValue:   r.RatingValue,
		Tier:          &tier,
		TierFR:        &tierFR,
		SubTier:       &subTier,
		TierLabel:     &tierLabel,
		RatingDelta:   r.RatingDelta,
		PlaylistGroup: &playlistGroup,
	}
}
