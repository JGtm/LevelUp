// Package handlers — backfill.go : lancement du pipeline de backfill (Sprint 51-B3).
//
// POST /backfill/start → crée un job "backfill" et retourne 202 immédiatement.
// Règles :
//   - 400 si player_slug absent ou scope entièrement vide
//   - 401 si tokens Halo absents (requis pour weapon kills)
//   - 404 si le joueur est introuvable dans db_profiles.json
//   - 409 si un job backfill actif existe déjà pour ce player_slug
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// (point de montage /backfill/start, middleware RequireAuth/RequireAdmin hérités)
// et enregistre le POST via huma.Post. Le corps est lu via RawBody (pas de Body
// typé) pour reproduire EXACTEMENT le contrat de décodage d'origine : un JSON
// invalide (corps absent inclus) renvoie 400 {invalid_body} et non le 422 de
// validation Huma. Logique métier inchangée, seul le wrapping HTTP change.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/jobs"
	go_sync "levelup/go-api/internal/sync"
)

// BackfillHandler gère le pipeline de backfill des données manquantes.
type BackfillHandler struct {
	cfg      *config.AppConfig
	jobStore *jobs.Store
}

// NewBackfillHandler crée un BackfillHandler.
func NewBackfillHandler(cfg *config.AppConfig, jobStore *jobs.Store) *BackfillHandler {
	return &BackfillHandler{cfg: cfg, jobStore: jobStore}
}

