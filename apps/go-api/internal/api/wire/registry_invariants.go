// Package api — registry_invariants.go : runner des invariants de données
// pour le dashboard admin « Intégrité des données ».
//
// Orchestration : énumère les joueurs suivis (db_profiles.json), résout leur
// PlayerDB via le pool (resolveByGT, pool-cached) et exécute
// invariants.CheckPlayer (lectures seules — NB : la résolution pool peut
// créer/migrer une player DB déclarée mais absente, même effet qu'une visite
// de page joueur). Les invariants GLOBAUX (CheckShared) sont exécutés UNE
// seule fois par run. Best-effort par joueur : une DB inaccessible produit un
// report avec CheckError, jamais un échec global.
//
// Logs : module explicite « invariants » → logs/invariants.log (le package
// api routerait sinon vers http.log, mauvais fichier pour un signal
// d'intégrité de données — pattern ModuleLeaderboard).
package wire

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/invariants"
)

// invariantsLog : logger taggé module=invariants (fichier logs/invariants.log).
var invariantsLog = slog.With("module", "invariants")

// RunDataInvariants exécute tous les invariants déclarés : les globaux une
// fois, puis les invariants par joueur pour chaque joueur suivi du titre.
// Phase 4 du plan .ai/PLAN_SYNC_INVARIANTS_GATE.md.
func (r *ServiceRegistry) RunDataInvariants(ctx context.Context, titleSlug string) (domain.AdminInvariantsResponse, error) {
	resp := domain.AdminInvariantsResponse{
		TitleSlug:        titleSlug,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		Reports:          []domain.PlayerInvariantsReport{},
		SharedViolations: []domain.InvariantViolation{},
	}
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return resp, err
	}

	totalFail, totalWarn := 0, 0
	sharedDone := false
	for _, p := range players {
		report := domain.PlayerInvariantsReport{
			PlayerSlug: p.PlayerSlug,
			Gamertag:   p.Gamertag,
			XUID:       p.XUID,
			Violations: []domain.InvariantViolation{},
		}
		playerSQL, sharedSQL, release := r.resolveInvariantDBs(ctx, titleSlug, &report)
		if playerSQL != nil {
			// Invariants globaux : une seule fois, portés par la première
			// player DB résolvable (ATTACH global nécessaire au check alias).
			if !sharedDone {
				r.fillSharedInvariants(ctx, &resp, playerSQL, sharedSQL)
				sharedDone = true
			}
			r.fillPlayerInvariants(ctx, &report, playerSQL, sharedSQL)
			release()
		}
		totalFail += report.FailCount
		totalWarn += report.WarnCount
		resp.Reports = append(resp.Reports, report)
	}
	if !sharedDone {
		resp.SharedCheckError = "aucune player DB résolvable pour vérifier les invariants globaux"
		invariantsLog.WarnContext(ctx, "admin_invariants: invariants globaux non vérifiés", "title", titleSlug)
	}
	totalFail += resp.SharedFailCount
	totalWarn += resp.SharedWarnCount

	// Compteurs expvar (pattern RunDualRowSentinel) : volumétrie observable
	// sans ouvrir le dashboard. *_last = état du DERNIER run (pas un cumul).
	observability.AddInt("invariants_runs_total", 1)
	observability.SetInt("invariants_fail_last", int64(totalFail))
	observability.SetInt("invariants_warn_last", int64(totalWarn))
	if totalFail > 0 {
		invariantsLog.WarnContext(ctx, "admin_invariants: violations FAIL détectées",
			"title", titleSlug, "fail_total", totalFail, "warn_total", totalWarn)
	} else {
		invariantsLog.InfoContext(ctx, "admin_invariants: run terminé",
			"title", titleSlug, "players", len(players), "warn_total", totalWarn)
	}
	return resp, nil
}

// resolveInvariantDBs résout les handles DB d'un joueur (best-effort). En cas
// d'échec, report.CheckError est posé, le WARN loggé, et (nil, nil, nil) est
// retourné. release() doit être appelé après usage du sharedSQL.
func (r *ServiceRegistry) resolveInvariantDBs(
	ctx context.Context, titleSlug string, report *domain.PlayerInvariantsReport,
) (playerSQL, sharedSQL *sql.DB, release func()) {
	if r.resolveByGT == nil || report.Gamertag == "" || report.XUID == "" {
		report.CheckError = "resolver ou identité joueur indisponible"
		invariantsLog.WarnContext(ctx, "admin_invariants: resolver ou identité indisponible",
			"gamertag", report.Gamertag, "xuid", report.XUID)
		return nil, nil, nil
	}
	tpdb, err := r.resolveByGT(ctx, titleSlug, report.Gamertag)
	if err != nil || tpdb == nil || tpdb.Player == nil {
		report.CheckError = "player DB inaccessible"
		invariantsLog.WarnContext(ctx, "admin_invariants: resolve player failed",
			"gamertag", report.Gamertag, "err", err)
		return nil, nil, nil
	}
	shared, rel, err := tpdb.SharedReadDB().Get(ctx)
	if err != nil {
		report.CheckError = "shared DB inaccessible"
		invariantsLog.WarnContext(ctx, "admin_invariants: shared reader failed",
			"gamertag", report.Gamertag, "err", err)
		return nil, nil, nil
	}
	return tpdb.Player.SQLDb(), shared, rel
}

// fillPlayerInvariants exécute les invariants par joueur (best-effort).
func (r *ServiceRegistry) fillPlayerInvariants(
	ctx context.Context, report *domain.PlayerInvariantsReport, playerSQL, sharedSQL *sql.DB,
) {
	rep, err := invariants.CheckPlayer(ctx, playerSQL, sharedSQL, report.XUID)
	if err != nil {
		report.CheckError = err.Error()
		invariantsLog.WarnContext(ctx, "admin_invariants: check player failed",
			"gamertag", report.Gamertag, "err", err)
		return
	}
	for _, v := range rep.Violations {
		report.Violations = append(report.Violations, toDomainViolation(v))
		switch v.Severity {
		case invariants.SeverityFail:
			report.FailCount++
		case invariants.SeverityWarn:
			report.WarnCount++
		}
	}
}

// fillSharedInvariants exécute les invariants globaux (une fois par run).
func (r *ServiceRegistry) fillSharedInvariants(
	ctx context.Context, resp *domain.AdminInvariantsResponse, playerSQL, sharedSQL *sql.DB,
) {
	rep, err := invariants.CheckShared(ctx, playerSQL, sharedSQL)
	if err != nil {
		resp.SharedCheckError = err.Error()
		invariantsLog.WarnContext(ctx, "admin_invariants: check shared failed", "err", err)
		return
	}
	for _, v := range rep.Violations {
		resp.SharedViolations = append(resp.SharedViolations, toDomainViolation(v))
		switch v.Severity {
		case invariants.SeverityFail:
			resp.SharedFailCount++
		case invariants.SeverityWarn:
			resp.SharedWarnCount++
		}
	}
}

func toDomainViolation(v invariants.Violation) domain.InvariantViolation {
	return domain.InvariantViolation{
		Key:         v.Key,
		Severity:    string(v.Severity),
		Count:       v.Count,
		Sample:      v.Sample,
		Description: v.Description,
	}
}
