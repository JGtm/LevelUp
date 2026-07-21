// Package api — registry_lusr_gaps.go : runners du panneau monitoring « Notes
// LUSR — trous & garde-fou » (garde-fou trous d'intérieur LUSR).
//
// LUSRGapsReport : lectures seules best-effort par joueur (miroir de
// ConvergenceReport) — un joueur inaccessible pose CheckError, jamais d'échec
// global. RecomputeLUSRGapsForPlayer : action admin = replay chronologique
// in-server (SyncEngine.RecomputeLUSRCanonical, prend les leases player+shared).
package wire

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	sync_pkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/skill"
)

// maxTopGapsPerPlayer borne l'échantillon de trous renvoyé par joueur (les plus
// anciens d'abord — ScanLUSRGaps les rend en ordre chrono ASC).
const maxTopGapsPerPlayer = 10

// LUSRGapsReport scanne les trous LUSR de tous les joueurs suivis d'un titre.
// Read-only. Best-effort par joueur (CheckError isolé). Agrège + trie par impact
// (trous d'intérieur décroissants).
func (r *ServiceRegistry) LUSRGapsReport(ctx context.Context, titleSlug string) (domain.AdminLUSRGaps, error) {
	// La chaîne LUSR est title-aware : injecter le slug dans le ctx pour ScanLUSRGaps.
	ctx = ctxkeys.WithTitleSlug(ctx, titleSlug)
	resp := domain.AdminLUSRGaps{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Players:     []domain.LUSRGapPlayer{},
		Guardrail:   r.lusrGuardrailHealth(),
	}
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return resp, err
	}
	for _, p := range players {
		resp.Players = append(resp.Players, r.scanPlayerLUSRGaps(ctx, titleSlug, p))
	}
	for i := range resp.Players {
		pl := &resp.Players[i]
		resp.EligibleTotal += pl.Eligible
		resp.RatedTotal += pl.Rated
		resp.InteriorGapsTotal += pl.InteriorGaps
		resp.PendingRecentTotal += pl.PendingRecent
	}
	if resp.EligibleTotal > 0 {
		resp.CoveragePercent = float64(resp.RatedTotal) / float64(resp.EligibleTotal) * 100
	}
	// Joueurs les plus impactés d'abord (trous d'intérieur desc, puis pending desc).
	sort.SliceStable(resp.Players, func(a, b int) bool {
		if resp.Players[a].InteriorGaps != resp.Players[b].InteriorGaps {
			return resp.Players[a].InteriorGaps > resp.Players[b].InteriorGaps
		}
		return resp.Players[a].PendingRecent > resp.Players[b].PendingRecent
	})
	monitoringLog.DebugContext(ctx, "admin_monitoring: rapport trous LUSR calculé",
		"title", titleSlug, "players", len(resp.Players),
		"interior_gaps", resp.InteriorGapsTotal, "pending_recent", resp.PendingRecentTotal,
		"eligible", resp.EligibleTotal, "rated", resp.RatedTotal)
	return resp, nil
}

// scanPlayerLUSRGaps scanne un joueur (best-effort → CheckError si DB inaccessible).
func (r *ServiceRegistry) scanPlayerLUSRGaps(ctx context.Context, titleSlug string, p domain.PlayerSummary) domain.LUSRGapPlayer {
	out := domain.LUSRGapPlayer{PlayerSlug: p.PlayerSlug, Gamertag: p.Gamertag, XUID: p.XUID, TopGaps: []domain.LUSRGapItem{}}
	playerSQL, sharedSQL, release, errMsg := r.resolveMonitoringDBs(ctx, titleSlug, p.Gamertag, p.XUID)
	if errMsg != "" {
		out.CheckError = errMsg
		return out
	}
	defer release()
	rep, err := skill.ScanLUSRGaps(ctx, playerSQL, sharedSQL, p.XUID)
	if err != nil {
		// Log AVANT la dégradation best-effort (CLAUDE.md n°3) — sinon un scan qui
		// échoue en boucle serait invisible (seul CheckError du DTO le porterait).
		monitoringLog.WarnContext(ctx, "admin_monitoring: scan trous LUSR échoué",
			"gamertag", p.Gamertag, "xuid", p.XUID, "title", titleSlug, "err", err)
		out.CheckError = err.Error()
		return out
	}
	out.Eligible = rep.TotalEligible
	out.Rated = rep.TotalRated
	out.InteriorGaps = rep.TotalInteriorGaps
	out.PendingRecent = rep.TotalPendingRecent
	for _, g := range rep.Groups {
		for _, gap := range g.InteriorGaps {
			if len(out.TopGaps) >= maxTopGapsPerPlayer {
				break
			}
			out.TopGaps = append(out.TopGaps, domain.LUSRGapItem{
				MatchID:   gap.MatchID,
				Group:     gap.Group,
				Playlist:  gap.PairName,
				StartTime: gap.StartTime.UTC().Format(time.RFC3339),
			})
		}
	}
	return out
}