// Mount enregistre la route via Huma sur le routeur chi (point de montage
// /backfill/start, middleware RequireAuth/RequireAdmin hérités du groupe).
func (h *BackfillHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/backfill/start", h.handleStartBackfill)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// backfillStartInput : corps brut décodé maison. RawBody (pas Body typé) →
// préserve le contrat 400 {invalid_body} sur JSON invalide ou corps absent (un
// Body typé renverrait le 422 de validation Huma).
type backfillStartInput struct {
	RawBody []byte
}

// backfillStartOutput : 202 Accepted, corps = snapshot du job créé.
type backfillStartOutput struct {
	Status int
	Body   *domain.AsyncJobStatus
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleStartBackfill déclenche le pipeline backfill pour un joueur.
// POST /backfill/start → 202 AsyncJobStatus.
//
//nolint:funlen // pipeline d'orchestration : validation, lookup, conflit, lancement goroutine
func (h *BackfillHandler) handleStartBackfill(ctx context.Context, in *backfillStartInput) (*backfillStartOutput, error) {
	var req domain.BackfillStartRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}

	if req.PlayerSlug == "" || len(req.PlayerSlug) > 50 {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug vide ou trop long.")
	}

	// Chercher le joueur dans db_profiles.json.
	players, err := h.cfg.LoadPlayers()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "profiles_load_error",
			"Impossible de charger db_profiles.json.")
	}
	var gamertag, xuid string
	for _, p := range players {
		if p.PlayerSlug == req.PlayerSlug {
			gamertag = p.Gamertag
			xuid = p.XUID
			break
		}
	}
	if gamertag == "" {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", req.PlayerSlug))
	}

	// 409 si un job backfill est déjà actif pour ce joueur.
	if active := h.jobStore.FindActiveJob(domain.JobTypeBackfill, req.PlayerSlug); active != nil {
		return nil, humacore.NewError(http.StatusConflict, "backfill_already_active",
			"Un backfill est déjà en cours pour ce joueur.")
	}

	// Sprint B1 commit 17 : event_id pour tracer le backfill HTTP-triggered.
	// Log immédiat sans muter le contexte requête (data race avec chi middleware).
	_, evID := logging.WithEvent(ctx, "http.backfill:"+req.PlayerSlug)
	slog.InfoContext(ctx, "backfill_handler: StartBackfill démarré",
		"player_slug", req.PlayerSlug, "gamertag", gamertag, "event", evID)

	// Les weapons backfill nécessitent les tokens Halo.
	sess := middleware.GetSession(ctx)
	var tokens *domain.HaloTokens
	if sess != nil {
		tokens = sess.HaloTokens
	}

	// Construire le SyncScope depuis la requête.
	scope := buildSyncScope(req)

	job := h.jobStore.Create(domain.JobTypeBackfill, req.PlayerSlug)
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	go func() {
		step := "Détection des données manquantes"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		engine := go_sync.NewSyncEngine(h.cfg.RepoRoot, gamertag, xuid, tokens, nil)
		// Sprint B1 commit 11b : aligner le backfill HTTP-triggered sur auto_sync.
		// Sans Provider wired, RunBackfill* utiliseraient OpenSharedDB direct
		// (mode legacy) en parallèle des syncs auto qui passent par Provider →
		// conflit "different configuration" pour les readers HTTP. Le fix
		// double-dblease dans acquireSharedWriter (commit 10a + 11b) rend ce
		// wire sûr — Provider gère le lease en interne.
		if h.cfg.SharedProvider != nil {
			engine = engine.WithSharedProvider(h.cfg.SharedProvider)
		}

		// ── Phase 1 : détection ──────────────────────────────────────────
		missing, err := engine.RunBackfill(context.Background(), scope)
		if err != nil {
			errMsg := err.Error()
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.Status = domain.JobStatusFailed
				j.Error = &domain.JobErrorDetail{Code: "detection_error", Message: errMsg}
			})
			return
		}

		total := len(missing)
		pct := 50
		detStep := fmt.Sprintf("Détection terminée : %d match(s) à traiter", total)
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.MatchesTotal = &total
			j.ProgressPct = &pct
			j.CurrentStep = &detStep
		})

		// ── Phase 1.5 : Citations ────────────────────────────────────────────
		// Indépendant du `missing` de la détection : requête LEFT JOIN IS NULL
		// sur match_citations. Même pattern dblease que RunBackfillComebackBadges :
		// sérialise avec le sync engine, prévient le bug ART DuckDB.
		citationsUpdated := 0
		if scope.Citations && !req.DryRun {
			citStep := "Backfill citations"
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &citStep
			})
			n, citErr := engine.RunBackfillCitations(context.Background(), scope.ForceCitations)
			if citErr != nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						fmt.Sprintf("WARN citations: %v", citErr))
				})
			}
			citationsUpdated = n
		}

		// ── Phase 1.6 : Comeback badges (dominance_flag) ────────────────────
		// Indépendant du `missing` de la détection : utilise sa propre
		// requête SQL (matchs sans dominance_flag). Doit tourner avant
		// l'early-return total==0 pour couvrir les joueurs à jour en données
		// mais avec des flags dominance manquants.
		comebackUpdated := 0
		if scope.ComebackBadges && !req.DryRun {
			cbStep := "Backfill comeback badges (dominance_flag)"
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &cbStep
			})
			n, cbErr := engine.RunBackfillComebackBadges(context.Background(), scope.ForceComebackBadges)
			if cbErr != nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						fmt.Sprintf("WARN comeback badges: %v", cbErr))
				})
			}
			comebackUpdated = n
		}

		if req.DryRun || total == 0 {
			done := fmt.Sprintf("Terminé (dry_run=%v, %d match(s) détectés, citations: %d, comeback: %d)", req.DryRun, total, citationsUpdated, comebackUpdated)
			pct100 := 100
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.Status = domain.JobStatusSucceeded
				j.ProgressPct = &pct100
				j.CurrentStep = &done
			})
			return
		}

		// ── Phase 2 : weapon kills (seul type avec API impl. en Go) ─────
		weaponsInserted := 0
		if scope.Weapons {
			if tokens == nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						"WARN: weapon kills ignorés — tokens Halo absents")
				})
			} else {
				wkStep := "Backfill weapon kills"
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.CurrentStep = &wkStep
				})
				inserted, _, wkErr := engine.BackfillWeaponKillsForMatches(
					context.Background(), missing,
				)
				if wkErr != nil {
					h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
						j.Warnings = append(j.Warnings,
							fmt.Sprintf("WARN weapon kills: %v", wkErr))
					})
				}
				weaponsInserted = inserted
			}
		}

		// ── Phase 2.5 : engagement scores (Phase 6 plan engagement) ─────
		// Calcul purement local (pas d'API), dispo des que les events sont
		// synces. Skip silencieux si migration non appliquee.
		engagementComputed := 0
		if scope.EngagementScores {
			esStep := "Calcul scores d'engagement"
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &esStep
			})
			n, esErr := engine.RunBackfillEngagementScores(context.Background(), scope.ForceEngagementScores)
			if esErr != nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						fmt.Sprintf("WARN engagement scores: %v", esErr))
				})
			}
			engagementComputed = n
		}

		// ── Phase 2.6 : engagement coefficients only (Phase recompute coefs) ─
		// Recompute juste la mediane glissante des paces deja persistees.
		// Tres rapide, pas de re-scan. Si EngagementScores=true ci-dessus,
		// le recompute est deja fait en queue — on skip pour eviter le double
		// passage et son log. Sinon (option seule), on lance le recompute.
		engagementCoefsUpdated := 0
		if scope.EngagementCoefficients && !scope.EngagementScores {
			ecStep := "Recalcul coefficients d'engagement"
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &ecStep
			})
			n, ecErr := engine.RunBackfillEngagementCoefficients(context.Background())
			if ecErr != nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						fmt.Sprintf("WARN engagement coefficients: %v", ecErr))
				})
			}
			engagementCoefsUpdated = n
		}
		_ = engagementCoefsUpdated // exposable dans le résumé final si besoin

		// ── Phase 2.7 : highlight events (Phase 2 plan PLAN_HIGHLIGHT_EVENTS_BACKFILL) ─
		// Replay du parsing highlight events pour les matchs où :
		//   - events_loaded=FALSE (= jamais traités, présents dans `missing`)
		//   - OU si `force_events=true` : events_loaded=TRUE mais cassé
		//     silencieusement (par l'ancien parser byte-aligné ou le bug
		//     InsertKVP). Détection via FindBrokenHighlightEventMatches.
		//
		// Avant mai 2026, ce type était listé dans warnUnimplemented (no-op).
		eventsHealed := 0
		eventsTotal := 0
		if scope.Events {
			if tokens == nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						"WARN: highlight events ignorés — tokens Halo absents")
				})
			} else {
				evStep := "Backfill highlight events"
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.CurrentStep = &evStep
				})
				res, evErr := engine.BackfillEventsForMatches(
					context.Background(), missing, scope.ForceEvents, nil,
				)
				if evErr != nil {
					h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
						j.Warnings = append(j.Warnings,
							fmt.Sprintf("WARN events: %v", evErr))
					})
				}
				if res.ParseAnomaly > 0 {
					h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
						j.Warnings = append(j.Warnings,
							fmt.Sprintf("events parse_anomaly: %d match(s) avec chunk présent mais 0 events parsés (voir compteur expvar highlight_events_parse_anomaly_total)",
								res.ParseAnomaly))
					})
				}
				eventsHealed = res.Healed
				eventsTotal = res.EventsInserted
			}
		}

		// ── Phase 3 : LUSR v2 canonique (TrueSkill2, ADR 0024) ───────────
		// CR C3 : le backfill emprunte le chemin v2 (RecomputeLUSRCanonical),
		// jamais le legacy v1. scope.ForceLUSR n'est plus consommé ici : le
		// replay v2 est toujours complet.
		lusrUpdated := 0
		if scope.LUSR {
			lusrStep := "Backfill LUSR"
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &lusrStep
			})
			n, lusrErr := engine.RecomputeLUSRCanonical(context.Background())
			if lusrErr != nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						fmt.Sprintf("WARN lusr: %v", lusrErr))
				})
			}
			lusrUpdated = n
		}

		// ── Phase 3.5 : CSR (re-fetch RankRecap depuis l'API skill) ──────
		// Idempotent par défaut (skip les matchs ranked qui ont déjà une row
		// CSR), force-csr re-fetche tous les matchs ranked.
		csrInserted := 0
		csrSkipped := 0
		if scope.CSR {
			if tokens == nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						"WARN: CSR ignoré — tokens Halo absents (re-login requis)")
				})
			} else {
				csrStep := "Backfill CSR (re-fetch RankRecap)"
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.CurrentStep = &csrStep
				})
				csrRes, csrErr := engine.RunBackfillCSR(context.Background(), scope.ForceCSR)
				if csrErr != nil {
					h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
						j.Warnings = append(j.Warnings,
							fmt.Sprintf("WARN csr: %v", csrErr))
					})
				}
				csrInserted = csrRes.Inserted
				csrSkipped = csrRes.SkippedNoRankRecap + csrRes.SkillErrors
				if csrRes.SkillErrors > 0 {
					h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
						j.Warnings = append(j.Warnings,
							fmt.Sprintf("csr: %d appel(s) skill API en erreur (continuing)", csrRes.SkillErrors))
					})
				}
			}
		}

		// ── Phase 4 : Performance score relatif v5 ───────────────────────
		perfUpdated := 0
		if scope.PerformanceScores {
			perfStep := "Backfill performance score"
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &perfStep
			})
			n, perfErr := engine.RunBackfillPerf(context.Background(), scope.ForcePerformanceScores)
			if perfErr != nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						fmt.Sprintf("WARN perf: %v", perfErr))
				})
			}
			perfUpdated = n
		}

		// ── Phase 4.5 : Personal score awards (PSA) — par joueur ─────────
		// Pipeline parallèle à weapons : fetch GetMatchStats + extract PSA +
		// insert dans player DB. Idempotent (DELETE + INSERT batch).
		psaMatchesUpdated := 0
		psaRowsInserted := 0
		if scope.PersonalScores {
			if tokens == nil {
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.Warnings = append(j.Warnings,
						"WARN: personal scores ignorés — tokens Halo absents")
				})
			} else {
				psaStep := "Backfill personal scores"
				h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
					j.CurrentStep = &psaStep
				})
				m, n, psaErr := engine.BackfillPersonalScoreAwardsForMatches(
					context.Background(), missing,
				)
				if psaErr != nil {
					h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
						j.Warnings = append(j.Warnings,
							fmt.Sprintf("WARN personal scores: %v", psaErr))
					})
				}
				psaMatchesUpdated = m
				psaRowsInserted = n
			}
		}

		// ── Phase 5 : types non encore implémentés → avertissement ──────
		h.warnUnimplemented(job.JobID, scope)

		done := fmt.Sprintf(
			"Backfill terminé — matchs: %d, weapon kills insérés: %d, psa: %d match(s)/%d rows, engagement: %d, events healed: %d (%d events insérés), lusr: %d, csr: %d (skipped: %d), perf: %d, citations: %d, comeback: %d",
			total, weaponsInserted, psaMatchesUpdated, psaRowsInserted, engagementComputed, eventsHealed, eventsTotal, lusrUpdated, csrInserted, csrSkipped, perfUpdated, citationsUpdated, comebackUpdated,
		)
		pct100 := 100
		matchesDone := total
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct100
			j.CurrentStep = &done
			j.MatchesDone = &matchesDone
		})
	}()

	return &backfillStartOutput{Status: http.StatusAccepted, Body: &jobSnapshot}, nil
}

