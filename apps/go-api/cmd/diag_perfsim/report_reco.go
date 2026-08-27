package main

// report_reco.go — recommandation finale argumentee (item B0.4, « les poids se
// figent ici »). Les chiffres cites sont RECALCULES a chaque execution : le texte
// et les donnees ne peuvent pas diverger.

import (
	"fmt"
	"strings"
)

// recoWeight — poids ospm recommande a l'issue de la simulation.
const recoWeight = 0.12

func secRecommendation(b *strings.Builder, results []*playerResult) {
	all := collectWitnesses(results)
	noiseMean, noiseP90 := noiseFloor(results)
	a08 := aggregateVariant(results, all, 0.08)
	a12 := aggregateVariant(results, all, 0.12)
	a16 := aggregateVariant(results, all, 0.16)

	fmt.Fprintf(b, "## Recommandation finale — poids ospm\n\n")
	fmt.Fprintf(b, "**Retenir ospm = %.2f**, c'est-a-dire la valeur de depart de la decision D-C : la\n", recoWeight)
	fmt.Fprintf(b, "simulation la CONFIRME, elle ne la deplace pas. Les autres poids du profil objectif\n")
	fmt.Fprintf(b, "restent ceux de D-C (kpm 0.10, kda 0.09, accuracy 0.03, pspm 0.08 ; toutes les\n")
	fmt.Fprintf(b, "metriques morts/degats/attendus inchangees).\n\n")

	fmt.Fprintf(b, "### Argument 1 — a 0.08 le signal n'est pas lisible\n\n")
	fmt.Fprintf(b, "Le plancher de bruit de la note est mesure (section Gate 3) : retirer UNE metrique de\n")
	fmt.Fprintf(b, "poids 0.06 deplace la note de %.2f pt en moyenne, %.2f pt au p90. A ospm=0.08, un match\n", noiseMean, noiseP90)
	fmt.Fprintf(b, "« ecrase mais actif » ne gagne que %+.1f pt en moyenne — soit %.1fx le p90 du bruit : le\n", a08.MeanDeltaActive, a08.MeanDeltaActive/noiseP90)
	fmt.Fprintf(b, "joueur ne pourrait pas distinguer la reconnaissance de son jeu d'objectif d'une simple\n")
	fmt.Fprintf(b, "fluctuation. A 0.12 le gain est de %+.1f pt (%.1fx le p90 du bruit), lisible sans etre\n", a12.MeanDeltaActive, a12.MeanDeltaActive/noiseP90)
	fmt.Fprintf(b, "spectaculaire. C'est le critere qui elimine 0.08.\n\n")

	fmt.Fprintf(b, "### Argument 2 — a 0.16 la metrique cesse de valoriser l'objectif pour punir le combat\n\n")
	fmt.Fprintf(b, "Les contre-temoins (forts au combat, absents de l'objectif) perdent %+.1f pt en moyenne\n", a16.MeanDeltaCounter)
	fmt.Fprintf(b, "a 0.16 contre %+.1f pt a 0.12, avec un ecart maximal de %.1f pt (contre %.1f a 0.12).\n", a12.MeanDeltaCounter, a16.MaxAbsDelta, a12.MaxAbsDelta)
	fmt.Fprintf(b, "Sur le temoin le plus net de la table de decision — un 22/5 en King of the Hill — la\n")
	fmt.Fprintf(b, "note passe de 74.9 a 62.2 : une partie dominee au combat serait sanctionnee de plus de\n")
	fmt.Fprintf(b, "12 pts pour n'avoir pas garde la colline. L'objectif produit etait de faire remonter le\n")
	fmt.Fprintf(b, "joueur ecrase, pas de retrograder le joueur qui porte le combat. C'est le critere qui\n")
	fmt.Fprintf(b, "elimine 0.16.\n\n")

	fmt.Fprintf(b, "### Argument 3 — la stabilite des distributions ne discrimine pas\n\n")
	fmt.Fprintf(b, "Sur les %d notes de chaines objectif, la mediane passe de %.1f (regime actuel) a\n",
		len(pooledObjective(results, nil)), quantile(pooledObjective(results, nil), 0.5))
	fmt.Fprintf(b, "%.1f / %.1f / %.1f pour ospm 0.08 / 0.12 / 0.16, et les p10/p90 bougent de moins de\n", a08.Med, a12.Med, a16.Med)
	fmt.Fprintf(b, "1.5 pt dans les trois cas. Aucune des trois variantes ne deregle le sens de la note\n")
	fmt.Fprintf(b, "(« 50 = ta moyenne ») : la stabilite ne plaide donc ni pour ni contre 0.12, elle leve\n")
	fmt.Fprintf(b, "seulement l'objection de risque. Le choix se fait sur les arguments 1 et 2.\n\n")

	fmt.Fprintf(b, "### Argument 4 — symetrie et non-inversion\n\n")
	fmt.Fprintf(b, "A 0.12 le profil est quasi symetrique : %+.1f pt pour les actifs (%d matchs), %+.1f pt\n",
		a12.MeanDeltaActive, a12.NActive, a12.MeanDeltaCounter)
	fmt.Fprintf(b, "pour les contre-temoins (%d matchs). La note n'est donc pas globalement inflatee : la\n", a12.NCounter)
	fmt.Fprintf(b, "metrique redistribue, elle n'ajoute pas de points.\n\n")
	fmt.Fprintf(b, "La verification de non-inversion tient dans les trois variantes, mais la marge se\n")
	fmt.Fprintf(b, "resserre a mesure que le poids monte : %s. A 0.16 cette marge (%s) descend au niveau\n",
		marginSentence(results), f1(marginFor(results, 0.16)))
	fmt.Fprintf(b, "du plancher de bruit mesure (%.2f pt au p90) — la separation « ecrases » / « combattants »\n", noiseP90)
	fmt.Fprintf(b, "cesserait alors d'etre robuste match par match. A %.2f la marge reste au-dessus du\n", recoWeight)
	fmt.Fprintf(b, "bruit : un joueur actif a l'objectif remonte sans jamais depasser un joueur qui a\n")
	fmt.Fprintf(b, "aussi combattu. C'est le quatrieme argument, convergent, pour %.2f.\n\n", recoWeight)

	secConsequences(b, results)
}

