// Package teammates — teammates_squad_assist_pairs.go : le tableau « qui assiste qui »
// de la page Synergies.
//
// Même table et même doctrine que le graphe de la page match (Q21d) — voir
// domain/assist_pairs.go pour les trois états de l'assistance et la réserve sur les
// parts de dégâts. La différence tient en deux points, portés par Q32d :
//
//   - les DEUX joueurs de la paire sont contraints à l'escouade. La page pose une
//     question sur l'escouade : une assistance rendue à un allié de passage n'y répond
//     pas et fausserait le dénominateur de la colonne « part » ;
//   - la COUVERTURE est en matchs, pas en morts. Sur une sélection, ce qui manque n'est
//     pas « quelques morts » mais des MATCHS ENTIERS dont le film a expiré côté serveur.
//     C'est cette granularité que l'utilisateur peut recouper avec sa propre liste.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// buildSquadAssistPairs assemble le bloc `assist_pairs` de la page Escouade.
//
// Retourne nil dans les trois cas où il n'y a rien à dire, et un bloc dans tous les
// autres :
//
//	repo absent / périmètre vide     pas de lecture possible.
//	matches_measured == 0            AUCUN match de la sélection n'a d'assistance
//	                                 mesurée. C'est aussi l'état d'un titre sans
//	                                 décodeur de film — obtenu sans jamais regarder le
//	                                 slug, par la donnée seule.
//	sinon                            le bloc, y compris avec zéro paire : « mesuré sur
//	                                 N des M matchs, aucune assistance interne » est un
//	                                 fait, pas un vide.
//
// Les gamertags viennent du ROSTER de la page (main + coéquipiers sélectionnés), jamais
// du film : c'est ce qui garantit qu'un joueur n'apparaît pas sous deux orthographes
// dans le même tableau. Une paire dont un xuid n'est pas au roster est impossible par
// construction (Q32d les contraint), mais si elle survenait elle serait ÉCARTÉE plutôt
// qu'affichée sans nom.
func (s *TeammatesService) buildSquadAssistPairs(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadAssistPairs {
	if s.repo == nil {
		return nil
	}
	// Périmètre : mêmes matchs et mêmes joueurs que les autres blocs de la page
	// (helper partagé avec « premier frag / première mort »).
	matchIDs, xuidsOrdered, gtByXUID := firstBloodScope(allSquadRows, mainGamertag, mainXUID, teammates)
	if len(matchIDs) == 0 || len(xuidsOrdered) == 0 {
		return nil
	}

	raw, measured, err := s.repo.LoadSquadAssistPairs(ctx, matchIDs, xuidsOrdered)
	if err != nil {
		slog.WarnContext(ctx, "teammates_assist_pairs_load_failed",
			"matchs", len(matchIDs), "err", err)
		return nil
	}
	if measured == 0 {
		return nil
	}

	pairs := make([]domain.SquadAssistPair, 0, len(raw))
	total := 0
	for _, p := range raw {
		assistGT, okA := gtByXUID[p.AssistXUID]
		killerGT, okK := gtByXUID[p.KillerXUID]
		if !okA || !okK {
			// Impossible par construction (Q32d contraint les deux côtés au roster).
			// Si ça arrivait, afficher une ligne sans nom serait pire que l'écarter :
			// on le signale et on passe.
			slog.WarnContext(ctx, "teammates_assist_pair_hors_roster",
				"assist_xuid", p.AssistXUID, "killer_xuid", p.KillerXUID)
			continue
		}
		total += p.AssistCount
		pairs = append(pairs, domain.SquadAssistPair{
			AssistXUID:     p.AssistXUID,
			AssistGamertag: assistGT,
			KillerXUID:     p.KillerXUID,
			KillerGamertag: killerGT,
			AssistCount:    p.AssistCount,
			StolenCount:    p.StolenCount,
		})
	}
	return &domain.SquadAssistPairs{
		MatchesMeasured: measured,
		MatchesTotal:    len(matchIDs),
		TotalAssists:    total,
		Pairs:           pairs,
	}
}
