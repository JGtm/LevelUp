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

package wire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/coach"
	"levelup/go-api/internal/progression/coach_advisor"
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

	// ProgressionRecentNotifsLimit : nombre de notifs récentes lues pour la
	// dédup. La fenêtre de dédup est désormais résolue PAR CATÉGORIE via
	// coach.DedupWindowFor (DP13 : jusqu'à 30 jours pour les nudges d'état).
	// La limite est un COMPTE (pas une borne temporelle) : 200 notifs couvrent
	// largement 30 jours au volume observé (~59 notifs / 6 semaines).
	ProgressionRecentNotifsLimit = 200
)

// ProgressionDeps regroupe les dépendances injectées dans l'orchestrateur.
// Permet de mocker en test sans construire toute la chaîne DI.
type ProgressionDeps struct {
	StreaksEvaluator   *streaks.Evaluator
	RecordsDetector    *records.Detector
	MilestonesDetector *milestones.Detector
	ProfileService     *profile.Service
	CampaignService    *campaign.Service
	CoachGenerator     *coach.Generator
	Emitter            notifications.Emitter
	NotificationsRepo  notifications.Repository

	// CoachAdvisor invoque la génération de proposals (ADR 0020 Phase 8).
	// Nil → étape ignorée gracieusement (pipeline reste fonctionnel).
	CoachAdvisor coach_advisor.Service
	// CoachProactiveMode est lu depuis settings.AppSettings.CoachProactiveMode.
	// False (défaut) → coach_advisor.GenerateProposals short-circuit en interne.
	CoachProactiveMode bool
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
	// ProposalsGenerated : nombre de proposals coach_advisor créées dans cette
	// invocation (Phase 8, ADR 0020). Inclut challenges + arcs.
	ProposalsGenerated int
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

	res.StreakResults = evaluateStreaks(ctx, deps, userID, titleSlug, now, activities)
	res.RecordResults = detectRecords(ctx, deps, pdb.XUID, userID, titleSlug, now, matchInputs)
	res.MilestoneResults = detectMilestones(ctx, deps, userID, titleSlug, now, playerStats)

	if deps.CoachGenerator == nil {
		return res, nil
	}
	alerts := generateCoachAlerts(ctx, deps, userID, titleSlug, now, &res, cb, prof, activities)
	res.AlertsGenerated = len(alerts)

	deduped := dedupCoachAlerts(ctx, deps, alerts, now)
	res.AlertsDeduped = res.AlertsGenerated - len(deduped)

	if deps.Emitter == nil {
		return res, nil
	}
	res.AlertsEmitted = emitCoachAlerts(ctx, deps, deduped)

	// ── 8. Coach Advisor — proposals Prestige (Phase 8, ADR 0020) ──────────
	// Lit les alertes coach (avant ou après dédup peu importe — le coach_advisor
	// applique sa propre supersession). Best-effort : toute erreur loggée
	// sans interrompre le pipeline.
	res.ProposalsGenerated = generateCoachAdvisorProposals(ctx, deps, userID, titleSlug, now, alerts)
	return res, nil
}

// generateCoachAdvisorProposals exécute l'étape 8 du pipeline post-sync :
// traduit les alertes coach en Signals et invoque GenerateProposals.
//
// Retourne 0 si CoachAdvisor=nil ou si proactive disabled (short-circuit
// interne de GenerateProposals).
func generateCoachAdvisorProposals(
	ctx context.Context, deps ProgressionDeps,
	userID, titleSlug string, now time.Time, alerts []coach.Alert,
) int {
	if deps.CoachAdvisor == nil {
		return 0
	}
	signals := coach_advisor.SignalsFromAlerts(alerts)
	if len(signals) == 0 {
		return 0
	}
	proposals, err := deps.CoachAdvisor.GenerateProposals(ctx, coach_advisor.GenerateInput{
		UserID:           userID,
		TitleSlug:        titleSlug,
		Now:              now,
		ProactiveEnabled: deps.CoachProactiveMode,
		Signals:          signals,
	})
	if err != nil {
		slog.WarnContext(ctx, "progression: coach_advisor.GenerateProposals",
			"err", err, "user", userID, "titleSlug", titleSlug)
		return 0
	}
	return len(proposals)
}

