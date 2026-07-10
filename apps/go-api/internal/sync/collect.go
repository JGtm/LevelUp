// Package sync — collect.go : Phase 2.1 du refactor Collect→Persist.
//
// buildBatchFromFetchedMatch transforme un *fetchedMatch (data déjà fetchée
// + parsée depuis l'API Halo, sans I/O DB) en *persist.MatchBatch prêt à
// être Submit() dans la BatchQueue.
//
// Chemin unique Collect→Persist : c'est la seule voie d'écriture per-match du
// live sync (le chemin legacy insertFetchedMatch a été supprimé au lot D1b).
// buildBatchFromFetchedMatch et fetchMatchData() forment l'entrée du pipeline.
//
// **Logique pure** : pas d'I/O DB, pas d'I/O API. Les seules erreurs
// possibles viennent du parsing du chunk highlight_events (analysis.
// ParseHighlightEvents). Une erreur de parse N'EST PAS fatale pour le batch :
// le batch est quand même retourné avec les autres données, l'erreur est
// retournée pour que le caller logge un warning.

package sync

import (
	"context"
	"fmt"
	"log/slog"
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
	return buildBatchFromFetchedMatchCtx(context.Background(), fm, titleSlug, gamertag, xuid)
}

// buildBatchFromFetchedMatchCtx — variante avec ctx pour propager l'event_id
// dans les logs DEBUG du collect. Appelée par submitMatchAsBatch qui a un ctx
// avec event_id. La signature sans ctx (buildBatchFromFetchedMatch) reste
// disponible pour les tests purs sans observabilité.
func buildBatchFromFetchedMatchCtx(
	ctx context.Context,
	fm *fetchedMatch,
	titleSlug, gamertag, xuid string,
) (*persist.MatchBatch, error) {
	if fm == nil || fm.Registry == nil {
		return nil, fmt.Errorf("buildBatchFromFetchedMatch: fm nil ou Registry absent")
	}

	builder := persist.NewBatchBuilder(titleSlug, gamertag, xuid, "sync_delta")
	builder.SetMatch(fm.Registry)

	// Phase 3 : poser les PBit skill sur les participants AVANT AddParticipants
	// (INSERT-only, parité legacy MarkSkillLoaded). skillOK = l'API skill a
	// renvoyé des données (pas d'erreur + ≥1 team_mmr). Les bits registry sont
	// agrégés en fin de fonction (cf. doc domain.MatchRegistryRow.BackfillCompleted
	// : la valeur doit être calculée AVANT le Submit, mode INSERT-only).
	skillOK := fm.SkillError == nil && hasAnyTeamMMR(fm.Participants)
	if skillOK {
		for i := range fm.Participants {
			if fm.Participants[i].TeamMMR != nil {
				bits := skillBitsCombined
				if fm.Participants[i].BackfillBits != nil {
					bits |= *fm.Participants[i].BackfillBits
				}
				fm.Participants[i].BackfillBits = &bits
			}
		}
	}

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
						MatchID:        fm.MatchID,
						KillerXUID:     p.KillerXUID,
						KillerGamertag: p.KillerGT,
						VictimXUID:     p.VictimXUID,
						VictimGamertag: p.VictimGT,
						Count:          1, // 1 row par kill event (agg via SUM kill_count)
						TimeMS:         p.TimeMS,
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

	// Shared CSRs (lobby) — CSR de tous les participants ranked.
	if len(fm.SharedCSRs) > 0 {
		mcInserts := make([]persist.MatchCSRInsert, 0, len(fm.SharedCSRs))
		for _, s := range fm.SharedCSRs {
			mcInserts = append(mcInserts, SharedCSRRowToMatchCSRInsert(s))
		}
		builder.AddMatchCSRs(mcInserts)
	}

	// PVE Firefight stats — toutes les rows participants si match firefight.
	if len(fm.PveStats) > 0 {
		pveInserts := make([]persist.PVEMatchStatsInsert, 0, len(fm.PveStats))
		for _, p := range fm.PveStats {
			pveInserts = append(pveInserts, pveMatchStatsRowToInsert(p))
		}
		builder.AddPVEStats(pveInserts)
	}

	// Enrichment placeholder — les enrichments post-sync compute
	// (performance_score, session_id, is_with_friends, dominance_flag,
	// engagement_*) sont UPDATEd post-Submit par les fonctions
	// engine_postsync.go. Le placeholder ici porte juste le PK.
	builder.SetEnrichment(&persist.EnrichmentRow{
		MatchID: fm.MatchID,
	})

	batch := builder.Build()
	slog.DebugContext(ctx, "collect: batch built",
		"match_id", fm.MatchID,
		"gamertag", gamertag,
		"participants", len(batch.Shared.Participants),
		"medals", len(batch.Shared.Medals),
		"highlight_events", len(batch.Shared.HighlightEvents),
		"killer_victim", len(batch.Shared.KillerVictim),
		"shared_csrs", len(batch.Shared.MatchCSRs),
		"xuid_aliases", len(batch.Shared.XUIDAliases),
		"psa", len(batch.PlayerData.PersonalScoreAwards),
		"player_csr", batch.PlayerData.SkillRank != nil,
		"pve_rows", func() int {
			if batch.PVE == nil {
				return 0
			}
			return len(batch.PVE.Stats)
		}(),
	)
	applyCompletionBitsToBatch(batch, skillOK)
	return batch, parseErr
}

