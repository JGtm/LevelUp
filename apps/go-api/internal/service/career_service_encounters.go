// Package service — career_service_encounters.go : encounters, top encounters
// globaux et rivals pour la page Carrière.
// Découpé de career_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/service/teammates"
)

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
// globale, hors amis configurés du joueur. Enrichit chaque encounter avec
// le MÊME jeu de badges que la Match View et le hub Relations (4 badges de rencontre
// + 5 badges « solid »), via le calcul partagé relations.ComputeBadges.
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
	now := time.Now()
	out := make([]domain.MatchEncounterRow, 0, len(encounters))
	for _, e := range encounters {
		st, ok := statsByXUID[e.XUID]
		if !ok {
			out = append(out, e)
			continue
		}
		e.Badges = computeCareerEncounterBadges(e, st, now)
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

// resolveFriendXUIDs résout la liste des amis configurés (gamertags) en XUIDs, TOUS à la fois,
// par la résolution partagée avec la composition stricte de l'Escouade
// (teammates.ResolveFriendXUIDs) : d'abord le registre des profils suivis — un ami suivi a un
// xuid connu, sans lecture —, puis une seule lecture pour les autres
// (CareerRepo.ResolveFriendXUIDs : alias puis participants de l'historique du joueur, sans
// casse). Dégrade gracieusement : un ami non résolu n'est pas exclu (journalisé). XUIDs triés :
// la liste d'exclusion ne dépend pas de l'ordre d'itération d'une map.
func (s *CareerService) resolveFriendXUIDs(ctx context.Context) []string {
	if s.friendGamertags == nil || s.friendXUIDs == nil {
		return nil
	}
	var suivis map[string]string
	if s.friendsSuivis != nil {
		suivis = s.friendsSuivis(ctx)
	}
	resolus := teammates.ResolveFriendXUIDs(ctx, "career", s.friendGamertags(ctx), suivis, s.friendXUIDs)
	return slices.Sorted(maps.Keys(resolus))
}

// computeCareerEncounterBadges attribue les badges de rencontre de la page Carrière
// via le calcul PARTAGÉ (relations.ComputeBadges) — MÊME jeu de badges que la Match
// View et le hub Relations (les 4 badges de rencontre + les 5 « solid »). now sert
// aux badges temporels (recrue / ancien) alimentés par st.FirstSeen.
func computeCareerEncounterBadges(e domain.MatchEncounterRow, st domain.EncounterStatsRaw, now time.Time) []domain.MatchEncounterBadge {
	relStats := relationStatsFromEncounterStats(e.XUID, e.Gamertag, e.CountTogether, st, true)
	return convertRelationBadges(relations.ComputeBadges(relStats, now))
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