// evaluateStreaks invoque deps.StreaksEvaluator si présent. Best-effort.
func evaluateStreaks(
	ctx context.Context, deps ProgressionDeps,
	userID, titleSlug string, now time.Time, activities []streaks.MatchActivity,
) []streaks.EvaluationResult {
	if deps.StreaksEvaluator == nil {
		return nil
	}
	// V2 §4 : médiane personnelle KDA sur la fenêtre 120j servant de seuil
	// pour les streaks perf-based (daily_perf, weekly_kda_threshold).
	kdaMedian := medianKDA(activities)
	results, err := deps.StreaksEvaluator.Evaluate(ctx, streaks.EvaluateInput{
		UserID:    userID,
		TitleSlug: titleSlug,
		Now:       now,
		Matches:   activities,
		Thresholds: map[streaks.StreakType]float64{
			streaks.StreakTypeDailyPlay:          0,
			streaks.StreakTypeWeeklyPlay:         0,
			streaks.StreakTypeDailyPerf:          kdaMedian,
			streaks.StreakTypeWeeklyKDAThreshold: kdaMedian,
		},
	})
	if err != nil {
		slog.WarnContext(ctx, "progression: streaks.Evaluate", "err", err)
	}
	return results
}

// detectRecords invoque deps.RecordsDetector si présent. Best-effort.
func detectRecords(
	ctx context.Context, deps ProgressionDeps,
	xuid, userID, titleSlug string, now time.Time, matchInputs []records.MatchInput,
) []records.DetectionResult {
	if deps.RecordsDetector == nil {
		return nil
	}
	results, err := deps.RecordsDetector.Detect(ctx, records.DetectInput{
		XUID:      xuid,
		UserID:    userID,
		TitleSlug: titleSlug,
		Now:       now,
		Matches:   matchInputs,
	})
	if err != nil {
		slog.WarnContext(ctx, "progression: records.Detect", "err", err)
	}
	return results
}

// detectMilestones invoque deps.MilestonesDetector si présent. Best-effort.
func detectMilestones(
	ctx context.Context, deps ProgressionDeps,
	userID, titleSlug string, now time.Time, playerStats milestones.PlayerStats,
) []milestones.DetectionResult {
	if deps.MilestonesDetector == nil {
		return nil
	}
	results, err := deps.MilestonesDetector.Detect(ctx, milestones.DetectInput{
		UserID:    userID,
		TitleSlug: titleSlug,
		Now:       now,
		Stats:     playerStats,
	})
	if err != nil {
		slog.WarnContext(ctx, "progression: milestones.Detect", "err", err)
	}
	return results
}

// generateCoachAlerts construit l'input coach et invoque le CoachGenerator.
func generateCoachAlerts(
	ctx context.Context, deps ProgressionDeps,
	userID, titleSlug string, now time.Time,
	res *ProgressionResult, cb comebackContext, prof *profile.PlayerProfile,
	activities []streaks.MatchActivity,
) []coach.Alert {
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
	mOC := medianOC(activities)
	mDR := medianDR(activities)
	if mOC > 0 || mDR > 0 {
		var residualSum float64
		var residualCount int
		for _, a := range activities {
			if v, ok := a.Stats["engagement_score_brut"]; ok {
				residualSum += v
				residualCount++
			}
		}
		cm := &coach.CombatMedians{
			MedianOC:    mOC,
			MedianDR:    mDR,
			HasResidual: residualCount >= 10,
		}
		if residualCount > 0 {
			cm.AvgResidual = residualSum / float64(residualCount)
		}
		coachInput.CombatMedians = cm
		slog.DebugContext(ctx, "progression: combat medians computed",
			"user", userID, "median_oc", mOC, "median_dr", mDR,
			"residual_count", residualCount, "avg_residual", cm.AvgResidual)
	}
	return deps.CoachGenerator.Generate(ctx, coachInput)
}