// lusrGuardrailHealth ré-expose les compteurs LUSR v2 (gauge + cumuls held/owner)
// + l'horodatage du dernier cycle data_health (scan des trous).
func (r *ServiceRegistry) lusrGuardrailHealth() domain.LUSRGuardrailHealth {
	h := domain.LUSRGuardrailHealth{
		InteriorGapsGauge: skill.LUSRInteriorGapsGaugeValue(),
		HeldWatermark:     skill.LUSRCanonicalWriteHeldWatermarkValue(),
		OwnerMissing:      skill.LUSRCanonicalOwnerMissingValue(),
	}
	if _, at := r.lastDataHealth(); !at.IsZero() {
		h.LastAuditAt = at.UTC().Format(time.RFC3339)
	}
	return h
}

// RecomputeLUSRGapsForPlayer déclenche un replay LUSR chronologique complet pour
// un joueur (reset watermark + rejeu en ordre → comble les trous d'intérieur).
// In-server : SyncEngine.RecomputeLUSRCanonical prend les leases player+shared et
// coordonne le shared writer via le Provider (B-swap). playerRef = slug OU gamertag.
func (r *ServiceRegistry) RecomputeLUSRGapsForPlayer(ctx context.Context, titleSlug, playerRef string) (domain.AdminLUSRRecomputeResponse, error) {
	var empty domain.AdminLUSRRecomputeResponse
	p, ok := r.resolvePlayerRef(titleSlug, playerRef)
	if !ok {
		return empty, fmt.Errorf("joueur introuvable pour ce titre : %q", playerRef)
	}
	engine := sync_pkg.NewSyncEngineForTitle(r.cfg.RepoRoot, titleSlug, p.Gamertag, p.XUID, nil, r.provider)
	if r.cfg.SharedProvider != nil {
		engine = engine.WithSharedProvider(r.cfg.SharedProvider)
	}
	monitoringLog.InfoContext(ctx, "admin_actions: replay LUSR démarré",
		"gamertag", p.Gamertag, "xuid", p.XUID, "title", titleSlug)
	updated, err := engine.RecomputeLUSRCanonical(ctx)
	if err != nil {
		return empty, fmt.Errorf("replay LUSR échoué: %w", err)
	}
	observability.IncCounter("admin_action_lusr_recompute_total")
	monitoringLog.InfoContext(ctx, "admin_actions: replay LUSR terminé",
		"gamertag", p.Gamertag, "xuid", p.XUID, "updated", updated)
	return domain.AdminLUSRRecomputeResponse{
		Gamertag: p.Gamertag, XUID: p.XUID, Updated: updated, OK: true,
	}, nil
}

// resolvePlayerRef résout un slug OU gamertag (insensible à la casse) en
// PlayerSummary pour le titre donné.
func (r *ServiceRegistry) resolvePlayerRef(titleSlug, ref string) (domain.PlayerSummary, bool) {
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return domain.PlayerSummary{}, false
	}
	if ref == "" {
		return domain.PlayerSummary{}, false
	}
	for _, p := range players {
		if strings.EqualFold(p.PlayerSlug, ref) || strings.EqualFold(p.Gamertag, ref) {
			return p, true
		}
	}
	return domain.PlayerSummary{}, false
}
