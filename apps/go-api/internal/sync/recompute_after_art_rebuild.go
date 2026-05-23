// Package sync — recompute_after_art_rebuild.go : orchestrateur de recompute
// post-rebuild ART (plan stabilisation 2026-05-22 §4.4).
//
// Contexte : pendant la période où l'index ART de match_participants était
// corrompu, les batchs computed sur des participants partiellement visibles
// (typiquement 1-2 rows sur 8-16 attendus) ont produit des résultats faux :
//   - LUSR cascade alimentée par des MMRs adversaires manquants → tier figé
//     (cas documenté : Madina97294 figé Argent IV au lieu de Platine attendu).
//   - performance_score sur fenêtre glissante incomplète.
//   - dominance_flag basé sur Steaktacular d'adversaires absents.
//   - is_with_friends raté car les xuids amis n'étaient pas dans la liste
//     visible des participants.
//
// Phase 4.1 répare l'ART au boot via swap CTAS, mais les valeurs dérivées
// restent figées sur l'état corrompu. Ce wrapper expose une API unique pour
// recalculer les 4 cascades force=true sur un joueur donné, à utiliser après
// un rebuild ART pour réparer les données stales en BDD.
//
// Trigger : appelé manuellement via outil CLI (sub-phase 4.4.b) ou
// automatiquement post-rebuild si un mécanisme d'auto-trigger est ajouté
// plus tard. Ce fichier ne définit que la fonction — pas de wiring.

package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// RecomputeAfterARTRebuildReport agrège les counts de chaque étape pour
// observabilité (logs, métriques, CLI tool).
type RecomputeAfterARTRebuildReport struct {
	XUID               string                 // joueur ciblé
	LUSRUpdated        int                    // matchs LUSR recalculés (force=true)
	PerformanceUpdated int                    // matchs performance_score recalculés
	DominanceMatches   int                    // matchs traités par BackfillDominanceFlags
	FriendsResult      FriendsRecomputeResult // résultat de RecomputeIsWithFriendsCore
	Duration           time.Duration          // wall-time total
	Errors             []error                // erreurs non-fatales par étape (best-effort)
}

