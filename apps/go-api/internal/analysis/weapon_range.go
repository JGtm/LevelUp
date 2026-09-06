// Package analysis — weapon_range.go : LA PORTÉE ET LE DÉNIVELÉ MESURÉS, PAR ARME.
//
// # CE QUE CE FICHIER CALCULE, ET CE QU'IL NE CALCULE PAS
//
// Il agrège des frags DÉJÀ MESURÉS — un frag mesuré porte son arme, sa distance et son
// dénivelé — en une ligne par couple (arme, côté). Il ne lit aucune base, ne décode aucun
// film, ne classe aucune arme : la jointure `match_kill_events × kill_positions` et la
// classification `source_tag -> weapon_key` vivent chez le repo (lot 3), l'agrégat vit ici.
// C'est la frontière habituelle du dépôt : analysis est PUR.
//
// # LA PORTÉE EST UN USAGE, JAMAIS UNE PORTÉE THÉORIQUE (D8)
//
// Ce que ces nombres décrivent, c'est À QUELLE DISTANCE CE JOUEUR a effectivement tué et
// s'est fait tuer avec cette arme — pas la portée d'ingénierie de l'arme, que le jeu ne
// publie nulle part et que ces mesures ne permettent pas d'inférer. Aucun libellé produit
// ne doit dire « portée de l'arme » (doctrine `REFERENCE_VARIANTES_ARMES_REDDIT.md`).
//
// # POURQUOI p10 -> p90 ET PAS min -> max (D6)
//
// Sur des centaines de frags, le minimum et le maximum décrivent deux accidents : un tir de
// mêlée et un tir chanceux à travers la carte. Le bâton p10 -> p90 décrit la portée où
// l'arme est réellement employée, et la médiane s'y lit comme le centre de gravité.
//
// # LE DÉNIVELÉ EST SIGNÉ, ET LE SIGNE EST CELUI DU CÔTÉ DEMANDÉ (D4)
//
// La question de l'utilisateur est « d'en haut ou d'en bas ». Une classification non signée
// y répondrait par « il y avait un étage d'écart », ce qui n'est pas la même information.
// Voir `signedElevation` pour la convention, qui est le point le plus facile à inverser de
// tout ce fichier.
package analysis

import "sort"

// Side dit DE QUEL CÔTÉ de l'engagement la mesure est lue. Le couple (arme, côté) est la
// clé de l'agrégat : la même arme ne raconte pas la même chose selon qu'elle est la mienne
// ou celle qui m'a tué.
type Side string

const (
	// SideKiller — les frags du joueur, « où je frague ». L'arme est LA SIENNE.
	SideKiller Side = "killer"
	// SideVictim — les morts du joueur, « où je meurs ». L'arme est celle DU TUEUR : une
	// mort ne porte pas l'arme que la victime tenait, mais celle qui l'a abattue.
	SideVictim Side = "victim"
)

// WeaponRangeMinMeasured : nombre minimal de frags mesurés pour qu'un couple (arme, côté)
// soit PUBLIÉ (D9).
//
// D'OÙ VIENT LA VALEUR : 8 est le plus petit effectif sur lequel un p10 et un p90 ne sont
// pas le minimum et le maximum déguisés — en dessous, l'interpolation retombe sur les deux
// valeurs extrêmes de l'échantillon et le bâton décrirait deux accidents (cf. D6). Les
// armes écartées ne sont pas cachées en silence : leur NOMBRE est rendu par
// `WeaponRangeSummary`, pour que la section puisse dire « N armes sous le seuil ».
//
// ELLE EST DÉFINIE ICI ET NULLE PART AILLEURS (garde-rail : weapon_range_guard_test.go).
const WeaponRangeMinMeasured = 8

// WeaponRangeLevelBandM : demi-largeur, en mètres, de la classe « à niveau » du dénivelé (D4).
//
// D'OÙ VIENT LA VALEUR : la sonde du 2026-09-06 (`.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md`)
// mesure un |dz| médian de 0,4 à 0,9 m sur quatre cartes, et 16 à 40 % des engagements
// au-delà d'UN MÈTRE. Un mètre est donc le seuil qui sépare une marche, un rebord, un étage
// — un avantage de position — du simple décalage de capsule entre deux joueurs debout sur
// le même sol. Les bornes sont FERMÉES sur « à niveau » : exactement +1,0 m et exactement
// -1,0 m sont à niveau, il faut DÉPASSER le mètre pour être au-dessus ou en dessous.
//
// ELLE EST DÉFINIE ICI ET NULLE PART AILLEURS (garde-rail : weapon_range_guard_test.go).
const WeaponRangeLevelBandM = 1.0