// dedupCoachAlerts filtre les alertes par déduplication temporelle.
func dedupCoachAlerts(
	ctx context.Context, deps ProgressionDeps, alerts []coach.Alert, now time.Time,
) []coach.Alert {
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
	return coach.FilterRecent(alerts, recent, now, coach.DedupWindowFor)
}

// emitCoachAlerts émet les alertes via deps.Emitter. Retourne le nombre émis.
// ErrCategoryDisabled (pref off) est silencieux et n'incrémente pas le compteur.
func emitCoachAlerts(ctx context.Context, deps ProgressionDeps, deduped []coach.Alert) int {
	emitted := 0
	for i := range deduped {
		coach.AnnotateDedupKey(&deduped[i])
		input := deduped[i].ToEmitInput()
		if err := input.Validate(); err != nil {
			slog.WarnContext(ctx, "progression: invalid emit input", "type", deduped[i].Type, "err", err)
			continue
		}
		if err := deps.Emitter.Emit(ctx, input); err != nil {
			if errors.Is(err, notifications.ErrCategoryDisabled) {
				continue
			}
			slog.WarnContext(ctx, "progression: emit failed",
				"type", deduped[i].Type, "err", err)
			continue
		}
		emitted++
	}
	return emitted
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
		// V2 §4 : LeverageProvider câblé depuis le ProfileService pour
		// permettre à campaign.Evaluate de checker R5 condition 2
		// ("axe sort des leviers prioritaires").
		deps.CampaignService = campaign.NewService(
			duckdb.NewCampaignRepo(pdb.Player),
			duckdb.NewCampaignSampleProvider(pdb),
		).WithLeverageProvider(newProfileLeverageProvider(pdb))
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

// BuildPlayerProgressionDepsWithAdvisor étend BuildPlayerProgressionDeps avec
// le coach_advisor (Phase 8, ADR 0020). Tous les paramètres advisor sont
// optionnels — si advisorBundle ou prestigeBundle vaut nil, l'étape 8 est
// gracefully skippée.
//
// `proactiveMode` est lu depuis settings.AppSettings.CoachProactiveMode au
// moment de l'invocation (rebuild par sync → toggle dynamique sans restart).
func BuildPlayerProgressionDepsWithAdvisor(
	pdb *duckdb.PlayerDB,
	emitter notifications.Emitter,
	advisorBundle *CoachAdvisorBundle,
	prestigeBundle *PrestigeBundle,
	proactiveMode bool,
	playerSlug string,
) ProgressionDeps {
	deps := BuildPlayerProgressionDeps(pdb, emitter)
	if advisorBundle == nil || prestigeBundle == nil || pdb == nil || pdb.Player == nil {
		return deps
	}
	templates := prestigeBundle.TemplateRepoForCoach()
	if templates == nil {
		return deps
	}
	prestigeSvc, err := prestigeBundle.ServiceForPlayer(context.Background(), playerSlug)
	if err != nil {
		slog.Warn("coach_advisor: prestige service not available, advisor disabled",
			"player", playerSlug, "err", err)
		return deps
	}
	deps.CoachAdvisor = advisorBundle.ServiceForPlayer(pdb, templates, prestigeSvc)
	deps.CoachProactiveMode = proactiveMode
	return deps
}

// BuildProgressionAfterSyncHook retourne la closure injectée dans
// config.AppConfig.ProgressionAfterSync (mirror de BuildTitleReadyNotifier),
// appelée par le Runner live d'un titre (Halo 5+) après un cycle qui a inséré des
// matchs. Fait tourner le pipeline Progression V2 (streaks/records/milestones/coach)
// pour CE titre, comme le post-sync HINF (BuildPostSyncDeltaHook) — mais avec les
// deps de BASE (BuildPlayerProgressionDeps, SANS le PrestigeBundle/CoachAdvisor
// mono-titre, gracieusement absents pour un 2e titre). Best-effort, title-agnostic.
func BuildProgressionAfterSyncHook(reg *ServiceRegistry, cfg *config.AppConfig) func(ctx context.Context, titleSlug, playerSlug string) {
	return func(ctx context.Context, titleSlug, playerSlug string) {
		resCtx := ctxkeys.WithTitleSlug(ctx, titleSlug)
		pdb, err := reg.resolve(resCtx, playerSlug)
		if err != nil {
			slog.WarnContext(ctx, "progression after-sync: resolve",
				"titleSlug", titleSlug, "player", playerSlug, "err", err)
			return
		}
		// Emitter best-effort : si indisponible, le pipeline CALCULE (streaks/records/
		// milestones persistés) sans émettre de notif (deps.Emitter == nil → skip émission).
		emitter, eerr := reg.NotificationsEmitter(resCtx, playerSlug)
		if eerr != nil {
			emitter = nil
		}
		deps := BuildPlayerProgressionDeps(pdb, emitter)
		if _, err := EvaluateProgressionAfterSync(ctx, pdb, titleSlug, deps, time.Now().UTC()); err != nil {
			slog.WarnContext(ctx, "progression after-sync: evaluate", "titleSlug", titleSlug, "err", err)
		}
	}
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

// profileLeverageProvider adapte profile.Service.BuildProfile en un
// campaign.LeverageProvider. V2 §4 — R5 condition 2.
//
// Optimisation : on construit un profil minimaliste (fenêtre 30j default)
// puis on extrait les composantes des leviers identifiés. Le coût est ~1
// query agrégée + 8 round-trips trend (cf. profile.computeComponentTrend).
// Accepté dans le hook post-sync : le hook tourne après ingestion d'un
// match, latence ajoutée ~50-200ms négligeable face au sync HTTP.
type profileLeverageProvider struct {
	pdb *duckdb.PlayerDB
}

func newProfileLeverageProvider(pdb *duckdb.PlayerDB) *profileLeverageProvider {
	return &profileLeverageProvider{pdb: pdb}
}

func (p *profileLeverageProvider) CurrentLeverageComponents(
	ctx context.Context, userID, titleSlug string,
) ([]string, error) {
	svc := profile.NewServiceFromPlayerDB(p.pdb)
	prof, err := svc.BuildProfile(ctx, userID, titleSlug, 30, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(prof.Leverages))
	for _, lv := range prof.Leverages {
		out = append(out, lv.Component)
	}
	return out, nil
}

// medianKDA retourne la médiane des valeurs `kda` extraites des
// MatchActivity.Stats sur la fenêtre fournie. Retourne 0 si < 10 matchs
// (significativité insuffisante pour servir de seuil personnel).
//
// V2 §4 : sert de threshold pour les streaks perf-based daily_perf et
// weekly_kda_threshold. La médiane est robuste aux outliers (matchs à
// KDA aberrant) — préférée à la moyenne pour ce cas d'usage.
func medianKDA(activities []streaks.MatchActivity) float64 {
	return medianStat(activities, "kda")
}

func medianOC(activities []streaks.MatchActivity) float64 {
	return medianStat(activities, "oc")
}

func medianDR(activities []streaks.MatchActivity) float64 {
	return medianStat(activities, "dr")
}

// medianStat calcule la médiane d'une stat nommée parmi les activités.
// Retourne 0 si moins de 10 valeurs disponibles.
func medianStat(activities []streaks.MatchActivity, key string) float64 {
	values := make([]float64, 0, len(activities))
	for _, a := range activities {
		if v, ok := a.Stats[key]; ok && v > 0 {
			values = append(values, v)
		}
	}
	if len(values) < 10 {
		return 0
	}
	return analysis.MedianFloat(values)
}
