// Package service — backfill_orchestrator.go : pipeline de backfill multi-phase.
//
// Extrait du handler HTTP (K1f, 2026-07-06) : handlers.handleStartBackfill ne garde
// que validation (400/404/409) + création du job + 202. Toute l'orchestration métier
// (détection des matchs manquants puis N phases de backfill best-effort + résumé
// final) vit ici — anti-pattern « logique métier dans un handler » (CLAUDE.md §7).
// Extraction FIDÈLE : aucune logique ni ordre de phase changé, mêmes libellés d'étape
// et mêmes chaînes de warning/résumé.
package service

import (
	"context"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	go_sync "levelup/go-api/internal/sync"
)

// BackfillOrchestrator exécute le pipeline de backfill pour un joueur. Construit par
// le handler avec un SyncEngine déjà wiré (RepoRoot, gamertag, xuid, tokens, provider).
type BackfillOrchestrator struct {
	engine   *go_sync.SyncEngine
	jobStore *jobs.Store
	scope    *go_sync.SyncScope
	tokens   *domain.HaloTokens
	dryRun   bool
}

// NewBackfillOrchestrator assemble l'orchestrateur. engine doit être déjà construit
// (WithSharedProvider appliqué si disponible) par l'appelant.
func NewBackfillOrchestrator(engine *go_sync.SyncEngine, jobStore *jobs.Store,
	scope *go_sync.SyncScope, tokens *domain.HaloTokens, dryRun bool) *BackfillOrchestrator {
	return &BackfillOrchestrator{engine: engine, jobStore: jobStore, scope: scope, tokens: tokens, dryRun: dryRun}
}

// backfillCounts agrège les compteurs de chaque phase pour le résumé final.
type backfillCounts struct {
	total                  int
	weaponsInserted        int
	psaMatchesUpdated      int
	psaRowsInserted        int
	engagementComputed     int
	engagementCoefsUpdated int
	eventsHealed           int
	eventsTotal            int
	lusrUpdated            int
	csrInserted            int
	csrSkipped             int
	perfUpdated            int
	citationsUpdated       int
	comebackUpdated        int
}

// setStep met à jour l'étape courante affichée du job.
func (o *BackfillOrchestrator) setStep(jobID, step string) {
	o.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		s := step
		j.CurrentStep = &s
	})
}

// warn ajoute un avertissement non-fatal au job (phases best-effort).
func (o *BackfillOrchestrator) warn(jobID, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	o.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Warnings = append(j.Warnings, msg)
	})
}

// Run exécute le pipeline complet (appelé en goroutine par le handler). L'état vit
// dans le job du store ; Run ne retourne rien.
func (o *BackfillOrchestrator) Run(jobID string) {
	runStep := "Détection des données manquantes"
	o.jobStore.SetStatus(jobID, domain.JobStatusRunning, &runStep)

	// ── Phase 1 : détection ──────────────────────────────────────────
	missing, err := o.engine.RunBackfill(context.Background(), o.scope)
	if err != nil {
		errMsg := err.Error()
		o.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusFailed
			j.Error = &domain.JobErrorDetail{Code: "detection_error", Message: errMsg}
		})
		return
	}

	var c backfillCounts
	c.total = len(missing)
	pct := 50
	detStep := fmt.Sprintf("Détection terminée : %d match(s) à traiter", c.total)
	o.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.MatchesTotal = &c.total
		j.ProgressPct = &pct
		j.CurrentStep = &detStep
	})

	// Phases 1.5 + 1.6 : indépendantes de la détection, AVANT l'early-return total==0.
	o.runCitationsComeback(jobID, &c)

	if o.dryRun || c.total == 0 {
		done := fmt.Sprintf("Terminé (dry_run=%v, %d match(s) détectés, citations: %d, comeback: %d)", o.dryRun, c.total, c.citationsUpdated, c.comebackUpdated)
		pct100 := 100
		o.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct100
			j.CurrentStep = &done
		})
		return
	}

	o.runWeaponsEngagement(jobID, missing, &c)
	o.runEventsLusr(jobID, missing, &c)
	o.runCsrPerfPsa(jobID, missing, &c)

	// ── Phase 5 : types non encore implémentés → avertissement ──────
	o.warnUnimplemented(jobID)

	done := fmt.Sprintf(
		"Backfill terminé — matchs: %d, weapon kills insérés: %d, psa: %d match(s)/%d rows, engagement: %d, events healed: %d (%d events insérés), lusr: %d, csr: %d (skipped: %d), perf: %d, citations: %d, comeback: %d",
		c.total, c.weaponsInserted, c.psaMatchesUpdated, c.psaRowsInserted, c.engagementComputed, c.eventsHealed, c.eventsTotal, c.lusrUpdated, c.csrInserted, c.csrSkipped, c.perfUpdated, c.citationsUpdated, c.comebackUpdated,
	)
	pct100 := 100
	matchesDone := c.total
	o.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Status = domain.JobStatusSucceeded
		j.ProgressPct = &pct100
		j.CurrentStep = &done
		j.MatchesDone = &matchesDone
	})
}

