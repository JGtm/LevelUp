package replay

// hill_hold_threshold.go — LE SEUIL DE GARDE SE LIT DANS LE FILM, MATCH PAR MATCH.
//
// Le seuil est le nombre de tics de garde qui valent un point (`ScoreTimeline.HoldTicksPerPoint`).
// Le film porte les deux pieces qui le donnent : le compteur de garde de chaque camp
// (`HoldTicks`, cf. hill_hold_ticks.go) et les instants de point (les paliers de `Teams`). AU
// POINT, LA GARDE ACCUMULEE PAR LE CAMP QUI MARQUE DEPUIS LE POINT PRECEDENT EST LE SEUIL.
// L'artefact s'assemble une fois le film lu : des qu'un point est marque, le seuil vaut pour tout
// le match, depuis la premiere image.
//
// # LA STATISTIQUE : LA VALEUR LA PLUS FREQUENTE, ET A EGALITE LA PLUS HAUTE
//
// Les gains au point ne valent pas tous le seuil, et les deux ecarts vont dans des sens opposes :
//
//	EN DESSOUS   une seconde de garde interrompue (colline contestee, perdue puis reprise) peut
//	             perdre son tic : le gain vaut alors le seuil moins un ou deux. Releve sur 11
//	             films : 34 et 33 en arene (seuil 35), 39 en classe (seuil 40), le dernier tic
//	             tombant toujours sur l'image du point. Jamais au-dessus sur un intervalle sain.
//	AU-DESSUS    un point que la courbe de score ne porte pas (courbes anormales de deux films
//	             classes de Lattice) fusionne deux intervalles : 73 tics sur `26602661`.
//
// Le MAXIMUM tomberait dans le second piege, la MEDIANE dans le premier quand les points sont
// peu nombreux. La valeur la plus frequente ecarte les deux, puisque le seuil exact est la
// valeur qui se repete ; a egalite, la plus haute gagne, parce que le seul ecart d'un intervalle
// sain est un tic PERDU. Mesure : 35 sur les sept films d'arene et de Doubles, 40 sur cinq des six
// films classes (`26602661` compris, malgre son 73). Le sixieme, `7de0b91d`, n'a qu'un intervalle
// sain, et il a perdu un tic : 39, desaccord avec la table compte (le film prime).
//
// # UN INTERVALLE INCOHERENT N'EST PAS UN POINT
//
// Sur un intervalle sain, le camp qui marque a la plus grande garde : l'autre n'a pas atteint
// le seuil, sinon il aurait marque. Un point ou un autre camp a garde AUTANT OU PLUS est le signe
// d'un point manquant dans la courbe, et il est ecarte (compte au journal de cuisson) —
// `7de0b91d` : 66 tics pour le camp qui marque, 77 pour l'autre.
//
// # UN TIC PAR POINT : PAS DE BARRE
//
// Sur un match au score a la seconde (`e449a696`, 180 points), chaque tic de garde est un point :
// le seuil mesure vaut 1, et une barre qui se remplit et se vide a chaque seconde ne dirait rien
// que le bandeau de score ne montre deja. Ni la serie ni le seuil ne sont publies.
//
// # LA TABLE N'EST PLUS QU'UN REPLI, ET ELLE NE CONTREDIT PAS LE FILM
//
// Un match sans aucun point lisible ne porte pas son seuil : l'entree de sa variante dans
// `regulation.toml [hold_ticks_per_point]` est alors publiee, et le repli est compte au registre
// (`repli_seuil_garde_table_de_variante`). Sans entree, pas de barre. Quand le film et la table
// divergent, LE FILM PRIME, et la contradiction se compte au meme registre
// (`repli_seuil_garde_table_contredite`) : les deux se lisent dans `coverage.fallbacks[]`, sans
// champ neuf dans le document.
//
// LA SOURCE NOMINALE EST LE FILM : un seuil publie sans aucune de ces deux entrees dans
// `coverage.fallbacks[]` est « mesure dans le film ». Le detail de la lecture (points retenus,
// ecartes, valeur du film, entree de table) n'est pas publie : il est journalise par cuisson
// (`logHoldThreshold`, build_score.go).

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// Les sources du seuil de garde, telles que le journal de cuisson les nomme.
const (
	// holdThresholdFromFilm : mesure dans le film, aux points du match.
	holdThresholdFromFilm = "mesure dans le film"
	// holdThresholdFromTable : repli, l'entree de la variante dans la table du titre.
	holdThresholdFromTable = "table de la variante"
)

// holdThresholdMinTicks : le plus petit seuil qui donne une barre. A un tic par point, la garde
// EST le score (cf. en-tete).
const holdThresholdMinTicks = 2

