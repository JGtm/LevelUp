package skill_v2

// squad.go : correction de skill pour les escouades (TS2 §7, version offset
// additif Phase 2).
//
// Problème : le modèle traite N joueurs d'une même équipe comme N joueurs
// indépendants. Or un squad coordonné gagne PLUS que la somme de ses parties.
// Conséquence : en attribuant les victoires au seul skill individuel, le modèle
// SUR-estime le μ des joueurs qui jouent souvent en escouade.
//
// Correction : on mesure hors-ligne (cmd/lusr_v2_squad_estimate) un "offset de
// synergie" par paire de coéquipiers. Au runtime, on ajoute cet offset au μ
// EFFECTIF avant l'update EP : l'équipe paraît plus forte, le résultat est donc
// moins surprenant, et le μ INDIVIDUEL bouge moins (la victoire est en partie
// attribuée à la synergie, pas au skill individuel). L'offset étant constant
// pour le match, il est retiré du posterior après l'update — seul le delta de
// l'EP s'applique au μ individuel. σ n'est jamais modifié par l'offset.

// SquadOffsetCap borne l'offset de synergie (en unités μ) à ±2.0. Garde-fou
// contre des offsets délirants estimés sur un petit échantillon de matchs.
const SquadOffsetCap = 2.0

// SquadCoMatch représente une occurrence où une paire de coéquipiers a joué
// dans la même équipe. Won = issue du match du POV de l'équipe (1.0 victoire,
// 0.5 nul, 0.0 défaite) ; SoloWinProb = proba de victoire prédite par les
// ratings SOLO (sans correction squad) avant le match.
type SquadCoMatch struct {
	Won         float64
	SoloWinProb float64
}

// ComputeSquadOffset estime l'offset de synergie d'une paire à partir de
// l'historique de leurs matchs joués ensemble.
//
// Approche : résidu moyen de sur-performance = moyenne de (Won - SoloWinProb).
// Si la paire gagne systématiquement plus que ce que ses ratings solo
// prédisent, le résidu est positif → synergie positive. On convertit ce résidu
// (en espace probabilité ∈ [-1, 1]) en unités μ via muPerWinResidual, puis on
// borne à ±SquadOffsetCap.
//
// 0 match d'historique → offset 0 (aucune synergie mesurable).
func ComputeSquadOffset(matches []SquadCoMatch, muPerWinResidual float64) float64 {
	if len(matches) == 0 {
		return 0
	}
	var sum float64
	for _, m := range matches {
		sum += m.Won - m.SoloWinProb
	}
	meanResidual := sum / float64(len(matches))
	return ClampSquadOffset(meanResidual * muPerWinResidual)
}

// ClampSquadOffset borne une valeur d'offset à [-SquadOffsetCap, +SquadOffsetCap].
func ClampSquadOffset(v float64) float64 {
	if v > SquadOffsetCap {
		return SquadOffsetCap
	}
	if v < -SquadOffsetCap {
		return -SquadOffsetCap
	}
	return v
}

// ApplySquadOffset retourne une copie de g avec μ décalé de offset (σ inchangé).
// Sert à construire le μ EFFECTIF avant l'update EP.
func ApplySquadOffset(g Gaussian, offset float64) Gaussian {
	return Gaussian{Mu: g.Mu + offset, Sigma: g.Sigma}
}
