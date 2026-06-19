// Package sync — engine_process_match.go : méthode legacy processMatch (séquentielle).
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - processMatch : pipeline séquentiel par match (registry + participants + skill
//   - CSR + medals + highlight events + player enrichment + PSA).
//
// Le code de production utilise désormais le path parallèle fetchMatchData +
// insertFetchedMatch (cf. engine_fetch.go). processMatch reste utilisé par
// les tests engine_e2e_test.go pour valider le comportement séquentiel
// historique. Comportement INCHANGÉ — pur déplacement.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// processMatch récupère, transforme et insère un match dans les deux DBs.
func (e *SyncEngine) processMatch(
	ctx context.Context,
	client HaloClient,
	sharedDB, playerDB *sql.DB,
	result *domain.SyncResult,
	matchID string,
	opts domain.SyncOptions,
) error {
	start := time.Now()
	slog.DebugContext(ctx, "processMatch: début", "gamertag", e.gamertag, "match_id", matchID)

	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "processMatch: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("GetMatchStats: %w", err)
	}

	// ─── match_registry ────────────────────────────────────────────────────────
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "processMatch: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("ExtractRegistry: %w", err)
	}
	// Enrichissement post-Extract : résout les UUIDs bruts en noms canoniques
	// via metadata.asset_translations[en-US] AVANT l'INSERT, pour ne pas
	// stocker `playlist_name = playlist_id` quand l'API Halo n'a pas retourné
	// de PublicName. Best-effort : nil metaDB → no-op (préserve le fallback
	// historique). Cf. thought_log 2026-05-09.
	if err := EnrichRegistryFromMetadata(ctx, e.metaDB, reg); err != nil {
		slog.WarnContext(ctx, "processMatch: EnrichRegistryFromMetadata non-bloquant",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
	}
	if err := InsertRegistryIfNotExists(ctx, sharedDB, *reg); err != nil {
		slog.ErrorContext(ctx, "processMatch: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// ─── match_participants ────────────────────────────────────────────────────
	if opts.WithParticipants {
		participants := ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé.
		ensureGamertagForSelf(participants, e.xuid, e.gamertag)

		// Skill API (séparé du stats endpoint) : team_mmr, enemy_mmr, kills/deaths_expected.
		// Non-bloquant : un échec produit un warning mais le sync continue.
		if xuids := ParticipantXUIDs(participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				slog.WarnContext(ctx, "processMatch: GetMatchSkill échoué (continuing without skill)",
					"gamertag", e.gamertag, "match_id", matchID, "err", skillErr,
				)
				result.Warnings = append(result.Warnings, fmt.Sprintf("skill %s: %v", matchID, skillErr))
			} else if len(skillData) > 0 {
				participants = MergeSkillIntoParticipants(participants, skillData)
				slog.DebugContext(ctx, "processMatch: skill merged",
					"match_id", matchID, "players_with_skill", len(skillData),
				)
				// CSR par-match : pour les matchs classés, le payload skill
				// contient RankRecap.PostMatchCsr. On persiste côté player DB.
				// Non-bloquant : tout échec laisse le sync continuer.
				if row := ExtractCSRRowIfRanked(reg, skillData[e.xuid]); row != nil {
					if csrErr := UpsertCSRRow(ctx, playerDB, row); csrErr != nil {
						slog.WarnContext(ctx, "processMatch: UpsertCSRRow échoué",
							"gamertag", e.gamertag, "match_id", matchID, "err", csrErr,
						)
					} else {
						slog.DebugContext(ctx, "processMatch: CSR row écrite",
							"match_id", matchID, "tier", row.Tier, "tier_label", row.TierLabel,
						)
					}
				}
			}
		}

		if err := InsertParticipants(ctx, sharedDB, participants); err != nil {
			slog.ErrorContext(ctx, "processMatch: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", matchID, "count", len(participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(participants)

		// Alias xuid→gamertag : plus d'upsert vers le store global xbox_aliases
		// (consolidé dans shared 2026-06-19). Les gamertags sont déjà en
		// shared.match_participants (InsertParticipants ci-dessus) que lit
		// v_gamertag_lookup, et le chemin convergent upserte shared.xuid_aliases.
		slog.DebugContext(ctx, "processMatch: participants insérés",
			"match_id", matchID, "participants", len(participants),
		)
	}

	// ─── medals_earned ─────────────────────────────────────────────────────────
	if opts.WithMedals {
		medals := ExtractMedals(matchJSON)
		if err := InsertMedals(ctx, sharedDB, medals); err != nil {
			slog.ErrorContext(ctx, "processMatch: InsertMedals échoué",
				"gamertag", e.gamertag, "match_id", matchID, "count", len(medals), "err", err,
			)
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(medals)
		slog.DebugContext(ctx, "processMatch: médailles insérées",
			"match_id", matchID, "medals", len(medals),
		)
	}

	// ─── highlight_events + killer_victim_pairs ──────────────────────────────────────
	if opts.WithHighlightEvents {
		if err := ProcessHighlightEvents(ctx, client, sharedDB, matchID, result); err != nil {
			// Non-bloquant : on logge et on continue (pas de return).
			slog.WarnContext(ctx, "processMatch: highlight_events non chargés",
				"gamertag", e.gamertag, "match_id", matchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", matchID, err))
		}
	}

	// ─── player_match_enrichment (player DB) ───────────────────────────────────
	if err := UpsertPlayerEnrichment(ctx, playerDB, matchID, ""); err != nil {
		slog.ErrorContext(ctx, "processMatch: UpsertPlayerEnrichment échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	// ─── personal_score_awards (player DB) ─────────────────────────────────────
	psaRows := ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if len(psaRows) > 0 {
		if err := InsertPersonalScoreAwards(ctx, playerDB, matchID, e.xuid, psaRows); err != nil {
			slog.WarnContext(ctx, "processMatch: InsertPersonalScoreAwards échoué",
				"gamertag", e.gamertag, "match_id", matchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("psa %s: %v", matchID, err))
		}
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, matchID)
	slog.DebugContext(ctx, "processMatch: terminé",
		"gamertag", e.gamertag, "match_id", matchID,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}