// buildSyncScope construit un SyncScope depuis le payload de requête.
func buildSyncScope(req domain.BackfillStartRequest) *go_sync.SyncScope {
	scope := &go_sync.SyncScope{
		DetectionMode: "or",
		MaxMatches:    req.MaxMatches,
	}

	noExplicitScope := !req.Medals && !req.Events && !req.Skill &&
		!req.PersonalScores && !req.PerformanceScores &&
		!req.Aliases && !req.Weapons && !req.LUSR && !req.CSR && !req.EngagementScores &&
		!req.EngagementCoefficients && !req.ComebackBadges && !req.Citations
	if req.AllData || noExplicitScope {
		// Aucun scope explicite → activer tout
		scope.AllData = true
		scope.Medals = true
		scope.Events = true
		scope.Skill = true
		scope.PersonalScores = true
		scope.PerformanceScores = true
		scope.Aliases = true
		scope.Weapons = true
		scope.LUSR = true
		scope.CSR = true
		scope.EngagementScores = true
		scope.ComebackBadges = true
		scope.Citations = true
		// EngagementCoefficients implicite : les coefs sont recomputes en
		// queue de RunBackfillEngagementScores, pas besoin d'activer le flag.
	} else {
		scope.Medals = req.Medals
		scope.Events = req.Events
		scope.Skill = req.Skill
		scope.PersonalScores = req.PersonalScores
		scope.PerformanceScores = req.PerformanceScores
		scope.Aliases = req.Aliases
		scope.Weapons = req.Weapons
		scope.LUSR = req.LUSR
		scope.CSR = req.CSR
		scope.EngagementScores = req.EngagementScores
		scope.EngagementCoefficients = req.EngagementCoefficients
		scope.ComebackBadges = req.ComebackBadges
		scope.Citations = req.Citations
	}

	if req.ForceRescan {
		scope.ForceMedals = req.Medals || req.AllData
		scope.ForceEvents = req.Events || req.AllData
		scope.ForceSkill = req.Skill || req.AllData
		scope.ForceWeapons = req.Weapons || req.AllData
		scope.ForcePersonalScores = req.PersonalScores || req.AllData
		scope.ForcePerformanceScores = req.PerformanceScores || req.AllData
		scope.ForceAliases = req.Aliases || req.AllData
		scope.ForceLUSR = req.LUSR || req.AllData
		scope.ForceCSR = req.CSR || req.AllData
		scope.ForceEngagementScores = req.EngagementScores || req.AllData
		scope.ForceEngagementCoefficients = req.EngagementCoefficients
		scope.ForceComebackBadges = req.ComebackBadges || req.AllData
		scope.ForceCitations = req.Citations || req.AllData
	}

	scope.Resolve()
	return scope
}

// warnUnimplemented ajoute des avertissements pour les types dont le backfill
// API n'est pas encore implémenté en Go.
func (h *BackfillHandler) warnUnimplemented(jobID string, scope *go_sync.SyncScope) {
	var types []string
	if scope.Medals {
		types = append(types, "medals")
	}
	// `events` retiré ici (mai 2026) — désormais implémenté via Phase 2.7
	// ci-dessus (engine.BackfillEventsForMatches + ReplayHighlightEvents).
	if scope.Skill {
		types = append(types, "skill")
	}
	// personal_scores désormais implémenté (mai 2026, cf. Phase 4.5).
	if scope.Aliases {
		types = append(types, "aliases")
	}
	if len(types) == 0 {
		return
	}
	w := fmt.Sprintf(
		"Les types suivants sont détectés mais le backfill API n'est pas encore implémenté en Go : %v",
		types,
	)
	h.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Warnings = append(j.Warnings, w)
	})
}
