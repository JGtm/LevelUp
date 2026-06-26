// Package sync — enrichment_backfill.go : reconstruit l'enrichissement PAR JOUEUR
// d'un titre dont le sync est shared-only (typiquement Halo 5), à partir du shared
// déjà rempli.
//
// CONTEXTE : le runner live Halo 5 (internal/games/halo_5/livesync) capture + persiste
// UNIQUEMENT le shared (match_registry / match_participants / medals / events / ...).
// Il ne passe jamais par le post-sync Infinite (runScoringSteps), donc la player DB
// h5 n'existe pas et `player_match_enrichment` reste vide → aucun performance_score,
// session_id, is_with_friends, engagement, dominance côté joueur.
//
// Les FONCTIONS de calcul sont déjà title-agnostic (elles lisent le shared + écrivent
// la player DB par xuid). Seul le CALL-SITE était Infinite-only. Cette fonction
// reproduit la séquence du post-sync (runScoringSteps + ensure rows + dominance +
// friends + aggregates) hors LUSR — le LUSR est recalculé séparément (cmd/h5-lusr-backfill),
// APRÈS la classification ranked, pour ne pas mêler de l'Arena classé au rating social.
//
// Distinct de RecomputeAfterARTRebuild (qui suppose les rows enrichment déjà
// présentes et fait des UPDATE force=true) : ici la player DB est VIERGE, donc
// ensurePlayerEnrichmentRows DOIT s'exécuter en premier — sinon tous les recomputes,
// composés d'UPDATE, sont des no-op silencieux (cf. incident 2026-05-27).

package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
)

// EnrichmentBackfillReport agrège les counts de chaque étape (logs + CLI).
type EnrichmentBackfillReport struct {
	XUID                string
	BaselineRowsCreated int                    // rows player_match_enrichment créées (ensurePlayerEnrichmentRows)
	HadBotUpdated       int                    // matchs flaggés had_bot_teammate
	SessionsAssigned    int                    // matchs avec session_id assigné
	PerfComputed        int                    // performance_score calculés
	EngagementComputed  int                    // engagement_score calculés (0 si colonnes absentes)
	CoefsUpdated        int                    // coefficients d'engagement recalculés
	AssistsModes        int                    // modes du modèle d'assists calculés
	DisciplineAwards    int                    // lignes PSA discipline (suicides/trahisons) écrites — H5
	DominanceMatches    int                    // matchs traités par BackfillDominanceFlags
	FriendsResult       FriendsRecomputeResult // is_with_friends
	AggregatesCreated   int                    // vues mv_* (re)créées
	Duration            time.Duration
	Errors              []error // erreurs non-fatales par étape (best-effort)
}

