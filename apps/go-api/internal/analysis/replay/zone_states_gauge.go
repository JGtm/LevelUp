package replay

// zone_states_gauge.go — LA JAUGE EN DIRECT (schema 17) : comment la serie brute du tag 3 devient
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
//	                     en cours, abouties ou non (findZoneRamps, memes seuils que l'appariement).
//	                     Entre deux rampes la jauge n'a rien a montrer, et l'ABSENCE de point est
//	                     la donnee : le client efface l'arc une seconde apres le dernier point.
//	ALLEGEE              dans une rampe, un point n'est publie que si la jauge a bouge d'au moins
//	                     zoneGaugeMinDelta depuis le dernier point publie, OU si une seconde s'est
//	                     ecoulee sans point (zoneGaugeMaxGapMS) : c'est ce qui borne le poids
//	                     (<= +2 % de l'artefact, mesure au journal) sans casser l'escalier cote
//	                     client, qui tient une valeur au plus une seconde. Le premier et le dernier
//	                     point de chaque rampe sont toujours publies : le depart et le sommet sont
//	                     ce que l'oeil lit.
//	MEME ECHELLE         `v` est la part de l'excursion mesuree de la jauge de CETTE zone sur CE
//	                     match — exactement l'echelle de `progress` (cf. zoneGaugeOf) — arrondie a
//	                     trois decimales, et T est STRICTEMENT croissant.
//
// EN KOTH, la meme mecanique : la serie d'une colline est celle des rampes que la grappe a posees
// sur elle (les periodes brutes, avant fusion et extension), sur l'echelle de leur slot.

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
	// laquelle le client tient la derniere valeur avant d'effacer l'arc.
	zoneGaugeMaxGapMS = 1000
	// zoneGaugeMilli est le pas d'arrondi de `v` (trois decimales).
	zoneGaugeMilli = 1000
)

// zoneGaugeWindow est un intervalle de frames [t0, t1] pendant lequel la jauge est EN MOUVEMENT :
// une rampe (owner) ou une periode brute de colline (hill).
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
// `wins`, ramenee a [0, 1] par `scale`. Rend nil quand l'echelle est vide ou plate — une jauge qui
// ne bouge pas n'a pas de serie a montrer, meme regle que progressOf.
func zoneGaugeSeriesOf(ss []zoneSample, scale zoneGauge, wins []zoneGaugeWindow, gap int) []GaugePoint {
	if !scale.seen || scale.high <= scale.low || len(wins) == 0 || len(ss) == 0 {
		return nil
	}
	sort.SliceStable(wins, func(i, j int) bool { return wins[i].t0 < wins[j].t0 })
	var out []GaugePoint
	for _, w := range wins {
		out = appendGaugeWindow(out, ss, scale, w, gap)
	}
	return out
}

// appendGaugeWindow pousse les points d'UNE fenetre : le premier et le dernier toujours, et entre
// les deux ceux qui ont bouge d'au moins zoneGaugeMinDeltaMilli ou qui suivent d'au moins `gap`
// frames le dernier point publie.
func appendGaugeWindow(out []GaugePoint, ss []zoneSample, scale zoneGauge, w zoneGaugeWindow,
	gap int,
) []GaugePoint {
	i := sort.Search(len(ss), func(k int) bool { return ss[k].t >= w.t0 })
	first, lastT, lastM := true, 0, 0
	for ; i < len(ss) && ss[i].t <= w.t1; i++ {
		s := ss[i]
		m := gaugeMilliOf(scale, s.v)
		last := i+1 >= len(ss) || ss[i+1].t > w.t1
		if !first && !last && m-lastM < zoneGaugeMinDeltaMilli && s.t-lastT < gap {
			continue
		}
		out = pushGaugePoint(out, GaugePoint{T: s.t, V: float32(m) / zoneGaugeMilli})
		first, lastT, lastM = false, s.t, m
	}
	return out
}

// gaugeMilliOf ramene un quantum a l'echelle de la zone, en milliemes de [0, 1].
func gaugeMilliOf(scale zoneGauge, q uint64) int {
	p := scale.progressOf(q)
	if p == nil {
		return 0
	}
	return int(math.Round(float64(*p) * zoneGaugeMilli))
}

// pushGaugePoint ajoute un point en gardant T STRICTEMENT croissant : un point sur la MEME frame
// que le precedent le REMPLACE (la valeur que la frame porte a sa fin est celle que l'escalier
// doit tenir), un point ANTERIEUR est ecarte (deux fenetres de slots differents qui se recouvrent
// — theorique, une colline n'a qu'un slot de jauge sur le corpus — ne doivent pas rendre une
// serie qui revient en arriere).
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
