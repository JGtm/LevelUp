// Package teammates — teammates_service_trends.go : la vue Escouade de la page Tendances.
//
// La population d'une composition est celle de la page Escouade (ADR 0033) : mêmes
// lectures (top coéquipiers, matchs communs, équipe alliée) et même filtre de
// composition stricte. L'équipe alliée est lue UNE fois par requête (ADR 0036, invariant
// 4) : elle sert au filtre strict ET aux statistiques des membres. Le calcul est celui du
// paquet pur analysis/trends ; ce fichier ne fait que lire et assembler.
package teammates

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/trends"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// TrendsDeps regroupe ce que le câblage injecte pour la vue Escouade des tendances (le
// paquet n'importe pas internal/sync).
type TrendsDeps struct {
	// Loc : fuseau des jours, semaines et mois ; nil = UTC.
	Loc *time.Location
	// Now : horloge ; nil = horloge système.
	Now func() time.Time
	// ChainOf classe un match en chaîne de performance (type de partie).
	ChainOf func(pairName string, isRanked, isPvE bool) string
	// HpToKill : points de vie effectifs pour un frag.
	HpToKill float64
	// Capabilities : capacités publiées dans la réponse.
	Capabilities domain.TrendsCapabilities
}

// WithTrends injecte les dépendances de GetSquadTrends.
func (s *TeammatesService) WithTrends(deps TrendsDeps) *TeammatesService {
	if deps.Loc == nil {
		deps.Loc = time.UTC
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	s.trends = deps
	return s
}

// GetSquadTrends construit la vue Escouade pour la composition demandée. Une équipe
// alliée illisible, un historique illisible ou une requête annulée sont des erreurs
// rendues : jamais une page aux parts fausses. Un coéquipier introuvable sort de la
// composition (comme sur la page Escouade) ; sans aucun coéquipier résolu, la réponse est
// valide et sans indicateur.
func (s *TeammatesService) GetSquadTrends(
	ctx context.Context, playerXUID string, req domain.TrendsQueryRequest,
) (domain.TrendsPageResponse, error) {
	s, _ = s.pourLaRequete()
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.TrendsPageResponse{}, fmt.Errorf("TeammatesService: PlayerMatchesRepo non câblé (tendances Escouade)")
	}
	if s.trends.Now == nil || s.trends.Loc == nil {
		return domain.TrendsPageResponse{}, fmt.Errorf("TeammatesService: WithTrends non appelé (horloge ou fuseau absents, tendances Escouade)")
	}
	stop := timing.FromContext(ctx).Section("top_teammates")
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	stop()
	if err != nil {
		return domain.TrendsPageResponse{}, fmt.Errorf("TeammatesService: %w", err)
	}
	compo, err := s.lireComposition(ctx, playerXUID, req.SelectedGamertags, topRows)
	if err != nil {
		return domain.TrendsPageResponse{}, err
	}
	members := s.membresDeTendances(playerXUID, &compo)
	opts := trends.SquadOptions{
		Options: trends.Options{Now: s.trends.Now(), Loc: s.trends.Loc, GameType: req.GameType, HpToKill: s.trends.HpToKill},
		Members: members[:1],
	}
	if len(compo.selectedXUIDs) == 0 {
		return s.habillerTendances(trends.BuildSquad(nil, opts)), nil
	}

	allies, teamByMatch, err := s.lireEquipeAlliee(ctx, playerXUID, collectMatchIDs(compo.roster))
	if err != nil {
		return domain.TrendsPageResponse{}, s.erreurTendances(ctx, "equipe alliee", err)
	}
	kept := compo.roster
	if req.ExactComposition {
		kept = s.filtrerCompositionExacte(ctx, playerXUID, compo, teamByMatch).kept
	}
	matches, alone, err := s.matchsDeTendances(ctx, kept)
	if err != nil {
		return domain.TrendsPageResponse{}, err
	}
	trends.AttachSquad(matches, trends.SquadSamples(allies))
	opts.Members = members
	opts.Alone = alone
	slog.DebugContext(ctx, "teammates.squad_trends_resolved",
		"player", s.gamertag, "selected_count", len(compo.selectedXUIDs),
		"composition_matches", len(matches), "alone_matches", len(alone), "exact", req.ExactComposition)
	return s.habillerTendances(trends.BuildSquad(matches, opts)), nil
}

// matchsDeTendances lit les lignes canoniques du joueur principal : les matchs dont
// l'identifiant est dans kept (la composition) et ceux joués sans amis.
func (s *TeammatesService) matchsDeTendances(
	ctx context.Context, kept []domain.SquadMatchRow,
) (compo, alone []trends.Match, err error) {
	stop := timing.FromContext(ctx).Section("player_matches")
	rows, err := s.playerMatchesRepo.LoadPlayerMatches(ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{})
	stop()
	if err != nil {
		return nil, nil, s.erreurTendances(ctx, "historique du joueur", err)
	}
	inCompo := make(map[string]struct{}, len(kept))
	for _, r := range kept {
		inCompo[r.MatchID] = struct{}{}
	}
	defer timing.FromContext(ctx).Section("trends_build")()
	for _, m := range trends.FromCanonical(rows, trends.FromOptions{Loc: s.trends.Loc, ChainOf: s.trends.ChainOf}) {
		if _, ok := inCompo[m.ID]; ok {
			compo = append(compo, m)
		}
		if !m.IsWithFriends {
			alone = append(alone, m)
		}
	}
	return compo, alone, nil
}

// habillerTendances complète la réponse du paquet pur : la vue et les capacités.
func (s *TeammatesService) habillerTendances(resp domain.TrendsPageResponse) domain.TrendsPageResponse {
	resp.View = domain.TrendsViewSquad
	resp.Capabilities = s.trends.Capabilities
	return resp
}

// erreurTendances enveloppe une lecture en échec ; une requête annulée rend l'erreur du
// contexte, comme CompositionSessions.
func (s *TeammatesService) erreurTendances(ctx context.Context, lecture string, err error) error {
	if cerr := ctx.Err(); cerr != nil {
		return fmt.Errorf("TeammatesService: requete annulee: %w", cerr)
	}
	return fmt.Errorf("TeammatesService: tendances escouade (%s): %w", lecture, err)
}

// membresDeTendances rend les membres de la composition, uniques par xuid : le joueur
// principal d'abord, puis les coéquipiers dans l'ordre de la requête. Un coéquipier dont
// le xuid est celui du joueur principal ou déjà retenu (deux gamertags d'un même joueur)
// est ignoré, première occurrence gardée. compo est réduit aux coéquipiers retenus, que
// le filtre de composition stricte lit ensuite.
func (s *TeammatesService) membresDeTendances(playerXUID string, compo *compositionLue) []trends.SquadMember {
	members := []trends.SquadMember{{XUID: playerXUID, Gamertag: s.gamertag}}
	seen := map[string]struct{}{playerXUID: {}}
	var xuids, gamertags []string
	for i, x := range compo.selectedXUIDs {
		if _, dup := seen[x]; dup {
			continue
		}
		seen[x] = struct{}{}
		xuids = append(xuids, x)
		gamertags = append(gamertags, compo.selectedGamertags[i])
		members = append(members, trends.SquadMember{XUID: x, Gamertag: compo.selectedGamertags[i]})
	}
	compo.selectedXUIDs, compo.selectedGamertags = xuids, gamertags
	return members
}