// runCitationsComeback : Phases 1.5 (citations) + 1.6 (comeback badges). Indépendantes
// du `missing` de la détection ; tournent avant l'early-return pour couvrir les joueurs
// à jour en données mais avec citations/dominance_flag manquants.
func (o *BackfillOrchestrator) runCitationsComeback(jobID string, c *backfillCounts) {
	if o.scope.Citations && !o.dryRun {
		o.setStep(jobID, "Backfill citations")
		n, citErr := o.engine.RunBackfillCitations(context.Background(), o.scope.ForceCitations)
		if citErr != nil {
			o.warn(jobID, "WARN citations: %v", citErr)
		}
		c.citationsUpdated = n
	}
	if o.scope.ComebackBadges && !o.dryRun {
		o.setStep(jobID, "Backfill comeback badges (dominance_flag)")
		n, cbErr := o.engine.RunBackfillComebackBadges(context.Background(), o.scope.ForceComebackBadges)
		if cbErr != nil {
			o.warn(jobID, "WARN comeback badges: %v", cbErr)
		}
		c.comebackUpdated = n
	}
}

// runWeaponsEngagement : Phases 2 (weapon kills, tokens requis), 2.5 (scores
// d'engagement, local) et 2.6 (coefficients seuls si scores non demandés).
func (o *BackfillOrchestrator) runWeaponsEngagement(jobID string, missing []string, c *backfillCounts) {
	if o.scope.Weapons {
		if o.tokens == nil {
			o.warn(jobID, "WARN: weapon kills ignorés — tokens Halo absents")
		} else {
			o.setStep(jobID, "Backfill weapon kills")
			inserted, _, wkErr := o.engine.BackfillWeaponKillsForMatches(context.Background(), missing)
			if wkErr != nil {
				o.warn(jobID, "WARN weapon kills: %v", wkErr)
			}
			c.weaponsInserted = inserted
		}
	}
	if o.scope.EngagementScores {
		o.setStep(jobID, "Calcul scores d'engagement")
		n, esErr := o.engine.RunBackfillEngagementScores(context.Background(), o.scope.ForceEngagementScores)
		if esErr != nil {
			o.warn(jobID, "WARN engagement scores: %v", esErr)
		}
		c.engagementComputed = n
	}
	// Coefficients seuls : si EngagementScores actif, le recompute est déjà fait en
	// queue de RunBackfillEngagementScores — on skip pour éviter le double passage.
	if o.scope.EngagementCoefficients && !o.scope.EngagementScores {
		o.setStep(jobID, "Recalcul coefficients d'engagement")
		n, ecErr := o.engine.RunBackfillEngagementCoefficients(context.Background())
		if ecErr != nil {
			o.warn(jobID, "WARN engagement coefficients: %v", ecErr)
		}
		c.engagementCoefsUpdated = n
	}
}