// MeasuredKill est UN frag mesuré : l'unité d'entrée de l'agrégat.
type MeasuredKill struct {
	// WeaponKey est la clé d'arme déjà classifiée par l'appelant (jamais un `source_tag`
	// brut : la classification est un adaptateur de titre, elle ne remonte pas ici).
	WeaponKey string
	// Side dit si ce frag est lu du côté du tueur ou de celui de la victime.
	Side Side
	// DistanceM est la distance 3D tueur <-> victime à l'instant retenu, en mètres.
	DistanceM float64
	// DeltaZ est le dénivelé BRUT `killer_z - victim_z`, en mètres — la grandeur PHYSIQUE,
	// sans point de vue. C'est `signedElevation` qui lui donne celui du côté demandé ; le
	// producteur, lui, n'a pas à savoir pour qui il mesure.
	DeltaZ float64
}

// WeaponRange est UNE ligne publiée : un couple (arme, côté) et sa portée mesurée.
type WeaponRange struct {
	WeaponKey string
	Side      Side
	// Measured est l'effectif de la ligne — le dénominateur de tout ce qui suit.
	Measured int
	// P10, Median, P90 sont les percentiles de distance, en mètres (D6).
	P10, Median, P90 float64
	// Above, Level, Below ventilent le dénivelé VU DU CÔTÉ DEMANDÉ : au-dessus de
	// l'adversaire, à niveau, en dessous. Leur somme vaut Measured.
	Above, Level, Below int
}

// WeaponRangeSummary dit ce que l'agrégat a ÉCARTÉ. Il est le SECOND RETOUR de
// `WeaponRangeAggregate` plutôt qu'un champ de sortie : une ligne écartée n'a précisément
// pas de ligne, et un appelant qui ignore ce retour ne peut pas prendre un silence pour un
// zéro (D9 — « les armes sous le seuil ne sont pas cachées en silence »).
type WeaponRangeSummary struct {
	// BelowThreshold est le nombre de couples (arme, côté) sous le seuil.
	BelowThreshold int
	// BelowThresholdBySide ventile ce nombre par côté — la section publie « frags : N ·
	// morts : M », pas un total qui mélangerait les deux lectures.
	BelowThresholdBySide map[Side]int
	// MeasuredBelowThreshold est le nombre de FRAGS que ces couples portaient : il dit
	// combien de mesures réelles la publication laisse de côté, ce que le seul compte
	// d'armes ne dit pas.
	MeasuredBelowThreshold int
}

// weaponSideKey est la clé de groupement — le couple, jamais l'arme seule.
type weaponSideKey struct {
	weapon string
	side   Side
}

// WeaponRangeAggregate agrège des frags mesurés en lignes publiables.
//
// PUR : aucune I/O, aucun SQL, aucune horloge. L'entrée n'est pas mutée (elle n'est ni
// triée ni réordonnée : le tri porte sur des copies des distances).
//
// Les couples sous `minMeasured` ne sont pas publiés et sont comptés dans le résumé rendu.
// Le tri de sortie est DÉTERMINISTE — médiane croissante, puis clé d'arme, puis côté : le
// graphe se lit alors comme un continuum du contact à la longue portée (D6), et deux appels
// sur les mêmes données rendent le même ordre, y compris à médianes égales.
func WeaponRangeAggregate(kills []MeasuredKill, minMeasured int) ([]WeaponRange, WeaponRangeSummary) {
	sum := WeaponRangeSummary{BelowThresholdBySide: map[Side]int{}}
	if minMeasured < 1 {
		// Une portée « mesurée » sur zéro frag n'existe pas : un seuil nul ou négatif
		// publierait des lignes vides. On le remonte à 1 plutôt que de rendre un p10 de rien.
		minMeasured = 1
	}
	groups := map[weaponSideKey][]MeasuredKill{}
	order := make([]weaponSideKey, 0, len(kills))
	for _, k := range kills {
		g := weaponSideKey{weapon: k.WeaponKey, side: k.Side}
		if _, seen := groups[g]; !seen {
			order = append(order, g)
		}
		groups[g] = append(groups[g], k)
	}
	out := make([]WeaponRange, 0, len(order))
	for _, g := range order {
		grp := groups[g]
		if len(grp) < minMeasured {
			sum.BelowThreshold++
			sum.BelowThresholdBySide[g.side]++
			sum.MeasuredBelowThreshold += len(grp)
			continue
		}
		out = append(out, weaponRangeOf(g, grp))
	}
	sortWeaponRanges(out)
	return out, sum
}

