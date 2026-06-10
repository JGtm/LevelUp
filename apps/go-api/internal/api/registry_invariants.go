// Package api — registry_invariants.go : runner des invariants de données
// pour le dashboard admin « Intégrité des données ».
//
// Orchestration : énumère les joueurs suivis (db_profiles.json), résout leur
// PlayerDB via le pool (resolveByGT, pool-cached) et exécute
// invariants.CheckPlayer (lectures seules). Best-effort par joueur : une DB
// inaccessible produit un report avec CheckError, jamais un échec global.
package api

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/invariants"
)

// RunDataInvariants exécute tous les invariants déclarés pour chaque joueur
// suivi du titre. Phase 4 du plan .ai/PLAN_SYNC_INVARIANTS_GATE.md.
func (r *ServiceRegistry) RunDataInvariants(ctx context.Context, titleSlug string) (domain.AdminInvariantsResponse, error) {
	resp := domain.AdminInvariantsResponse{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Reports:     []domain.PlayerInvariantsReport{},
	}
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return resp, err
	}
	totalFail, totalWarn := 0, 0
	for _, p := range players {
		report := domain.PlayerInvariantsReport{
			PlayerSlug: p.PlayerSlug,
			Gamertag:   p.Gamertag,
			XUID:       p.XUID,
			Violations: []domain.InvariantViolation{},
		}
		r.fillPlayerInvariants(ctx, titleSlug, &report)
		totalFail += report.FailCount
		totalWarn += report.WarnCount
		resp.Reports = append(resp.Reports, report)
	}
	// Compteurs expvar (pattern RunDualRowSentinel) : volumétrie observable
	// sans ouvrir le dashboard. *_last = état du DERNIER run (pas un cumul).
	observability.AddInt("invariants_runs_total", 1)
	observability.SetInt("invariants_fail_last", int64(totalFail))
	observability.SetInt("invariants_warn_last", int64(totalWarn))
	if totalFail > 0 {
		slog.WarnContext(ctx, "admin_invariants: violations FAIL détectées",
			"title", titleSlug, "fail_total", totalFail, "warn_total", totalWarn)
	}
	return resp, nil
}

// fillPlayerInvariants vérifie un joueur (best-effort, CheckError si harnais KO).
func (r *ServiceRegistry) fillPlayerInvariants(ctx context.Context, titleSlug string, report *domain.PlayerInvariantsReport) {
	if r.resolveByGT == nil || report.Gamertag == "" || report.XUID == "" {
		report.CheckError = "resolver ou identité joueur indisponible"
		return
	}
	tpdb, err := r.resolveByGT(ctx, titleSlug, report.Gamertag)
	if err != nil || tpdb == nil || tpdb.Player == nil {
		report.CheckError = "player DB inaccessible"
		slog.WarnContext(ctx, "admin_invariants: resolve player failed",
			"gamertag", report.Gamertag, "err", err)
		return
	}
	sharedDB, release, err := tpdb.SharedReadDB().Get(ctx)
	if err != nil {
		report.CheckError = "shared DB inaccessible"
		slog.WarnContext(ctx, "admin_invariants: shared reader failed",
			"gamertag", report.Gamertag, "err", err)
		return
	}
	defer release()

	rep, err := invariants.CheckPlayer(ctx, tpdb.Player.SQLDb(), sharedDB, report.XUID)
	if err != nil {
		report.CheckError = err.Error()
		slog.WarnContext(ctx, "admin_invariants: check failed",
			"gamertag", report.Gamertag, "err", err)
		return
	}
	for _, v := range rep.Violations {
		report.Violations = append(report.Violations, domain.InvariantViolation{
			Key:         v.Key,
			Severity:    string(v.Severity),
			Count:       v.Count,
			Sample:      v.Sample,
			Description: v.Description,
		})
		switch v.Severity {
		case invariants.SeverityFail:
			report.FailCount++
		case invariants.SeverityWarn:
			report.WarnCount++
		}
	}
}