// RecomputeAfterARTRebuild exécute en séquence les 4 cascades force=true pour
// un joueur, après un rebuild ART qui a libéré la table match_participants
// des rows masquées par la corruption d'index.
//
// Best-effort par étape : si une cascade échoue, l'erreur est accumulée
// dans report.Errors et on continue avec la suivante. Cette stratégie
// privilégie la récupération partielle (recompute partiel > pas de recompute
// du tout) — le caller décide de l'action selon report.Errors.
//
// friendGamertags : liste de gamertags amis pour la cascade is_with_friends.
// nil ou vide → skip cette étape (FriendsResult.FriendXUIDsCount = 0).
//
// Pré-conditions :
//   - playerDB ouvert en RW (cascade LUSR/perf/dominance écrivent dedans)
//   - sharedDB ouvert en RO (cascade lit match_participants + medals)
//   - aucun writer concurrent sur playerDB (la cascade LUSR est O(N) sur les
//     matchs et peut prendre plusieurs minutes sur 1000+ matchs)
func RecomputeAfterARTRebuild(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	friendGamertags []string,
) (RecomputeAfterARTRebuildReport, error) {
	start := time.Now()
	report := RecomputeAfterARTRebuildReport{XUID: xuid}

	slog.InfoContext(ctx, "recompute_after_art_rebuild: début",
		"xuid", xuid, "friends_provided", len(friendGamertags))

	// 1. LUSR cascade force=true.
	lusrUpdated, err := BatchComputeLUSR(ctx, playerDB, sharedDB, xuid, true)
	if err != nil {
		slog.ErrorContext(ctx, "recompute_after_art_rebuild: LUSR échoué (continue)",
			"xuid", xuid, "err", err)
		report.Errors = append(report.Errors, fmt.Errorf("LUSR: %w", err))
	}
	report.LUSRUpdated = lusrUpdated
	slog.InfoContext(ctx, "recompute_after_art_rebuild: LUSR done",
		"xuid", xuid, "updated", lusrUpdated)

	// 2. Performance scores force=true.
	perfUpdated, err := BatchComputePerformanceScores(ctx, playerDB, sharedDB, xuid, true)
	if err != nil {
		slog.ErrorContext(ctx, "recompute_after_art_rebuild: performance_score échoué (continue)",
			"xuid", xuid, "err", err)
		report.Errors = append(report.Errors, fmt.Errorf("performance: %w", err))
	}
	report.PerformanceUpdated = perfUpdated
	slog.InfoContext(ctx, "recompute_after_art_rebuild: performance done",
		"xuid", xuid, "updated", perfUpdated)

	// 3. Dominance flags. BackfillDominanceFlags n'a pas de mode "all" — on
	// charge la liste des matchIDs du joueur depuis shared (table-scan via
	// `xuid || '' = ?` pour court-circuiter ART au cas où la corruption
	// reviendrait avant le prochain reboot).
	matchIDs, err := loadAllMatchIDsForXUID(ctx, sharedDB, xuid)
	if err != nil {
		slog.ErrorContext(ctx, "recompute_after_art_rebuild: load matchIDs échoué (continue, skip dominance)",
			"xuid", xuid, "err", err)
		report.Errors = append(report.Errors, fmt.Errorf("load matchIDs: %w", err))
	} else if len(matchIDs) > 0 {
		if err := BackfillDominanceFlags(ctx, sharedDB, playerDB, xuid, matchIDs); err != nil {
			slog.ErrorContext(ctx, "recompute_after_art_rebuild: BackfillDominanceFlags échoué (continue)",
				"xuid", xuid, "err", err)
			report.Errors = append(report.Errors, fmt.Errorf("dominance: %w", err))
		}
		report.DominanceMatches = len(matchIDs)
	}
	slog.InfoContext(ctx, "recompute_after_art_rebuild: dominance done",
		"xuid", xuid, "matches", report.DominanceMatches)

	// 4. is_with_friends. Skip si liste vide (RecomputeIsWithFriendsCore le
	// fait déjà gracieusement, mais on évite l'appel pour la lisibilité log).
	if len(friendGamertags) > 0 {
		fr, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, xuid, friendGamertags, false)
		if err != nil {
			slog.ErrorContext(ctx, "recompute_after_art_rebuild: friends recompute échoué (continue)",
				"xuid", xuid, "err", err)
			report.Errors = append(report.Errors, fmt.Errorf("friends: %w", err))
		}
		report.FriendsResult = fr
	}
	slog.InfoContext(ctx, "recompute_after_art_rebuild: friends done",
		"xuid", xuid, "friend_xuids", report.FriendsResult.FriendXUIDsCount,
		"promoted", report.FriendsResult.MatchesPromoted)

	report.Duration = time.Since(start)
	slog.InfoContext(ctx, "recompute_after_art_rebuild: terminé",
		"xuid", xuid,
		"lusr", report.LUSRUpdated,
		"performance", report.PerformanceUpdated,
		"dominance_matches", report.DominanceMatches,
		"friends_promoted", report.FriendsResult.MatchesPromoted,
		"errors_count", len(report.Errors),
		"duration_ms", report.Duration.Milliseconds(),
	)

	// Si toutes les étapes ont échoué → erreur globale ; sinon best-effort.
	if len(report.Errors) > 0 && allCascadesFailed(report) {
		return report, fmt.Errorf("recompute_after_art_rebuild: toutes cascades échouées: %w",
			errors.Join(report.Errors...))
	}
	return report, nil
}

// loadAllMatchIDsForXUID charge la liste des match_ids où le xuid a participé.
// Utilise `xuid || ” = ?` pour forcer un table-scan (court-circuite l'index
// ART au cas où). Sans ORDER BY — la chronologie n'importe pas pour dominance.
func loadAllMatchIDsForXUID(ctx context.Context, sharedDB *sql.DB, xuid string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ?`, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadAllMatchIDsForXUID: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("loadAllMatchIDsForXUID scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// allCascadesFailed retourne true si chacune des 4 cascades a échoué (count=0
// ET au moins une erreur listée). Sert à décider si l'erreur retournée par
// RecomputeAfterARTRebuild doit être bloquante côté caller.
func allCascadesFailed(r RecomputeAfterARTRebuildReport) bool {
	return r.LUSRUpdated == 0 &&
		r.PerformanceUpdated == 0 &&
		r.DominanceMatches == 0 &&
		r.FriendsResult.MatchesPromoted == 0 &&
		len(r.Errors) >= 3
}
