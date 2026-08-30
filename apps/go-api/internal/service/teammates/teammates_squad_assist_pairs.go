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

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// compteurAssistPairsSansMesure : le nombre de fois où la page a RETIRÉ le tableau faute de
// mesure. Publié en expvar (ADR 0009), donc lisible sur /debug/vars sans redéploiement.
// Son jumeau côté vue match est `compteurMatchAssistSansMesure` (internal/service). Publié
// TITRE-AWARE : `halo_5` est actif, et une clé nue confondrait « la passe de film s'est
// arrêtée » avec « quelqu'un a navigué dans un autre titre ».
const compteurAssistPairsSansMesure = "assist_pairs_squad_retire_sans_mesure_total"

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
		// UN BLOC QUI SE RETIRE LAISSE UNE TRACE — ET C EST LA LEÇON DU 2026-08-29.
		// Ce `return nil` a fait disparaître le tableau de la page pendant CINQ MOIS sans
		// qu'aucun log, compteur ni ligne d'admin ne le dise : l'effondrement de
		// `assist_known` (cache de films gelé le 2026-04-07, registre
		// `.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`) n'a été découvert que parce qu'un
		// utilisateur a demandé pourquoi un tableau manquait. Le silence était le défaut.
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), compteurAssistPairsSansMesure)
		slog.InfoContext(ctx, "teammates_assist_pairs_bloc_retire_sans_mesure",
			"matchs", len(matchIDs), "joueurs", len(xuidsOrdered),
			"cause", "aucune mort de la sélection ne porte assist_known — passe de film absente")
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
