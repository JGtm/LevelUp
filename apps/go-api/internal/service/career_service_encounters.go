// Package service — career_service_encounters.go : top matches (best/worst),
// encounters, top encounters globaux et rivals pour la page Carrière.
// Découpé de career_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// GetTopMatches retourne les 10 meilleurs et 10 moins bons matchs.
func (s *CareerService) GetTopMatches(ctx context.Context) (domain.CareerTopMatchesResponse, error) {
	rows, err := s.loadTopMatchRows(ctx)
	if err != nil {
		return domain.CareerTopMatchesResponse{}, fmt.Errorf("CareerService.GetTopMatches: %w", err)
	}

	// Q9 retourne jusqu'à 20 matchs classés DESC par performance_score.
	// Les 10 premiers = meilleurs, les 10 derniers = moins bons.
	bestRows, worstRows := splitTopRows(rows)
	best := convertTopMatches(bestRows)
	// Inverser l'affichage des pires (les moins bons en premier)
	reverseTopMatches(worstRows)
	worst := convertTopMatches(worstRows)

	return domain.CareerTopMatchesResponse{
		BestMatches:  best,
		WorstMatches: worst,
	}, nil
}

// loadTopMatchRows centralise la résolution repo/adapter pour les top matches
// (HIGH-C). Adapter-first via LoadTopMatches + reconstitution byte-identique
// (OutcomeCode → Outcome BRUT) ; fallback repo.GetTopMatches sur capability absente.
func (s *CareerService) loadTopMatchRows(ctx context.Context) ([]domain.TopMatchRawRow, error) {
	if s.dataAdapter != nil {
		canonicalRows, err := s.dataAdapter.LoadTopMatches(ctx, "")
		if err == nil {
			if len(canonicalRows) == 0 {
				return nil, nil
			}
			out := make([]domain.TopMatchRawRow, 0, len(canonicalRows))
			for _, c := range canonicalRows {
				out = append(out, topMatchRowFromCanonical(c))
			}
			return out, nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
	}
	return s.repo.GetTopMatches(ctx)
}

// topMatchRowFromCanonical projette canonical.CareerTopMatch → domain.TopMatchRawRow
// (copie profonde des pointeurs). Outcome = OutcomeCode BRUT (le split WIN/LOSS aval
// compare à domain.OutcomeWin ; jamais via une string canonique).
func topMatchRowFromCanonical(c canonical.CareerTopMatch) domain.TopMatchRawRow {
	r := domain.TopMatchRawRow{
		MatchID:          c.MatchID,
		PerformanceScore: c.PerformanceScore,
		Outcome:          c.OutcomeCode,
		Kills:            c.Kills,
		Deaths:           c.Deaths,
		DominanceFlag:    c.DominanceFlag,
	}
	if c.StartTime != nil {
		v := *c.StartTime
		r.StartTime = &v
	}
	if c.MapName != nil {
		v := *c.MapName
		r.MapName = &v
	}
	if c.PairName != nil {
		v := *c.PairName
		r.PairName = &v
	}
	if c.PlaylistName != nil {
		v := *c.PlaylistName
		r.PlaylistName = &v
	}
	if c.KDA != nil {
		v := *c.KDA
		r.KDA = &v
	}
	if c.TeamMMR != nil {
		v := *c.TeamMMR
		r.TeamMMR = &v
	}
	if c.EnemyMMR != nil {
		v := *c.EnemyMMR
		r.EnemyMMR = &v
	}
	return r
}

// GetEncounters retourne les joueurs les plus fréquemment croisés.
//
// Phase C+ multi-titres : si un dataAdapter est injecté et que sa capability
// career.progression est supportée, la lecture passe par LoadEncounters avec
// projection canonical.EncounterRow → domain.EncounterDTO. Sinon fallback
// gracieux sur s.repo.GetEncounters (parité comportementale par construction).
func (s *CareerService) GetEncounters(ctx context.Context) (domain.CareerEncountersResponse, error) {
	rows, err := s.loadEncounterRows(ctx)
	if err != nil {
		return domain.CareerEncountersResponse{}, fmt.Errorf("CareerService.GetEncounters: %w", err)
	}

	// Init [] plutôt que nil : un slice nil sérialise en JSON `null` et crashe
	// le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	teammates := make([]domain.EncounterDTO, 0)
	enemies := make([]domain.EncounterDTO, 0)
	for _, r := range rows {
		if r.AsTeammate >= r.AsEnemy {
			teammates = append(teammates, r)
		} else {
			enemies = append(enemies, r)
		}
	}

	return domain.CareerEncountersResponse{
		Teammates: teammates,
		Enemies:   enemies,
		Total:     len(rows),
	}, nil
}

// GetTopEncounters retourne les 10 joueurs les plus croisés au niveau carrière
// globale, hors amis configurés (FriendGamertags). Enrichit chaque encounter
// avec les badges narratifs (ally_plus / tough_enemy / ordinal) via le même
// algorithme que MatchView.
func (s *CareerService) GetTopEncounters(ctx context.Context) (domain.CareerTopEncountersResponse, error) {
	excludeXUIDs := s.resolveFriendXUIDs(ctx)
	encounters, stats, err := s.repo.GetTopEncountersGlobal(ctx, excludeXUIDs)
	if err != nil {
		return domain.CareerTopEncountersResponse{}, fmt.Errorf("CareerService.GetTopEncounters: %w", err)
	}
	// Index stats par xuid pour O(1) lookup lors de l'application des badges.
	statsByXUID := make(map[string]domain.EncounterStatsRaw, len(stats))
	for _, st := range stats {
		statsByXUID[st.XUID] = st
	}
	out := make([]domain.MatchEncounterRow, 0, len(encounters))
	for _, e := range encounters {
		st, ok := statsByXUID[e.XUID]
		if !ok {
			out = append(out, e)
			continue
		}
		e.Badges = computeCareerEncounterBadges(e, st)
		out = append(out, e)
	}
	return domain.CareerTopEncountersResponse{Items: out}, nil
}

// GetRivals retourne le top 10 des némésis (deaths DESC) et top 10 des
// souffre-douleur (frags DESC). Le ratio est calculé côté service (frags/deaths
// avec garde div-par-zéro : 0 morts → ratio = float64(Frags)).
func (s *CareerService) GetRivals(ctx context.Context) (domain.CareerRivalsResponse, error) {
	nemesesRaw, victimsRaw, err := s.repo.GetRivals(ctx)
	if err != nil {
		return domain.CareerRivalsResponse{}, fmt.Errorf("CareerService.GetRivals: %w", err)
	}
	return domain.CareerRivalsResponse{
		Nemeses: convertRivals(nemesesRaw),
		Victims: convertRivals(victimsRaw),
	}, nil
}

// resolveFriendXUIDs résout la liste des amis configurés (gamertags) en XUIDs.
// Dégrade gracieusement : skip silencieux pour chaque gamertag non résolvable.
// En cas d'amis non résolus, log Warn pour signaler une dérive de config (un
// gamertag dans settings n'existe ni dans xuid_aliases ni dans match_participants).
func (s *CareerService) resolveFriendXUIDs(ctx context.Context) []string {
	if s.friendGamertags == nil || s.friendXUIDResolver == nil {
		return nil
	}
	gts := s.friendGamertags(ctx)
	if len(gts) == 0 {
		return nil
	}
	out := make([]string, 0, len(gts))
	var unresolved []string
	for _, gt := range gts {
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		xuid, err := s.friendXUIDResolver(ctx, gt)
		if err != nil || xuid == "" {
			unresolved = append(unresolved, gt)
			continue
		}
		out = append(out, xuid)
	}
	if len(unresolved) > 0 {
		slog.WarnContext(ctx, "career.top_encounters.friends_unresolved",
			"unresolved", unresolved,
			"resolved", len(out),
		)
	}
	return out
}

// computeCareerEncounterBadges applique narrative.ComputeEncounterBadges
// (ordinal + ally_plus + tough_enemy) avec le même protocole que MatchView.
func computeCareerEncounterBadges(e domain.MatchEncounterRow, st domain.EncounterStatsRaw) []domain.MatchEncounterBadge {
	winrateAsAlly := encounterBadgeWinrate(st.WinsAsAlly, st.LossesAsAlly)
	winrateVsEnemy := encounterBadgeWinrate(st.WinsVsEnemy, st.LossesVsEnemy)
	stats := narrative.EncounterStats{
		XUID:            e.XUID,
		Gamertag:        e.Gamertag,
		TotalEncounters: e.CountTogether,
		AllyCount:       st.AllyCount,
		EnemyCount:      st.EnemyCount,
		WinrateAsAlly:   winrateAsAlly,
		WinrateVsEnemy:  winrateVsEnemy,
		KillsDealt:      st.KillsDealt,
		DeathsSuffered:  st.DeathsSuffered,
	}
	ordinal := e.CountTogether - 1
	if ordinal < 0 {
		ordinal = 0
	}
	raw := narrative.ComputeEncounterBadges(stats, ordinal)
	if len(raw) == 0 {
		return nil
	}
	out := make([]domain.MatchEncounterBadge, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return out
}

// encounterBadgeWinrate : retourne nil si W+L == 0, sinon le ratio (0..1).
// Mirror de service.encounterWinrate (match_view_service.go) — duplication
// volontaire pour éviter le couplage entre les deux services.
func encounterBadgeWinrate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// convertRivals projette CareerRivalRawRow → CareerRival (calcule le ratio).
func convertRivals(raw []domain.CareerRivalRawRow) []domain.CareerRival {
	out := make([]domain.CareerRival, 0, len(raw))
	for _, r := range raw {
		var ratio float64
		if r.Deaths > 0 {
			ratio = float64(r.Frags) / float64(r.Deaths)
		} else {
			ratio = float64(r.Frags) // 0 morts → ratio = nb de frags (semantically "infini" approximé)
		}
		out = append(out, domain.CareerRival{
			Gamertag:   r.Gamertag,
			Frags:      r.Frags,
			Deaths:     r.Deaths,
			Ratio:      ratio,
			MatchCount: r.MatchCount,
		})
	}
	return out
}

// loadEncounterRows centralise la résolution repo/adapter et garantit la
// même forme de sortie []domain.EncounterDTO quel que soit le chemin.
func (s *CareerService) loadEncounterRows(ctx context.Context) ([]domain.EncounterDTO, error) {
	if s.dataAdapter != nil {
		canonicalRows, err := s.dataAdapter.LoadEncounters(ctx, "")
		if err == nil {
			out := make([]domain.EncounterDTO, 0, len(canonicalRows))
			for _, r := range canonicalRows {
				out = append(out, encounterDTOFromCanonical(r))
			}
			return out, nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
		// capability not supported → fallback repo
	}

	rows, err := s.repo.GetEncounters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.EncounterDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.EncounterDTO{
			Gamertag:   r.Gamertag,
			XUID:       r.XUID,
			MatchCount: r.MatchCount,
			AsTeammate: r.AsTeammate,
			AsEnemy:    r.AsEnemy,
			AvgKDA:     r.AvgKDA,
		})
	}
	return out, nil
}

// encounterDTOFromCanonical projette canonical.EncounterRow → domain.EncounterDTO
// avec strictement la même forme JSON que la projection legacy depuis
// domain.EncounterRawRow. Garantit la parité de payload pour la golden parity.
func encounterDTOFromCanonical(r canonical.EncounterRow) domain.EncounterDTO {
	return domain.EncounterDTO{
		Gamertag:   r.Identity.Gamertag,
		XUID:       r.Identity.XUID,
		MatchCount: r.MatchCount,
		AsTeammate: r.AsTeammate,
		AsEnemy:    r.AsEnemy,
		AvgKDA:     r.AvgKDA,
	}
}

// ---------------------------------------------------------------------------
// Top matches helpers (split/reverse/convert)
// ---------------------------------------------------------------------------

func splitTopRows(rows []domain.TopMatchRawRow) (best, worst []domain.TopMatchRawRow) {
	for _, r := range rows {
		if r.Outcome == domain.OutcomeWin {
			best = append(best, r)
		} else {
			worst = append(worst, r)
		}
	}
	return
}

func reverseTopMatches(rows []domain.TopMatchRawRow) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

func convertTopMatches(rows []domain.TopMatchRawRow) []domain.TopMatchDTO {
	out := make([]domain.TopMatchDTO, 0, len(rows))
	for _, r := range rows {
		mapUI := derefStr(r.MapName)
		modeUI := derefStr(r.PairName)
		var mapPtr, modePtr *string
		if mapUI != "" {
			mapPtr = &mapUI
		}
		if modeUI != "" {
			modePtr = &modeUI
		}
		var startPtr *string
		if r.StartTime != nil {
			s := r.StartTime.UTC().Format(time.RFC3339)
			startPtr = &s
		}
		out = append(out, domain.TopMatchDTO{
			MatchID:          r.MatchID,
			StartTime:        startPtr,
			PerformanceScore: r.PerformanceScore,
			MapUI:            mapPtr,
			ModeUI:           modePtr,
			OutcomeCode:      r.Outcome,
			OutcomeLabel:     outcomeLabel(r.Outcome),
			Kills:            r.Kills,
			Deaths:           r.Deaths,
			KDA:              r.KDA,
		})
	}
	return out
}
