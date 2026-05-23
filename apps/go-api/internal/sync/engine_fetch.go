// Package sync — engine_fetch.go : pipeline parallèle fetch + insert séquentiel.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - fetchedMatch : container des données extraites d'un GetMatchStats prêtes
//     pour insertion (registry, participants+skill, medals, PSA, highlight chunk).
//   - fetchMatchData : exécute fetch + extraction d'un match (pur, sans DB).
//     Appelé en parallèle par run() via errgroup (Phase 2 — RPS limité par
//     HaloAPIClient).
//   - insertFetchedMatch : insère les données fetchées d'un match (séquentiel,
//     order-preserving). Appelé après le wait du errgroup (Phase 3).
//   - hasAnyTeamMMR : helper pour décider si MarkSkillLoaded doit être appelé.
//
// Le découpage fetch/insert permet de paralléliser les fetches tout en gardant
// les inserts séquentiels (order-preserving, évite races sur les UPSERT shared).
// Comportement INCHANGÉ — pur déplacement.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// fetchedMatch contient les données extraites d'un GetMatchStats, prêtes pour
// insertion (chemin legacy `insertFetchedMatch`) ou conversion en MatchBatch
// (chemin Collect→Persist `buildBatchFromFetchedMatch`).
//
// Les deux chemins consomment ce type : insertFetchedMatch écrit directement
// dans les DBs (legacy), buildBatchFromFetchedMatch produit un *persist.MatchBatch
// à Submit dans la BatchQueue. Coexistence pendant la transition Phase 2.
type fetchedMatch struct {
	MatchID        string
	Registry       *MatchRegistryRow
	Participants   []ParticipantRow
	Medals         []MedalRow
	PSA            []PersonalScoreAwardRow // PersonalScores du joueur courant (player DB)
	HighlightData  []byte                  // Raw highlight events chunk (ou nil si absent)
	FilmMajorVer   int
	HasHighlights  bool
	HighlightError error // Non-bloquant si présent
	SkillError     error // Non-bloquant si présent
	// CSRRow : ligne CSR à insérer côté player DB. Renseignée uniquement
	// pour les matchs classés dont le payload skill contient RankRecap.
	// Inséré dans insertFetchedMatch / batch.PlayerData.SkillRank.
	CSRRow *MatchCSRRow

	// SharedCSRs : CSR de TOUS les participants ranked du match (lobby
	// context). Produit par ExtractAllSharedCSRRows à partir de skillData,
	// utilisé par batch.Shared.MatchCSRs. Vide si match non-ranked ou
	// skillData absent. Cf. ADR/csr_shared_writes.go.
	SharedCSRs []SharedMatchCSRRow

	// PveStats : stats Firefight pour TOUS les participants du match
	// (1 row par joueur). Produit par ExtractPveStats si le match est
	// firefight (GameVariantCategory 41/42). Vide sinon.
	// Utilisé par batch.PVE.Stats (slice).
	PveStats []PveMatchStatsRow
}

// fetchMatchData exécute le fetch et l'extraction pour un match (pur, sans DB).
// Retourne les données extraites prêtes pour insertion séquentielle.
func (e *SyncEngine) fetchMatchData(
	ctx context.Context,
	client HaloClient,
	matchID string,
	opts domain.SyncOptions,
) (*fetchedMatch, error) {
	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "sync: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("GetMatchStats: %w", err)
	}

	fm := &fetchedMatch{
		MatchID: matchID,
	}

	// Extract registry (obligatoire).
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "sync: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("ExtractRegistry: %w", err)
	}
	fm.Registry = reg

	// Extract optionnels.
	if opts.WithParticipants {
		fm.Participants = ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé : l'API renvoie
		// parfois Gamertag/PlayerName vide pour le joueur appelant.
		ensureGamertagForSelf(fm.Participants, e.xuid, e.gamertag)

		// Skill API : team_mmr, enemy_mmr, kills/deaths_expected.
		// Endpoint séparé du stats — non-bloquant : un échec produit un warning.
		if xuids := ParticipantXUIDs(fm.Participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				fm.SkillError = fmt.Errorf("GetMatchSkill: %w", skillErr)
			} else if len(skillData) > 0 {
				fm.Participants = MergeSkillIntoParticipants(fm.Participants, skillData)
				// CSR par-match (player DB) : extraction depuis RankRecap si
				// match classé. L'écriture en player DB est différée à
				// insertFetchedMatch (legacy) ou batch.PlayerData.SkillRank.
				fm.CSRRow = ExtractCSRRowIfRanked(fm.Registry, skillData[e.xuid])
				// CSR de tous les participants ranked (shared.match_csrs)
				// — lobby context, utilisé par batch.Shared.MatchCSRs.
				fm.SharedCSRs = ExtractAllSharedCSRRows(fm.Registry, skillData)
			}
		}
	}
	if opts.WithMedals {
		fm.Medals = ExtractMedals(matchJSON)
	}
	// PVE Firefight stats — extraits si le match est firefight (déterminé
	// dans la registry). Tous les participants ; le batch écrit dans
	// shared_pve.pve_match_stats PK (match_id, xuid). Cf. batch.PVE.Stats.
	if fm.Registry != nil && fm.Registry.IsFirefight {
		fm.PveStats = ExtractPveStats(matchID, matchJSON)
	}
	// PersonalScores du joueur courant — toujours extraits (pas de flag dédié,
	// même cycle de vie que les participants). La table n'est pas dans shared :
	// l'insertion se fera côté playerDB dans insertFetchedMatch.
	fm.PSA = ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if opts.WithHighlightEvents {
		data, filmMajorVer, found, err := client.GetHighlightEventsChunk(ctx, matchID)
		fm.HasHighlights = found
		fm.FilmMajorVer = filmMajorVer
		if err != nil {
			fm.HighlightError = fmt.Errorf("GetHighlightEventsChunk: %w", err)
		} else if found {
			fm.HighlightData = data
		}
	}

	return fm, nil
}

