// Package analysis — weapon_opening_delta.go : DE L'ENTAME AU COUP FATAL, FRAG PAR FRAG.
//
// # CE QUE CE FICHIER MESURE
//
// Un engagement a deux distances : celle du coup fatal, et celle d'un temps-pour-tuer plus
// tôt (proxy d'entame validé le 2026-09-06, écart médian 1,24 m — D5 du plan
// .ai/PLAN_DUELS_PORTEE_2026-09-06.md). L'écart entre les deux dit si le joueur FERME la
// distance pendant l'échange ou s'il la garde.
//
// # POURQUOI PAR FRAG APPARIÉ, ET JAMAIS ENTRE DEUX MÉDIANES
//
// C'est le point qui justifie ce fichier. La couverture des deux mesures N'EST PAS LA MÊME :
// une mort survenue dans les premières secondes d'un match n'a pas d'entame lisible, et le
// filtre « même vie » en écarte d'autres. Soustraire la médiane des entames de la médiane des
// coups fatals reviendrait à comparer DEUX POPULATIONS DIFFÉRENTES d'engagements — le nombre
// obtenu serait plausible, stable, et faux, sans qu'aucun test de cohérence ne le dise. On
// apparie donc par la clé du frag `(MatchID, KillerXUID, TimeMS)`, et on ne mesure que les
// frags qui portent LES DEUX distances.
//
// # LA CLÉ EST UNIQUE À TRAVERS LES DEUX CÔTÉS
//
// Un frag donné n'est jamais lu des deux côtés à la fois pour un même joueur : les frags qu'il
// inflige et ceux qu'il subit sont deux populations disjointes (il n'est pas son propre
// tueur). La clé ne peut donc pas collisionner d'un côté à l'autre. Le côté reste un filtre
// de LECTURE — « où je frague » et « où je meurs » ne se moyennent pas ensemble.
package analysis

import "sort"

// WeaponOpeningStats — ce qu'un côté d'engagement dit de son entame.
//
// Les deux effectifs sont DISTINCTS et tous deux publiés : `MeasuredOpenings` compte les
// entames lues, `Paired` les frags qui portent AUSSI leur distance de coup fatal. Le second
// est le dénominateur du delta ; les confondre ferait présenter un écart calculé sur 40 frags
// comme s'il en décrivait 600.
type WeaponOpeningStats struct {
	// MedianOpeningM est la médiane des distances d'entame, en mètres.
	MedianOpeningM float64
	// MeasuredOpenings est le nombre d'entames mesurées du côté demandé.
	MeasuredOpenings int
	// MedianDeltaM est la médiane, PAR FRAG APPARIÉ, de `distance au coup fatal −
	// distance à l'entame`. NÉGATIF = l'engagement se ferme (le joueur a réduit la
	// distance pendant l'échange) ; positif = elle s'ouvre.
	MedianDeltaM float64
	// ClosingShare est la part des frags appariés dont la distance se FERME
	// (delta < 0), en unité 0..1 (convention API canonique, ADR 0006).
	ClosingShare float64
	// Paired est le nombre de frags portant les deux mesures — le dénominateur de
	// MedianDeltaM et de ClosingShare.
	Paired int
}

// measuredKillKey identifie UN frag. Le côté n'en fait pas partie : cf. l'en-tête.
type measuredKillKey struct {
	matchID    string
	killerXUID string
	timeMS     int64
}

func keyOf(k MeasuredKill) measuredKillKey {
	return measuredKillKey{matchID: k.MatchID, killerXUID: k.KillerXUID, timeMS: k.TimeMS}
}

// WeaponOpeningDelta apparie les entames aux coups fatals du côté demandé.
//
// PUR : aucune I/O, aucune horloge, aucune mutation des entrées. Une entame sans coup fatal
// correspondant (le frag n'est pas dans le scope, ou sa position de mort manque) n'est pas
// appariée : elle compte dans `MeasuredOpenings`, jamais dans `Paired`.
//
// AUCUN SEUIL DE PUBLICATION ICI. `WeaponRangeMinMeasured` gouverne les lignes PAR ARME, où
// un p10/p90 sur trois frags décrirait deux accidents (D9) ; ce bloc-ci ne publie que des
// médianes globales, qu'un effectif faible rend imprécises mais jamais absurdes. C'est à
// l'appelant de ne rien publier quand `MeasuredOpenings` vaut zéro — la section dit alors
// « aucune entame mesurée », JAMAIS un zéro (D5).
func WeaponOpeningDelta(kills, openings []MeasuredKill, side Side) WeaponOpeningStats {
	distanceAuKill := make(map[measuredKillKey]float64, len(kills))
	for _, k := range kills {
		if k.Side == side {
			distanceAuKill[keyOf(k)] = k.DistanceM
		}
	}

	entames := make([]float64, 0, len(openings))
	deltas := make([]float64, 0, len(openings))
	fermetures := 0
	for _, o := range openings {
		if o.Side != side {
			continue
		}
		entames = append(entames, o.DistanceM)
		auKill, apparie := distanceAuKill[keyOf(o)]
		if !apparie {
			continue
		}
		delta := auKill - o.DistanceM
		deltas = append(deltas, delta)
		if delta < 0 {
			fermetures++
		}
	}

	st := WeaponOpeningStats{MeasuredOpenings: len(entames), Paired: len(deltas)}
	if len(entames) > 0 {
		sort.Float64s(entames)
		st.MedianOpeningM = percentileLinear(entames, 50)
	}
	if len(deltas) > 0 {
		sort.Float64s(deltas)
		st.MedianDeltaM = percentileLinear(deltas, 50)
		st.ClosingShare = float64(fermetures) / float64(len(deltas))
	}
	return st
}
