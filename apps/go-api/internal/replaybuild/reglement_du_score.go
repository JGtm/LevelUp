package replaybuild

import "levelup/go-api/internal/games/halo_infinite/film/replay"

// poserLeReglementDuScore pose sur l'entree du calque de score ce que le NOM DE VARIANTE et la
// table de reglement du titre savent et que le film ne dit pas. `in` nil (aucun enregistrement
// d'entite) ne recoit rien.
//
//	la CIBLE DE VICTOIRE      la table (`[score_target]`) ; sa garde de publication vit chez le
//	                          calque (`publishableTarget` : une table perimee se tait).
//	le MODE A COLLINE         la garde de mode de la barre de garde (`ScoreInput.HillScoring`),
//	                          lue sur la variante par le meme predicat que la methode des
//	                          collines (`isHillVariant`) : Bases et Total Control n'y entrent pas.
//	l'ENTREE DE SEUIL         la table (`[hold_ticks_per_point]`), simple REPLI : le seuil se
//	                          mesure dans le film, aux points du match ; la table ne sert qu'un
//	                          match sans point lisible (cf. replay/hill_hold_threshold.go).
//
// UN SEUL SITE, ET UN GARDE-RAIL : la pose etait recopiee par la cuisson et par deux outils de
// recherche ; `TestReglementDuScoreUnSeulSite` interdit qu'elle se recopie.
func (b *Builder) poserLeReglementDuScore(in *replay.ScoreInput, variant string) {
	if in == nil {
		return
	}
	in.TargetScore, _ = b.regulation.ScoreTarget(variant)
	in.HillScoring = isHillVariant(variant)
	in.HoldTicksPerPointTable, _ = b.regulation.HoldTicksPerPoint(variant)
}
