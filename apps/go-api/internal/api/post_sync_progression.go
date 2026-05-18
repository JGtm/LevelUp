// Package api — post_sync_progression.go : orchestrateur de la couche
// progression V2 (Ascension), appelé après sync_handler succès.
//
// Pipeline complet en 7 étapes :
//
//  1. Build inputs : matchs récents (120j) + agrégats joueur + profile LUSR
//     + contexte comeback (2 derniers matchs)
//  2. streaks.Evaluate → résultats par type (daily_play, weekly_play, …)
//  3. records.Detect → résultats par (métrique × période)
//  4. milestones.Detect → résultats par entrée catalogue
//  5. coach.Generate → []Alert depuis tous les résultats + signaux propres
//     (LUSR tier approach, LOWESS positif, comeback welcome)
//  6. FilterRecent sur 24h via les notifs déjà émises
//  7. Pour chaque alerte restante : AnnotateDedupKey puis emitter.Emit
//
// Toutes les étapes sont idempotentes : ré-exécuter le pipeline sur les
// mêmes données ne produit pas de notifs en double (PB déjà persisté →
// pas de NewPB ; alerte déjà émise dans la fenêtre → filtrée).
//
// Réf : .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.2.

package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/coach"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/profile"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

const (
	// ProgressionMatchHistoryDays : fenêtre de matchs récents fournie aux
	// détecteurs streaks et records (décision §6 plan : 120j).
	ProgressionMatchHistoryDays = 120

	// ProgressionFreshnessThreshold : un match dont start_time est dans cette
	// fenêtre par rapport à `now` est considéré « tout juste synchronisé »
	// (utilisé pour HasNewActivity et le calcul comeback).
	ProgressionFreshnessThreshold = 6 * time.Hour

	// ProgressionDedupWindow : fenêtre pendant laquelle une même alerte
	// (catégorie + dedup_key) ne sera pas ré-émise.
	ProgressionDedupWindow = 24 * time.Hour

	// ProgressionRecentNotifsLimit : nombre de notifs récentes lues pour la
	// dédup. Largement supérieur au volume coach attendu sur 24h.
	ProgressionRecentNotifsLimit = 200
)

// ProgressionDeps regroupe les dépendances injectées dans l'orchestrateur.
// Permet de mocker en test sans construire toute la chaîne DI.
type ProgressionDeps struct {
	StreaksEvaluator    *streaks.Evaluator
	RecordsDetector     *records.Detector
	MilestonesDetector  *milestones.Detector
	ProfileService      *profile.Service
	CampaignService     *campaign.Service
	CoachGenerator      *coach.Generator
	Emitter             notifications.Emitter
	NotificationsRepo   notifications.Repository
}

// ProgressionResult capture l'état post-évaluation pour observabilité (tests
// + slog éventuel). Pas exposé dans la HTTP API.
type ProgressionResult struct {
	StreakResults    []streaks.EvaluationResult
	RecordResults    []records.DetectionResult
	MilestoneResults []milestones.DetectionResult
	AlertsGenerated  int
	AlertsEmitted    int
	AlertsDeduped    int
}