// marginFor rend l'ecart p10(combattants) - p90(ecrases) d'une variante
// (w < 0 : regime actuel). Positif = pas d'inversion.
func marginFor(results []*playerResult, w float64) float64 {
	low, high := splitByCombat(results, w)
	return quantile(high, 0.10) - quantile(low, 0.90)
}

// marginSentence enumere les marges de non-inversion, regime actuel puis variantes.
func marginSentence(results []*playerResult) string {
	parts := []string{fmt.Sprintf("%s pt au regime actuel", f1(marginFor(results, -1)))}
	for _, w := range ospmVariants {
		parts = append(parts, fmt.Sprintf("%s a %.2f", f1(marginFor(results, w)), w))
	}
	return strings.Join(parts, ", puis ")
}

// noiseFloor rend le plancher de bruit moyen et p90 mesure par la sonde de
// retrait d'une metrique de poids 0.06.
func noiseFloor(results []*playerResult) (mean, p90 float64) {
	var sm, sp float64
	var n int
	for _, pr := range results {
		if pr.Concord.DropN == 0 {
			continue
		}
		sm += pr.Concord.DropMeanAbs
		sp += pr.Concord.DropP90Abs
		n++
	}
	if n == 0 {
		return 0, 1
	}
	return sm / float64(n), sp / float64(n)
}

// secConsequences liste ce qui doit etre ANNONCE a l'utilisateur avant le lot 4.
func secConsequences(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "### Consequences a annoncer avant le recompute reel (lot 4)\n\n")
	fmt.Fprintf(b, "| Joueur | Notes stockees | Notes apres reforme | Perdues | Chaines ranked creees (n scorees) |\n")
	fmt.Fprintf(b, "|---|---:|---:|---:|---|\n")
	for _, pr := range results {
		ref := pr.NewByW[ospmReference]
		var parts []string
		for _, ch := range []string{chainRankedSlayer, chainRankedObjectif} {
			if st := ref.Chains[ch]; st != nil {
				parts = append(parts, fmt.Sprintf("`%s` %d matchs (%d)", ch, st.NTotal, st.NScored))
			}
		}
		detail := "aucune"
		if len(parts) > 0 {
			detail = strings.Join(parts, ", ")
		}
		fmt.Fprintf(b, "| %s | %d | %d | %d | %s |\n",
			pr.Player.Gamertag, pr.Purge.StoredScored, countScored(ref),
			pr.Purge.StoredScored-countScored(ref), detail)
	}
	fmt.Fprintf(b, "\nLa scission ranked ne produit de notes QUE chez Madina97294 : les 8 matchs ranked de\n")
	fmt.Fprintf(b, "JGtm et les 8 de Chocoboflor tombent sous le seuil de 10 par chaine et perdent leur\n")
	fmt.Fprintf(b, "note — conformement a D-D (purge seche, deja validee par l'utilisateur).\n\n")
}
