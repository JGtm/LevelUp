package replay

// zone_states_gauge.go — LA JAUGE EN DIRECT (schema 18) : comment la serie brute du tag 3 devient
// `ZoneState.gauge`, c'est-a-dire une jauge qui se remplit a l'image, et pas seulement son sommet.
//
// CE QUE v16 MONTRAIT, ET POURQUOI CE N'ETAIT PAS UNE JAUGE. `ZoneSpan.progress` porte le SOMMET
// de la jauge sur l'intervalle de propriete : une valeur par intervalle, souvent 1,0, tenue
// pendant toute la duree de la propriete. Dessinee en arc, elle restait pleine des minutes durant
// et se lisait comme « la zone est en cours de capture » — le contraire de ce qu'elle disait. La
// rampe pas a pas, elle, est dans le film (~40 emissions par montee, un pas de ~1 199 quanta), et
// le lecteur de production la lit deja pour apparier les slots : il ne manquait que de la PUBLIER.
//
// LES TROIS REGLES DE LA SERIE, ecrites AVANT la mesure des temoins (plan lot C-ter volet 3) :
//
//	RIEN HORS RAMPE      la serie ne porte que les montees monotones de la jauge — les captures
//	                     en cours, abouties ou non (findZoneRamps, memes seuils que l'appariement)
//	                     — et le RETOUR A ZERO qui ferme chacune quand le film le porte
//	                     (appendGaugeReset : la jauge ne redescend jamais autrement). Entre deux
//	                     rampes la jauge n'a rien a montrer.
//	ALLEGEE              dans une rampe, un point n'est publie que si la jauge a bouge d'au moins
//	                     zoneGaugeMinDelta depuis le dernier point publie, OU si une seconde s'est
//	                     ecoulee sans point (zoneGaugeMaxGapMS) : c'est ce qui borne le poids
//	                     (<= +2 % de l'artefact, mesure au journal). Le premier et le dernier
//	                     point de chaque rampe sont toujours publies : le depart et le sommet sont
//	                     ce que l'oeil lit. Cote client, l'escalier TIENT la derniere valeur
//	                     jusqu'au point suivant (une jauge figee — zone contestee — reste
//	                     affichee), et efface l'arc une seconde apres le DERNIER point de la serie.
//	MEME ECHELLE         `v` est la fraction de capture sur l'echelle du JEU — 0 = jauge au repos,
//	                     1 = pleine — exactement l'echelle de `progress` (cf. gaugeProgressOf dans
//	                     zone_states.go, et pourquoi ce n'est plus l'excursion du match) — arrondie
//	                     a trois decimales, et T est STRICTEMENT croissant.
//
// EN KOTH, RIEN : la serie n'est publiee que sur les modes a zones SIMULTANEES (Bastion), la ou le
// tag 3 est la vraie rampe de capture (97 % des captures precedees d'une rampe, lot C-bis). Sur une
// colline, le meme tag est un compteur de transfert d'environ une seconde (volet 1 du lot C-ter),
// pas la progression de garde : `buildHillStates` ne pose aucune serie, et le dit.

import (
	"math"
	"sort"
)

const (
	// zoneGaugeMinDeltaMilli est la variation MINIMALE de la jauge entre deux points publies
	// d'une meme rampe, en milliemes de l'echelle [0, 1] : 0,02.
	zoneGaugeMinDeltaMilli = 20
	// zoneGaugeMaxGapMS est la duree au bout de laquelle une rampe re-emet un point meme si la
	// jauge n'a pas bouge de zoneGaugeMinDeltaMilli : une seconde. C'est aussi la duree pendant
	// laquelle le client tient la derniere valeur de la SERIE avant d'effacer l'arc.
	zoneGaugeMaxGapMS = 1000
	// zoneGaugeMilli est le pas d'arrondi de `v` (trois decimales).
	zoneGaugeMilli = 1000
)

// zoneGaugeWindow est un intervalle de frames [t0, t1] pendant lequel la jauge est EN MOUVEMENT :
// une rampe du slot de jauge de la zone (cf. rampWindowsOf).
type zoneGaugeWindow struct {
	t0, t1 int
}

// zoneGaugeGapFrames rend l'ecart maximal entre deux points publies d'une rampe, en frames — au
// moins une.
func zoneGaugeGapFrames(intervalMS int) int {
	if intervalMS <= 0 {
		return 1
	}
	if g := zoneGaugeMaxGapMS / intervalMS; g > 1 {
		return g
	}
	return 1
}