// weaponRangeOf calcule la ligne d'un couple (arme, côté) non vide.
func weaponRangeOf(g weaponSideKey, grp []MeasuredKill) WeaponRange {
	wr := WeaponRange{WeaponKey: g.weapon, Side: g.side, Measured: len(grp)}
	dist := make([]float64, len(grp))
	for i, k := range grp {
		dist[i] = k.DistanceM
		switch dz := signedElevation(k); {
		case dz > WeaponRangeLevelBandM:
			wr.Above++
		case dz < -WeaponRangeLevelBandM:
			wr.Below++
		default:
			wr.Level++
		}
	}
	sort.Float64s(dist)
	wr.P10 = percentileLinear(dist, 10)
	wr.Median = percentileLinear(dist, 50)
	wr.P90 = percentileLinear(dist, 90)
	return wr
}

// signedElevation ramène le dénivelé AU POINT DE VUE DU CÔTÉ DEMANDÉ, en mètres.
//
// C'EST LE POINT LE PLUS FACILE À INVERSER DE CE FICHIER, donc il est écrit une seule fois
// et testé aux deux bornes. `MeasuredKill.DeltaZ` porte la grandeur physique
// `killer_z - victim_z`, sans point de vue. Le produit, lui, répond toujours à la même
// question : « MOI, étais-je au-dessus ou en dessous de l'autre ? »
//
//	côté tueur   — je SUIS le tueur : dz = killer_z - victim_z, tel quel.
//	côté victime — je SUIS la victime : dz = victim_z - killer_z, donc l'OPPOSÉ.
//
// Conséquence produit, et elle est voulue : un frag « d'en haut » et la mort d'en face vue
// par la victime sont classés « d'en bas ». Sans cette inversion, « où je meurs, d'en haut
// ou d'en bas » répondrait la position du TUEUR — l'inverse de la question posée.
func signedElevation(k MeasuredKill) float64 {
	if k.Side == SideVictim {
		return -k.DeltaZ
	}
	return k.DeltaZ
}

// sortWeaponRanges impose l'ordre de sortie : médiane croissante, puis arme, puis côté.
func sortWeaponRanges(rs []WeaponRange) {
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Median != rs[j].Median {
			return rs[i].Median < rs[j].Median
		}
		if rs[i].WeaponKey != rs[j].WeaponKey {
			return rs[i].WeaponKey < rs[j].WeaponKey
		}
		return rs[i].Side < rs[j].Side
	})
}

// percentileLinear rend le percentile p (0..100) d'une série TRIÉE, par INTERPOLATION
// LINÉAIRE entre les deux rangs encadrants (la définition « type 7 », celle de numpy et de
// DuckDB `quantile_cont`).
//
// POURQUOI L'INTERPOLATION ET PAS LE RANG LE PLUS PROCHE : sur les effectifs de ce chantier
// (8 à quelques centaines de frags), le rang le plus proche fait sauter le p10 d'une valeur
// entière de l'échantillon à chaque frag ajouté ; le bâton du graphe tressauterait d'un
// filtre de période à l'autre sans que la portée ait bougé.
//
// UNE SEULE ÉCRITURE DANS CE PAQUET. Le paquet `analysis` porte déjà `MedianFloat`
// (squad_session_window.go), qui est la médiane classique et NON un percentile paramétré ;
// les deux coïncident en p=50 et `MedianFloat` reste l'appel canonique pour une médiane
// seule. Une série VIDE rend 0 — jamais NaN : un zéro se lit, un NaN se propage.
func percentileLinear(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 || p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[n-1]
	}
	pos := p / 100 * float64(n-1)
	lo := int(pos)
	if lo >= n-1 {
		return sorted[n-1]
	}
	return sorted[lo] + (pos-float64(lo))*(sorted[lo+1]-sorted[lo])
}