// holdThresholdReading est ce que la resolution du seuil a lu d'un match a colline. Elle n'est
// pas publiee (cf. en-tete) : `coverage.fallbacks[]` porte le repli et la contradiction, le
// journal de cuisson porte le reste.
type holdThresholdReading struct {
	// source nomme l'origine du seuil publie ; vide quand aucun seuil n'est publie.
	source string
	// film est le seuil mesure aux points du match ; 0 quand aucun point n'est lisible.
	film int
	// points sont les points retenus pour la mesure ; rejected ceux ecartes (un autre camp a
	// garde autant ou plus que celui qui marque, signe d'un point absent de la courbe de score).
	points, rejected int
	// table est l'entree de la variante dans la table du titre ; 0 quand elle n'y est pas.
	table int
	// tableDisagrees : le film et la table divergent, le film prime.
	tableDisagrees bool
	// oneTickPerPoint : chaque tic de garde est un point (score a la seconde), rien n'est publie.
	oneTickPerPoint bool
}

// resolveHoldThreshold rend le seuil a publier (0 = aucun) et ce qui a ete lu.
//
// `table` est l'entree de la variante (0 = absente). Repli et contradiction ne se comptent que
// quand un seuil est publie : sans serie de garde, rien n'est publie et rien n'est compte.
func resolveHoldThreshold(holds []TeamHold, teams []TeamScore, table int,
	fb *fallback.Compteur,
) (int, holdThresholdReading) {
	gains, rejected := holdGainsAtPoints(holds, teams)
	r := holdThresholdReading{points: len(gains), rejected: rejected, table: table,
		film: mostFrequentHighest(gains)}
	switch {
	case len(holds) == 0:
		return 0, r
	case r.film > 0 && r.film < holdThresholdMinTicks:
		r.oneTickPerPoint = true
		return 0, r
	case r.film > 0:
		r.source = holdThresholdFromFilm
		if table > 0 && table != r.film {
			r.tableDisagrees = true
			fb.Declenche(fallback.NomSeuilGardeTableContredite)
		}
		return r.film, r
	case table > 0:
		r.source = holdThresholdFromTable
		fb.Declenche(fallback.NomSeuilGardeTableDeVariante)
		return table, r
	default:
		return 0, r
	}
}

// holdPoint est un point marque : son image et le camp qui marque.
type holdPoint struct{ t, team int }

// scoredPoints rend les points marques, tries par image : les montees de la courbe cumulee de
// chaque camp situe. Une reemission de la meme valeur, ou une valeur qui descend, n'est pas un
// point — meme lecture que le client (`lastResetFrame`, hillHoldLogic.ts).
func scoredPoints(teams []TeamScore) []holdPoint {
	var out []holdPoint
	for _, tm := range teams {
		if tm.TeamID == nil {
			continue
		}
		prev := 0
		for _, k := range tm.Total {
			if k.V > prev {
				out = append(out, holdPoint{t: k.T, team: *tm.TeamID})
				prev = k.V
			}
		}
	}
	// Comparateur TOTAL : un camp ne monte qu'une fois par image (`scoreTicksOf`), donc
	// (image, camp) est une cle unique.
	slices.SortFunc(out, func(a, b holdPoint) int {
		return cmp.Or(cmp.Compare(a.t, b.t), cmp.Compare(a.team, b.team))
	})
	return out
}

// holdGainsAtPoints rend, pour chaque point coherent, la garde prise par le camp qui marque
// depuis le point precedent (tous camps confondus) ; et le nombre de points ecartes.
//
// L'intervalle est `]point precedent, point]`, la borne du client : la barre repart de zero a
// l'image d'un point, et le tic qui tombe sur cette image appartient a l'intervalle qu'il clot.
func holdGainsAtPoints(holds []TeamHold, teams []TeamScore) (gains []int, rejected int) {
	series := map[int][]ScoreTick{}
	for _, h := range holds {
		if h.TeamID != nil {
			series[*h.TeamID] = h.Ticks
		}
	}
	since := -1
	for _, p := range scoredPoints(teams) {
		if g, ok := coherentGain(series, p, since); ok {
			gains = append(gains, g)
		} else {
			rejected++
		}
		since = p.t
	}
	return gains, rejected
}

// coherentGain rend la garde prise par le camp qui marque sur `]since, p.t]`, et faux quand
// l'intervalle n'est pas sain : rien de pris, ou un autre camp a garde autant ou plus.
func coherentGain(series map[int][]ScoreTick, p holdPoint, since int) (int, bool) {
	own, ok := series[p.team]
	if !ok {
		return 0, false
	}
	gain := holdValueAt(own, p.t) - holdValueAt(own, since)
	if gain <= 0 {
		return 0, false
	}
	for team, ticks := range series {
		if team != p.team && holdValueAt(ticks, p.t)-holdValueAt(ticks, since) >= gain {
			return 0, false
		}
	}
	return gain, true
}

// holdValueAt rend la valeur d'une serie cumulative en escalier a l'image `frame` (0 avant la
// premiere emission).
func holdValueAt(ticks []ScoreTick, frame int) int {
	v := 0
	for _, k := range ticks {
		if k.T > frame {
			break
		}
		v = k.V
	}
	return v
}

// mostFrequentHighest rend la valeur la plus frequente, la plus haute a egalite ; 0 sur une
// liste vide.
func mostFrequentHighest(vals []int) int {
	counts := map[int]int{}
	best, bestN := 0, 0
	for _, v := range vals {
		counts[v]++
	}
	for v, n := range counts {
		if n > bestN || (n == bestN && v > best) {
			best, bestN = v, n
		}
	}
	return best
}