// runEventsLusr : Phases 2.7 (highlight events, tokens requis) et 3 (LUSR v2 canonique).
func (o *BackfillOrchestrator) runEventsLusr(jobID string, missing []string, c *backfillCounts) {
	if o.scope.Events {
		if o.tokens == nil {
			o.warn(jobID, "WARN: highlight events ignorés — tokens Halo absents")
		} else {
			o.setStep(jobID, "Backfill highlight events")
			res, evErr := o.engine.BackfillEventsForMatches(context.Background(), missing, o.scope.ForceEvents, nil)
			if evErr != nil {
				o.warn(jobID, "WARN events: %v", evErr)
			}
			if res.ParseAnomaly > 0 {
				o.warn(jobID, "events parse_anomaly: %d match(s) avec chunk présent mais 0 events parsés (voir compteur expvar highlight_events_parse_anomaly_total)", res.ParseAnomaly)
			}
			c.eventsHealed = res.Healed
			c.eventsTotal = res.EventsInserted
		}
	}
	if o.scope.LUSR {
		o.setStep(jobID, "Backfill LUSR")
		n, lusrErr := o.engine.RecomputeLUSRCanonical(context.Background())
		if lusrErr != nil {
			o.warn(jobID, "WARN lusr: %v", lusrErr)
		}
		c.lusrUpdated = n
	}
}

// runCsrPerfPsa : Phases 3.5 (CSR re-fetch RankRecap, tokens requis), 4 (performance
// score relatif) et 4.5 (personal score awards, tokens requis).
func (o *BackfillOrchestrator) runCsrPerfPsa(jobID string, missing []string, c *backfillCounts) {
	if o.scope.CSR {
		if o.tokens == nil {
			o.warn(jobID, "WARN: CSR ignoré — tokens Halo absents (re-login requis)")
		} else {
			o.setStep(jobID, "Backfill CSR (re-fetch RankRecap)")
			csrRes, csrErr := o.engine.RunBackfillCSR(context.Background(), o.scope.ForceCSR)
			if csrErr != nil {
				o.warn(jobID, "WARN csr: %v", csrErr)
			}
			c.csrInserted = csrRes.Inserted
			c.csrSkipped = csrRes.SkippedNoRankRecap + csrRes.SkillErrors
			if csrRes.SkillErrors > 0 {
				o.warn(jobID, "csr: %d appel(s) skill API en erreur (continuing)", csrRes.SkillErrors)
			}
		}
	}
	if o.scope.PerformanceScores {
		o.setStep(jobID, "Backfill performance score")
		n, perfErr := o.engine.RunBackfillPerf(context.Background(), o.scope.ForcePerformanceScores)
		if perfErr != nil {
			o.warn(jobID, "WARN perf: %v", perfErr)
		}
		c.perfUpdated = n
	}
	if o.scope.PersonalScores {
		if o.tokens == nil {
			o.warn(jobID, "WARN: personal scores ignorés — tokens Halo absents")
		} else {
			o.setStep(jobID, "Backfill personal scores")
			m, n, psaErr := o.engine.BackfillPersonalScoreAwardsForMatches(context.Background(), missing)
			if psaErr != nil {
				o.warn(jobID, "WARN personal scores: %v", psaErr)
			}
			c.psaMatchesUpdated = m
			c.psaRowsInserted = n
		}
	}
}

// warnUnimplemented ajoute des avertissements pour les types détectés dont le backfill
// API n'est pas encore implémenté en Go (medals, skill, aliases).
func (o *BackfillOrchestrator) warnUnimplemented(jobID string) {
	var types []string
	if o.scope.Medals {
		types = append(types, "medals")
	}
	// `events` retiré ici (mai 2026) — implémenté via Phase 2.7.
	if o.scope.Skill {
		types = append(types, "skill")
	}
	// personal_scores désormais implémenté (mai 2026, cf. Phase 4.5).
	if o.scope.Aliases {
		types = append(types, "aliases")
	}
	if len(types) == 0 {
		return
	}
	o.warn(jobID, "Les types suivants sont détectés mais le backfill API n'est pas encore implémenté en Go : %v", types)
}