// insertFetchedMatch insère les données fetchées d'un match (séquentiel, order-preserving).
func (e *SyncEngine) insertFetchedMatch(
	ctx context.Context,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	// Registry (obligatoire).
	if err := InsertRegistryIfNotExists(ctx, sharedDB, *fm.Registry); err != nil {
		slog.ErrorContext(ctx, "sync: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// Participants.
	if len(fm.Participants) > 0 {
		if fm.SkillError != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skill %s: %v", fm.MatchID, fm.SkillError))
		}
		if err := InsertParticipants(ctx, sharedDB, fm.Participants); err != nil {
			slog.ErrorContext(ctx, "sync: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(fm.Participants)

		// Phase 2 du plan PLAN_BITMASKS_AUDIT_FIX : marquer le bit
		// participants pour que `levelup backfill --participants` ne re-traite
		// pas indéfiniment ce match.
		if markErr := MarkParticipantsDone(ctx, sharedDB, fm.MatchID); markErr != nil {
			slog.WarnContext(ctx, "sync: MarkParticipantsDone échoué",
				"match_id", fm.MatchID, "err", markErr)
		}

		// Phase 2 — skill bits : on ne marque que si l'API skill a renvoyé des
		// données (fm.SkillError nil ET team_mmr présent sur ≥1 participant).
		// MarkSkillLoaded filtre lui-même sur team_mmr IS NOT NULL côté SQL.
		if fm.SkillError == nil && hasAnyTeamMMR(fm.Participants) {
			if markErr := MarkSkillLoaded(ctx, sharedDB, fm.MatchID); markErr != nil {
				slog.WarnContext(ctx, "sync: MarkSkillLoaded échoué",
					"match_id", fm.MatchID, "err", markErr)
			}
		}

		// Upsert XUID aliases.
		aliased := 0
		for _, p := range fm.Participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				if globalDB != nil {
					_ = UpsertXUIDAlias(ctx, globalDB, p.XUID, *p.Gamertag)
				}
				aliased++
			}
		}
		slog.DebugContext(ctx, "sync: participants insérés",
			"match_id", fm.MatchID, "participants", len(fm.Participants), "aliases_upserted", aliased,
		)
	}

	// Medals.
	if len(fm.Medals) > 0 {
		if err := InsertMedals(ctx, sharedDB, fm.Medals); err != nil {
			slog.ErrorContext(ctx, "sync: InsertMedals échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Medals), "err", err,
			)
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(fm.Medals)
		slog.DebugContext(ctx, "sync: médailles insérées",
			"match_id", fm.MatchID, "medals", len(fm.Medals),
		)
	}

	// Highlight events.
	if fm.HasHighlights && fm.HighlightData != nil {
		if err := insertHighlightEventsFromData(ctx, sharedDB, globalDB, fm.MatchID, fm.HighlightData, fm.FilmMajorVer, result); err != nil {
			slog.WarnContext(ctx, "sync: highlight_events insertion échouée",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", fm.MatchID, err))
		}
	} else if fm.HighlightError != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", fm.MatchID, fm.HighlightError))
	}

	// Player enrichment.
	if err := UpsertPlayerEnrichment(ctx, playerDB, fm.MatchID, ""); err != nil {
		slog.ErrorContext(ctx, "sync: UpsertPlayerEnrichment échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
		)
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	// PersonalScoreAwards (player DB, par joueur synchronisé). Non-bloquant :
	// un échec produit un warning, le sync continue.
	if len(fm.PSA) > 0 {
		if err := InsertPersonalScoreAwards(ctx, playerDB, fm.MatchID, e.xuid, fm.PSA); err != nil {
			slog.WarnContext(ctx, "sync: InsertPersonalScoreAwards échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("psa %s: %v", fm.MatchID, err))
		}
	}

	// CSR par-match (player DB). Renseigné par fetchMatchData uniquement pour
	// les matchs classés dont RankRecap était présent. Non-bloquant.
	if fm.CSRRow != nil {
		if err := UpsertCSRRow(ctx, playerDB, fm.CSRRow); err != nil {
			slog.WarnContext(ctx, "sync: UpsertCSRRow échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("csr %s: %v", fm.MatchID, err))
		} else {
			slog.DebugContext(ctx, "sync: CSR row écrite",
				"match_id", fm.MatchID, "tier", fm.CSRRow.Tier, "tier_label", fm.CSRRow.TierLabel,
			)
		}
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)
	return nil
}

// hasAnyTeamMMR retourne true si au moins un participant a team_mmr renseigné.
// Utilisé pour décider si MarkSkillLoaded doit être appelé après
// MergeSkillIntoParticipants (Phase 2 plan PLAN_BITMASKS_AUDIT_FIX).
func hasAnyTeamMMR(parts []ParticipantRow) bool {
	for _, p := range parts {
		if p.TeamMMR != nil {
			return true
		}
	}
	return false
}