// EvaluateProgressionAfterSync exécute le pipeline complet. À appeler depuis
// la closure post-sync (post_sync_deltas.go). Signe les erreurs via slog
// mais ne fait jamais panic — feedback positif uniquement, jamais bloquant.
//
// `now` est injecté pour testabilité (en prod : time.Now().UTC()).
// `titleSlug` est attendu non vide (typ. "halo_infinite" pour ce projet).
//
// Retourne ProgressionResult pour observabilité ; les erreurs sont
// remontées seulement pour signaler des problèmes structurels (DB non
// attachée par exemple). Les erreurs individuelles de détecteurs sont
// loggées et n'arrêtent pas le pipeline.
func EvaluateProgressionAfterSync(
	ctx context.Context,
	pdb *duckdb.PlayerDB,
	titleSlug string,
	deps ProgressionDeps,
	now time.Time,
) (ProgressionResult, error) {
	res := ProgressionResult{}
	if pdb == nil {
		return res, errors.New("EvaluateProgressionAfterSync: nil PlayerDB")
	}
	if titleSlug == "" {
		return res, errors.New("EvaluateProgressionAfterSync: empty titleSlug")
	}

	userID := pdb.XUID // dans stats.duckdb, user_id = xuid du joueur courant.

	// ── 1. Build inputs ────────────────────────────────────────────────────
	activities, matchInputs, err := loadProgressionMatches(ctx, pdb, ProgressionMatchHistoryDays, now)
	if err != nil {
		slog.WarnContext(ctx, "progression: load matches", "slug", titleSlug, "err", err)
		return res, nil // dégrade gracieusement
	}
	playerStats, err := loadPlayerStats(ctx, pdb)
	if err != nil {
		slog.WarnContext(ctx, "progression: load stats", "slug", titleSlug, "err", err)
		return res, nil
	}
	cb, err := loadComebackContext(ctx, pdb, now, ProgressionFreshnessThreshold)
	if err != nil {
		slog.WarnContext(ctx, "progression: load comeback ctx", "slug", titleSlug, "err", err)
		// non bloquant
	}

	var prof *profile.PlayerProfile
	if deps.ProfileService != nil {
		p, err := deps.ProfileService.Load(ctx, userID, titleSlug, coach.LOWESSObservationWindow, now)
		if err != nil {
			slog.WarnContext(ctx, "progression: load profile", "err", err)
		} else {
			prof = p
		}
	}

	// Évaluation de la campagne active du joueur (si présente).
	// Re-compute LOWESS + Mann-Whitney U + R5 auto-closure check. Idempotent.
	if deps.CampaignService != nil {
		if err := deps.CampaignService.EvaluateActive(ctx, userID, titleSlug, now); err != nil {
			slog.WarnContext(ctx, "progression: evaluate campaign", "err", err)
		}
	}

	// ── 2. streaks ─────────────────────────────────────────────────────────
	if deps.StreaksEvaluator != nil {
		results, err := deps.StreaksEvaluator.Evaluate(ctx, streaks.EvaluateInput{
			UserID:    userID,
			TitleSlug: titleSlug,
			Now:       now,
			Matches:   activities,
			// Pour V1 du hook, on évalue les types universels (daily_play,
			// weekly_play) — pas de seuils personnels requis. Les types
			// perf-based seront activés quand PlayerProfile exposera les
			// médianes personnelles (commit follow-up).
			Thresholds: map[streaks.StreakType]float64{
				streaks.StreakTypeDailyPlay:  0,
				streaks.StreakTypeWeeklyPlay: 0,
			},
		})
		if err != nil {
			slog.WarnContext(ctx, "progression: streaks.Evaluate", "err", err)
		}
		res.StreakResults = results
	}

	// ── 3. records ─────────────────────────────────────────────────────────
	if deps.RecordsDetector != nil {
		results, err := deps.RecordsDetector.Detect(ctx, records.DetectInput{
			XUID:      pdb.XUID,
			UserID:    userID,
			TitleSlug: titleSlug,
			Now:       now,
			Matches:   matchInputs,
		})
		if err != nil {
			slog.WarnContext(ctx, "progression: records.Detect", "err", err)
		}
		res.RecordResults = results
	}

	// ── 4. milestones ──────────────────────────────────────────────────────
	if deps.MilestonesDetector != nil {
		results, err := deps.MilestonesDetector.Detect(ctx, milestones.DetectInput{
			UserID:    userID,
			TitleSlug: titleSlug,
			Now:       now,
			Stats:     playerStats,
		})
		if err != nil {
			slog.WarnContext(ctx, "progression: milestones.Detect", "err", err)
		}
		res.MilestoneResults = results
	}

	// ── 5. coach ───────────────────────────────────────────────────────────
	if deps.CoachGenerator == nil {
		return res, nil
	}
	coachInput := coach.GenerateInput{
		UserID:           userID,
		TitleSlug:        titleSlug,
		Now:              now,
		StreakResults:    res.StreakResults,
		RecordResults:    res.RecordResults,
		MilestoneResults: res.MilestoneResults,
		LastMatchAt:      cb.PrevMatchAt,
		HasNewActivity:   cb.HasNewActivity,
	}
	if prof != nil {
		coachInput.LUSR = coach.LUSRSnapshot{
			Mu:           prof.LUSR.Mu,
			NextTierName: prof.NextTier.Label,
			NextTierMu:   prof.NextTier.LowerMu,
		}
		if prof.MuTrend.IsPositive(coach.LOWESSObservationWindow) {
			coachInput.LOWESSTrends = []coach.LOWESSTrend{{
				Component: prof.MuTrend.Metric,
				Slope:     prof.MuTrend.Slope,
				Window:    prof.MuTrend.Window,
			}}
		}
	}
	alerts := deps.CoachGenerator.Generate(ctx, coachInput)
	res.AlertsGenerated = len(alerts)

	// ── 6. Dédup ───────────────────────────────────────────────────────────
	var recent []notifications.Notification
	if deps.NotificationsRepo != nil {
		lr, err := deps.NotificationsRepo.List(ctx, notifications.ListFilter{
			Limit: ProgressionRecentNotifsLimit,
		})
		if err != nil {
			slog.WarnContext(ctx, "progression: list recent notifs", "err", err)
		} else {
			recent = lr.Items
		}
	}
	deduped := coach.FilterRecent(alerts, recent, now, ProgressionDedupWindow)
	res.AlertsDeduped = res.AlertsGenerated - len(deduped)

	// ── 7. Émission ────────────────────────────────────────────────────────
	if deps.Emitter == nil {
		return res, nil
	}
	for i := range deduped {
		coach.AnnotateDedupKey(&deduped[i])
		input := deduped[i].ToEmitInput()
		if err := input.Validate(); err != nil {
			slog.WarnContext(ctx, "progression: invalid emit input", "type", deduped[i].Type, "err", err)
			continue
		}
		if err := deps.Emitter.Emit(ctx, input); err != nil {
			// ErrCategoryDisabled = pref off → silencieux, pas un échec.
			if errors.Is(err, notifications.ErrCategoryDisabled) {
				continue
			}
			slog.WarnContext(ctx, "progression: emit failed",
				"type", deduped[i].Type, "err", err)
			continue
		}
		res.AlertsEmitted++
	}
	return res, nil
}