// BackfillEnrichmentFromShared crée + peuple l'enrichissement par joueur depuis le
// shared, pour un xuid donné. Hors LUSR (géré séparément). Best-effort par étape :
// une étape qui échoue accumule son erreur dans report.Errors et n'interrompt pas
// la suite (récupération partielle privilégiée).
//
// Pré-conditions :
//   - playerDB ouvert en RW (provisionné via migrations TargetPlayer — schéma PME présent)
//   - sharedDB ouvert en RO/RW (lecture match_participants + medals + highlight_events)
//   - aucun writer concurrent (sessions/perf/dominance écrivent la player DB)
//
// friendGamertags : pour is_with_friends + le mode session squad. Vide → ces étapes
// dégradent gracieusement (is_with_friends non promu, sessions en mode teammates).
//
// force : true = recompute complet (backfill initial CLI) ; false = incrémental
// (hook live post-sync — seuls les matchs sans score calculé sont traités, comme
// runScoringSteps Infinite). ensurePlayerEnrichmentRows crée toujours les rows
// baseline manquantes (le delta est vide si tout existe déjà).
func BackfillEnrichmentFromShared(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	friendGamertags []string,
	force bool,
) (EnrichmentBackfillReport, error) {
	start := time.Now()
	report := EnrichmentBackfillReport{XUID: xuid}

	slog.InfoContext(ctx, "enrichment_backfill: début",
		"xuid", xuid, "friends_provided", len(friendGamertags))

	track := func(step string, err error) {
		if err != nil {
			slog.ErrorContext(ctx, "enrichment_backfill: étape échouée (continue)",
				"step", step, "xuid", xuid, "err", err)
			report.Errors = append(report.Errors, fmt.Errorf("%s: %w", step, err))
		}
	}

	// -2. Lignes baseline. INDISPENSABLE EN PREMIER : sur une player DB vierge,
	// sans row baseline, tous les UPDATE en aval sont des no-op silencieux.
	if n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, xuid); err != nil {
		track("ensure_rows", err)
	} else {
		report.BaselineRowsCreated = n
	}
	slog.InfoContext(ctx, "enrichment_backfill: baseline rows", "xuid", xuid, "created", report.BaselineRowsCreated)

	// -0.3 had_bot_teammate (dérivé des participants).
	if n, err := computeAndPersistHadBotTeammate(ctx, playerDB, sharedDB, xuid); err != nil {
		track("had_bot", err)
	} else {
		report.HadBotUpdated = n
	}

	// 0. Sessions (regroupement temporel) — opts par défaut, amis pour le mode squad.
	if n, err := recalculateSessionsInline(ctx, playerDB, sharedDB, xuid, analysis.DefaultSessionOptions(), friendGamertags); err != nil {
		track("sessions", err)
	} else {
		report.SessionsAssigned = n
	}

	// 1. Performance scores (force selon le mode : backfill=true, live=false).
	if n, err := BatchComputePerformanceScores(ctx, playerDB, sharedDB, xuid, force); err != nil {
		track("performance", err)
	} else {
		report.PerfComputed = n
	}

	// 1.5 Engagement scores — self-skip si colonnes engagement absentes du schéma.
	if n, err := batchComputeEngagementScores(ctx, playerDB, sharedDB, xuid, force); err != nil {
		track("engagement", err)
	} else {
		report.EngagementComputed = n
	}

	// 1.5.b Coefficients d'engagement (sinon coef figé à 1.0 → courbes superposées).
	if n, err := batchRecomputeCoefficients(ctx, playerDB, xuid); err != nil {
		track("engagement_coefs", err)
	} else {
		report.CoefsUpdated = n
	}

	// 1.52 Modèle d'assists (OLS per-mode) — self-skip si migration absente.
	if n, err := batchComputePlayerAssistsModel(ctx, playerDB, sharedDB, xuid, false); err != nil {
		track("assists_model", err)
	} else {
		report.AssistsModes = n
	}

	// 1.55 Discipline (suicides/trahisons) → PSA self_destruction/betrayed_player,
	// dérivés du shared (killer_victim_pairs + teams). Parité Infinite « fun stats ».
	// H5-ONLY (cette fonction n'est appelée que par l'enrich H5) ; ne JAMAIS l'utiliser
	// pour un titre à PSA natif (double-compte). Self-skip si schéma PSA absent.
	if n, err := computeAndPersistH5DisciplineAwards(ctx, playerDB, sharedDB, xuid); err != nil {
		track("discipline_awards", err)
	} else {
		report.DisciplineAwards = n
	}

	// 1.7 Dominance flags. Pas de mode "all" → on charge la liste des matchs du joueur.
	if matchIDs, err := loadAllMatchIDsForXUID(ctx, sharedDB, xuid); err != nil {
		track("load_match_ids", err)
	} else if len(matchIDs) > 0 {
		if err := BackfillDominanceFlags(ctx, sharedDB, playerDB, xuid, matchIDs); err != nil {
			track("dominance", err)
		} else {
			report.DominanceMatches = len(matchIDs)
		}
	}

	// 3.5 is_with_friends — skip explicite si pas d'amis fournis (lisibilité log).
	if len(friendGamertags) > 0 {
		fr, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, xuid, friendGamertags, false)
		if err != nil {
			track("friends", err)
		}
		report.FriendsResult = fr
	}

	// 4. Vues matérialisées (mv_*).
	if created, _, err := RefreshAggregates(ctx, playerDB); err != nil {
		track("aggregates", err)
	} else {
		report.AggregatesCreated = created
	}

	report.Duration = time.Since(start)
	slog.InfoContext(ctx, "enrichment_backfill: terminé",
		"xuid", xuid,
		"baseline_rows", report.BaselineRowsCreated,
		"had_bot", report.HadBotUpdated,
		"sessions", report.SessionsAssigned,
		"performance", report.PerfComputed,
		"engagement", report.EngagementComputed,
		"coefs", report.CoefsUpdated,
		"dominance_matches", report.DominanceMatches,
		"friends_promoted", report.FriendsResult.MatchesPromoted,
		"aggregates", report.AggregatesCreated,
		"errors_count", len(report.Errors),
		"duration_ms", report.Duration.Milliseconds(),
	)

	// Erreur globale seulement si l'étape baseline (linchpin) a échoué ET aucune row
	// n'existe — sinon best-effort (le reste est dégradable).
	if report.BaselineRowsCreated == 0 && len(report.Errors) > 0 && baselineFailed(report) {
		return report, fmt.Errorf("enrichment_backfill: étape baseline échouée: %w", errors.Join(report.Errors...))
	}
	return report, nil
}

// baselineFailed retourne true si l'erreur ensure_rows est présente (le linchpin a
// cassé → les autres étapes n'ont rien pu faire de toute façon).
func baselineFailed(r EnrichmentBackfillReport) bool {
	for _, e := range r.Errors {
		if e != nil && len(e.Error()) >= len("ensure_rows") && e.Error()[:len("ensure_rows")] == "ensure_rows" {
			return true
		}
	}
	return false
}