// applyCompletionBitsToBatch agrège les bits de complétude sur la registry row du
// batch (Phase 3, INSERT-only — cf. doc domain.MatchRegistryRow.BackfillCompleted :
// valeur calculée AVANT le Submit). Lit l'état RÉEL du batch construit
// (highlight_events / killer_victim effectivement ajoutés) → backfill_completed
// fiable sur le chemin batch, sans UPDATE post-persist. Reproduit la parité de
// l'ancien insertFetchedMatch (supprimé D1b : MarkParticipantsDone /
// MarkSkillLoaded + events/kv).
// skillOK = l'API skill a renvoyé des données (cf. buildBatchFromFetchedMatch).
func applyCompletionBitsToBatch(batch *persist.MatchBatch, skillOK bool) {
	m := batch.Shared.Match
	if m == nil {
		return
	}
	var bits int64
	if m.BackfillCompleted != nil {
		bits = *m.BackfillCompleted
	}
	if len(batch.Shared.Participants) > 0 {
		bits |= backfillFlagParticipants
	}
	if skillOK {
		bits |= backfillFlagSkill
	}
	if len(batch.Shared.HighlightEvents) > 0 {
		bits |= MBitEvents
	}
	if len(batch.Shared.KillerVictim) > 0 {
		bits |= MBitKillerVictim
	}
	m.BackfillCompleted = &bits
}

// SharedCSRRowToMatchCSRInsert convertit SharedMatchCSRRow (sync) →
// persist.MatchCSRInsert (batch). Exporté : réutilisé par l'import OpenSpartan
// (service) qui construit son propre batch persist (E1, ADR 0019).
func SharedCSRRowToMatchCSRInsert(s SharedMatchCSRRow) persist.MatchCSRInsert {
	tier := s.Tier
	subTier := s.SubTier
	tierLabel := s.TierLabel
	out := persist.MatchCSRInsert{
		MatchID:     s.MatchID,
		XUID:        s.XUID,
		RatingType:  s.RatingType,
		RatingValue: s.RatingValue,
		Tier:        &tier,
		SubTier:     &subTier,
		TierLabel:   &tierLabel,
		RatingDelta: s.RatingDelta,
	}
	if s.MeasurementMatchesRemaining != 0 {
		mr := s.MeasurementMatchesRemaining
		out.MeasurementMatchesRemaining = &mr
	}
	if s.SeasonID != "" {
		sid := s.SeasonID
		out.SeasonID = &sid
	}
	return out
}

// pveMatchStatsRowToInsert convertit PveMatchStatsRow (sync) →
// persist.PVEMatchStatsInsert (batch). PveBits *int car colonne optionnelle.
func pveMatchStatsRowToInsert(r PveMatchStatsRow) persist.PVEMatchStatsInsert {
	out := persist.PVEMatchStatsInsert{
		MatchID:        r.MatchID,
		XUID:           r.XUID,
		WavesCompleted: r.WavesCompleted,
		BossKills:      r.BossKills,
		GruntKills:     r.GruntKills,
		EliteKills:     r.EliteKills,
		JackalKills:    r.JackalKills,
		BruteKills:     r.BruteKills,
		HunterKills:    r.HunterKills,
		SkimmerKills:   r.SkimmerKills,
		CrawlerKills:   r.CrawlerKills,
		SoldierKills:   r.SoldierKills,
		KnightKills:    r.KnightKills,
		WardenKills:    r.WardenKills,
		SentinelKills:  r.SentinelKills,
		MarineKills:    r.MarineKills,
		TotalKills:     r.TotalKills,
		Deaths:         r.Deaths,
		DamageDealt:    r.DamageDealt,
	}
	if r.PveBitsValue != 0 {
		v := int(r.PveBitsValue)
		out.PveBits = &v
	}
	return out
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
