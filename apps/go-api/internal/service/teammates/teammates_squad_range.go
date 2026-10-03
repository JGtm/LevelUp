// Package teammates — teammates_squad_range.go : LES ROLES DE PORTEE de la page Escouade
// (plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lot N2, decision D22-5).
//
// ─── UN PROFIL PAR MATCH, PAS UN AGREGAT DE PERIODE ────────────────────────────────────
//
// Le nuage de la page pose un point par (match, coequipier) : c'est la TENDANCE qui dit qui
// tient la ligne de front et qui a change, et une mediane de periode l'aplatirait. Le
// service ne calcule donc aucun role et aucun seuil de bande — ils se lisent sur une fenetre
// glissante de 5 matchs, cote client, sur l'echelle relative que ce bloc sert.
//
// ─── L ECART AU LOBBY, JAMAIS LA MEDIANE BRUTE ─────────────────────────────────────────
//
// La mediane de frag d'un joueur ne se compare d'un match a l'autre que RAPPORTEE a celle de
// tous les joueurs du match, camp adverse compris (cf. domain/match_range_profile.go). D'ou
// une lecture SANS filtre de joueur : les huit ou seize joueurs servent de referentiel, et
// seuls les joueurs du roster affiche sont publies.
//
// ─── LE MEME CADRAGE QUE LES AUTRES BLOCS DE LA PAGE ───────────────────────────────────
//
// `firstBloodScope` donne les matchs du perimetre filtre, les xuid du roster dans l'ordre
// (le joueur principal en tete) et leurs gamertags — le MEME cadrage que l'echange, les
// paires d'assistance (Q32d) et le premier frag. Une seconde definition du roster aurait
// donne deux listes de joueurs sur la meme page.
package teammates

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// WithMatchRange injecte le lecteur de portee « tout le lobby ». Cablage INCONDITIONNEL :
// c'est le repo qui sait si ce titre a des positions par kill (games.ErrCapabilityNotSupported),
// et une porte de capability posee ici prendrait la meme decision une seconde fois.
func (s *TeammatesService) WithMatchRange(repo port.MatchRangeRepository) *TeammatesService {
	s.matchRangeRepo = repo
	return s
}

// buildSquadRange assemble le bloc « roles de portee ».
//
// Retourne nil dans tous les cas ou il n'y a rien a dire, et c'est une OMISSION assumee
// (jamais des zeros) : lecteur non cable, perimetre vide, titre sans positions par kill,
// lecture en echec, ou aucun match du perimetre porteur d'un frag mesure.
func (s *TeammatesService) buildSquadRange(
	ctx context.Context,
	scopeRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.MatchRangeBlock {
	defer timing.FromContext(ctx).Section("range_profiles")()
	if s.matchRangeRepo == nil {
		return nil
	}
	scopeIDs, xuidsOrdered, gtByXUID := firstBloodScope(scopeRows, mainGamertag, mainXUID, teammates)
	if len(scopeIDs) == 0 || len(xuidsOrdered) == 0 {
		return nil
	}

	read, err := s.matchRangeRepo.LoadMatchRangeKills(ctx, s.titleSlug, port.WeaponRangeFilters{
		MatchIDs:   scopeIDs,
		AllPlayers: true,
	})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "teammates_portee_titre_sans_positions",
				"player", mainGamertag, "slug", s.titleSlug, "matchs", len(scopeIDs))
			return nil
		}
		slog.WarnContext(ctx, "teammates_portee_lecture_en_echec",
			"player", mainGamertag, "matchs", len(scopeIDs), "err", err)
		return nil
	}

	profiles := analysis.MatchRangeProfiles(analysis.MatchRangeInput{
		Kills:        read.Kills,
		Matches:      squadRangeScope(scopeRows, scopeIDs),
		Publish:      gtByXUID,
		PublishOrder: xuidsOrdered,
	})
	if len(profiles) == 0 {
		slog.InfoContext(ctx, "teammates_portee_section_retiree_sans_mesure",
			"player", mainGamertag, "matchs", len(scopeIDs),
			"cause", "aucun match du perimetre ne porte de frag mesure")
		return nil
	}
	slog.InfoContext(ctx, "teammates_portee_profils",
		"player", mainGamertag, "matchs_profiles", len(profiles),
		"frags_mesures", len(read.Kills), "frags_publiables", read.KillsTotal)
	return &domain.MatchRangeBlock{
		Profiles:      profiles,
		KillsMeasured: len(read.Kills),
		KillsTotal:    read.KillsTotal,
	}
}

// squadRangeScope habille les match_id (DEJA ordonnes du plus ancien au plus recent par
// firstBloodScope — c'est l'abscisse du nuage) de leur date et de leur carte, pour
// l'etiquette « #N · carte » commune aux graphes de la page.
func squadRangeScope(rows []domain.SquadMatchRow, orderedIDs []string) []analysis.MatchRangeMatch {
	parID := make(map[string]domain.SquadMatchRow, len(rows))
	for _, r := range rows {
		if _, dup := parID[r.MatchID]; !dup {
			parID[r.MatchID] = r
		}
	}
	out := make([]analysis.MatchRangeMatch, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		r := parID[id]
		out = append(out, analysis.MatchRangeMatch{
			MatchID: id, PlayedAt: r.StartTime, MapName: squadRangeMapName(r),
		})
	}
	return out
}

// squadRangeMapName prefere le libelle d'affichage de la carte (MapUI, deja resolu par le
// repo) au nom brut — la MEME preference que le tableau historique de la page.
func squadRangeMapName(r domain.SquadMatchRow) string {
	if r.MapUI != "" {
		return r.MapUI
	}
	return r.MapName
}