// BuildPlayerProgressionDeps construit l'ensemble des dépendances de
// l'orchestrateur depuis un PlayerDB + emitter déjà résolu. Idempotent et
// peu coûteux (pas d'init lourde), peut être appelé à chaque sync.
//
// Retourne ProgressionDeps incomplet si le PlayerDB n'a pas toutes les DB
// attachées (Metadata ou SharedSocial). Dans ce cas le pipeline dégrade
// gracieusement (les détecteurs concernés sont skippés via AssertProgressionDeps
// au boot, ou silencieusement à l'évaluation).
func BuildPlayerProgressionDeps(pdb *duckdb.PlayerDB, emitter notifications.Emitter) ProgressionDeps {
	deps := ProgressionDeps{Emitter: emitter}
	if pdb == nil {
		return deps
	}
	if pdb.Player != nil {
		deps.StreaksEvaluator = streaks.NewEvaluator(duckdb.NewStreaksRepo(pdb.Player))
		deps.ProfileService = profile.NewService(pdb.Player)
		deps.CampaignService = campaign.NewService(
			duckdb.NewCampaignRepo(pdb.Player),
			duckdb.NewCampaignSampleProvider(pdb.Player),
		)
		// History repo (stats.duckdb) + PB repo (shared_social via pdb).
		history := duckdb.NewRecordHistoryRepo(pdb.Player)
		if pdb.SharedSocial != nil {
			pbRepo := duckdb.NewPersonalRecordsRepo(pdb)
			deps.RecordsDetector = records.NewDetector(pbRepo, history)
		}
		// Milestones : catalog dans Metadata, earned dans Player.
		earned := duckdb.NewMilestoneEarnedRepo(pdb.Player)
		if pdb.Metadata != nil {
			catalog := duckdb.NewMilestoneCatalogRepo(pdb.Metadata)
			deps.MilestonesDetector = milestones.NewDetector(catalog, earned)
		}
	}
	if pdb.SharedSocial != nil {
		deps.NotificationsRepo = duckdb.NewNotificationsRepo(pdb)
	}
	deps.CoachGenerator = coach.NewGenerator()
	return deps
}

// AssertProgressionDeps vérifie que toutes les dépendances sont câblées
// (utile au boot pour fail-fast plutôt qu'un slog.Warn à chaque sync).
// Retourne nil si tout est en place.
func AssertProgressionDeps(d ProgressionDeps) error {
	missing := []string{}
	if d.StreaksEvaluator == nil {
		missing = append(missing, "StreaksEvaluator")
	}
	if d.RecordsDetector == nil {
		missing = append(missing, "RecordsDetector")
	}
	if d.MilestonesDetector == nil {
		missing = append(missing, "MilestonesDetector")
	}
	if d.ProfileService == nil {
		missing = append(missing, "ProfileService")
	}
	if d.CampaignService == nil {
		missing = append(missing, "CampaignService")
	}
	if d.CoachGenerator == nil {
		missing = append(missing, "CoachGenerator")
	}
	if d.Emitter == nil {
		missing = append(missing, "Emitter")
	}
	if d.NotificationsRepo == nil {
		missing = append(missing, "NotificationsRepo")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("ProgressionDeps incomplete: missing %v", missing)
}