// zoneGaugeSeriesOf rend la serie allegee des emissions `ss` d'un slot de jauge sur les fenetres
// `wins`, ramenee a [0, 1] sur l'echelle du jeu (gaugeProgressOf). Rend nil sans fenetre ni
// emission.
func zoneGaugeSeriesOf(ss []zoneSample, wins []zoneGaugeWindow, gap int) []GaugePoint {
	if len(wins) == 0 || len(ss) == 0 {
		return nil
	}
	sort.SliceStable(wins, func(i, j int) bool { return wins[i].t0 < wins[j].t0 })
	var out []GaugePoint
	for _, w := range wins {
		out = appendGaugeWindow(out, ss, w, gap)
	}
	return out
}

// appendGaugeWindow pousse les points d'UNE fenetre : le premier et le dernier toujours, et entre
// les deux ceux qui ont bouge d'au moins zoneGaugeMinDeltaMilli ou qui suivent d'au moins `gap`
// frames le dernier point publie — puis le RETOUR A ZERO qui ferme la rampe, quand le film le
// porte (cf. appendGaugeReset).
func appendGaugeWindow(out []GaugePoint, ss []zoneSample, w zoneGaugeWindow, gap int) []GaugePoint {
	i := sort.Search(len(ss), func(k int) bool { return ss[k].t >= w.t0 })
	first, lastT, lastM := true, 0, 0
	for ; i < len(ss) && ss[i].t <= w.t1; i++ {
		s := ss[i]
		m := gaugeMilliOf(s.v)
		last := i+1 >= len(ss) || ss[i+1].t > w.t1
		if !first && !last && m-lastM < zoneGaugeMinDeltaMilli && s.t-lastT < gap {
			continue
		}
		out = pushGaugePoint(out, GaugePoint{T: s.t, V: float32(m) / zoneGaugeMilli})
		first, lastT, lastM = false, s.t, m
	}
	return appendGaugeReset(out, ss, i)
}

// appendGaugeReset publie le RETOUR A ZERO qui suit une rampe : la premiere emission apres la
// fenetre, si elle vaut le zero du jeu (ou moins).
//
// POURQUOI CE POINT EST LA FIN DE LA RAMPE, ET PAS UN POINT « HORS RAMPE » (mesure du 2026-08-19,
// `echelle_7344d24f.log`) : la jauge ne REDESCEND JAMAIS pas a pas — sur les trois slots de jauge
// des zones de Bastion, TOUS les pas descendants (18, 18 et 16) sont des retours au zero exact, et
// il n'y a aucun pas nul. Le canal est une marche d'escalier : il monte tant qu'on capture, se TAIT
// tant que la capture est figee (zone contestee : 29 s a 0,92 sur `7344d24f`) ou abandonnee, et
// se remet a zero d'une seule emission — a la capture menee a terme (une frame apres le sommet)
// comme a l'abandon (1,4 s a 11 s apres le dernier pas). Sans ce point, le client ne peut pas
// distinguer « figee » de « finie » ; avec lui, il tient la derniere valeur jusqu'au point suivant
// et efface l'arc quand le film le dit. Un retour a zero deja publie comme DEPART de la rampe
// suivante n'entre pas deux fois : pushGaugePoint le fond sur la meme frame.
func appendGaugeReset(out []GaugePoint, ss []zoneSample, next int) []GaugePoint {
	if next >= len(ss) || ss[next].v > zoneGaugeQuantZero {
		return out
	}
	return pushGaugePoint(out, GaugePoint{T: ss[next].t, V: 0})
}

// gaugeMilliOf ramene un quantum a l'echelle du jeu, en milliemes de [0, 1].
func gaugeMilliOf(q uint64) int {
	return int(math.Round(float64(gaugeProgressOf(q)) * zoneGaugeMilli))
}

// pushGaugePoint ajoute un point en gardant T STRICTEMENT croissant : un point sur la MEME frame
// que le precedent le REMPLACE (la valeur que la frame porte a sa fin est celle que l'escalier
// doit tenir — c'est aussi ce qui fond un retour a zero avec le depart de la rampe suivante), un
// point ANTERIEUR est ecarte (la serie ne revient jamais en arriere).
func pushGaugePoint(out []GaugePoint, p GaugePoint) []GaugePoint {
	if n := len(out); n > 0 && out[n-1].T >= p.T {
		if out[n-1].T == p.T {
			out[n-1] = p
		}
		return out
	}
	return append(out, p)
}

// rampWindowsOf traduit des rampes en fenetres de jauge.
func rampWindowsOf(ramps []zoneRamp) []zoneGaugeWindow {
	out := make([]zoneGaugeWindow, 0, len(ramps))
	for _, r := range ramps {
		out = append(out, zoneGaugeWindow{t0: r.t0, t1: r.tPeak})
	}
	return out
}

// tallyZoneStates compte ce que les etats publient, toutes zones confondues : les intervalles et
// les points de jauge en direct.
func tallyZoneStates(states []ZoneState, cov *ZonesCoverage) {
	for _, s := range states {
		cov.Spans += len(s.Spans)
		cov.GaugePoints += len(s.Gauge)
	}
}
