// Package handlers — backfill.go : lancement du pipeline de backfill (Sprint 51-B3).
//
// POST /backfill/start → crée un job "backfill" et retourne 202 immédiatement.
// Règles :
//   - 400 si player_slug absent ou scope entièrement vide
//   - 401 si tokens Halo absents (requis pour weapon kills)
//   - 404 si le joueur est introuvable dans db_profiles.json
//   - 409 si un job backfill actif existe déjà pour ce player_slug
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
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

// StartBackfill déclenche le pipeline backfill pour un joueur.
// POST /backfill/start → 202 AsyncJobStatus.
func (h *BackfillHandler) StartBackfill(w http.ResponseWriter, r *http.Request) {
	var req domain.BackfillStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
		return
	}

	if req.PlayerSlug == "" || len(req.PlayerSlug) > 50 {
		writeError(w, http.StatusBadRequest, "invalid_player_slug", "player_slug vide ou trop long.")
		return
	}

	// Chercher le joueur dans db_profiles.json.
	players, err := h.cfg.LoadPlayers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profiles_load_error",
			"Impossible de charger db_profiles.json.")
		return
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
		writeError(w, http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", req.PlayerSlug))
		return
	}

	// 409 si un job backfill est déjà actif pour ce joueur.
	if active := h.jobStore.FindActiveJob(domain.JobTypeBackfill, req.PlayerSlug); active != nil {
		writeError(w, http.StatusConflict, "backfill_already_active",
			"Un backfill est déjà en cours pour ce joueur.")
		return
	}

	// Les weapons backfill nécessitent les tokens Halo.
	sess := middleware.GetSession(r.Context())
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

		if req.DryRun || total == 0 {
			done := fmt.Sprintf("Terminé (dry_run=%v, %d match(s) détectés)", req.DryRun, total)
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

		// ── Phase 3 : types non encore implémentés → avertissement ──────
		h.warnUnimplemented(job.JobID, scope)

		done := fmt.Sprintf(
			"Backfill terminé — matchs: %d, weapon kills insérés: %d",
			total, weaponsInserted,
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

	writeJSON(w, http.StatusAccepted, &jobSnapshot)
}

// buildSyncScope construit un SyncScope depuis le payload de requête.
func buildSyncScope(req domain.BackfillStartRequest) *go_sync.SyncScope {
	scope := &go_sync.SyncScope{
		DetectionMode: "or",
		MaxMatches:    req.MaxMatches,
	}

	if req.AllData || (!req.Medals && !req.Events && !req.Skill &&
		!req.PersonalScores && !req.PerformanceScores &&
		!req.Aliases && !req.Weapons && !req.LUSR) {
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
	} else {
		scope.Medals = req.Medals
		scope.Events = req.Events
		scope.Skill = req.Skill
		scope.PersonalScores = req.PersonalScores
		scope.PerformanceScores = req.PerformanceScores
		scope.Aliases = req.Aliases
		scope.Weapons = req.Weapons
		scope.LUSR = req.LUSR
	}

	if req.ForceRescan {
		scope.ForceMedals = req.Medals || req.AllData
		scope.ForceSkill = req.Skill || req.AllData
		scope.ForceWeapons = req.Weapons || req.AllData
		scope.ForcePerformanceScores = req.PerformanceScores || req.AllData
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
	if scope.Events {
		types = append(types, "events")
	}
	if scope.Skill {
		types = append(types, "skill")
	}
	if scope.PersonalScores {
		types = append(types, "personal_scores")
	}
	if scope.PerformanceScores {
		types = append(types, "performance_scores")
	}
	if scope.Aliases {
		types = append(types, "aliases")
	}
	if scope.LUSR {
		types = append(types, "lusr")
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
