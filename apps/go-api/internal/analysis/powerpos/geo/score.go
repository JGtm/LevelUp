package geo

// score.go — NORMALISATION PAR CARTE, POIDS, SCORE.
//
// SCORE = w1*H + w2*V - w3*E + w4*R + w5*M, sur des variables NORMALISEES PAR CARTE.
//
// LA NORMALISATION EST UN MIN-MAX ROBUSTE (p5..p95, borne a [0, 1]) et non un rang. Un rang
// etalerait une variable plate sur tout [0, 1] et lui donnerait, sur une carte ou elle ne
// discrimine rien, le meme poids que sur une carte ou elle discrimine tout. Le min-max
// robuste garde la forme de la distribution et ne fait qu'en couper les queues.
//
// R ET M SONT DES PROXIMITES AVANT NORMALISATION : 1 - min(d, D) / D, avec D la portee de
// reference. C'est ce qui donne a la formule son signe (cf. doc.go).

import (
	"math"
	"sort"
)

// Reglage porte les poids et les seuils de selection. Il se fige sur les trois cartes de
// calibrage, sans regarder l'oracle.
type Reglage struct {
	PoidsH, PoidsV, PoidsE, PoidsR, PoidsM float64
	// PorteeRessourceM : distance de deplacement a laquelle R vaut 0.
	PorteeRessourceM float64
	// QuantileSeuil : quantile du score au-dessus duquel un noeud est retenu.
	QuantileSeuil float64
	// TailleMiniComposante : nombre minimal de noeuds d'une position.
	TailleMiniComposante int
	// MaxPositions : nombre maximal de positions par carte.
	MaxPositions int
	// RayonMaximumLocalM : rayon de deplacement dans lequel un noeud doit dominer pour etre
	// un maximum local.
	RayonMaximumLocalM float64
	// RayonPositionM : rayon de deplacement, autour du maximum, dans lequel une position
	// grandit. C'est ce qui separe une position d'une salle.
	RayonPositionM float64
}

// ReglageGeoV1 rend le reglage FIGE le 2026-09-20 sur les distributions de Recharge,
// Aquarius et Streets, SANS regarder l'oracle (cf. GEOMETRIE_2026-09-20.md, section
// « Reglage geometrique fige »). Ce qui a decide des poids, en une ligne chacun :
//   - H est la variable la plus INDEPENDANTE des quatre autres (|r| <= 0,33 sur les trois
//     cartes) et la premiere que citent les guides : le plus gros poids ;
//   - V et E portent presque la meme information (r = 0,82 / 0,85 / 0,89) : leur difference
//     ponderee est ce qui reste, et elle doit rester petite — V un peu au-dessus de E, pour
//     qu'un lieu qui voit beaucoup et n'est vu que d'un cote ressorte sans que l'ouverture
//     brute ne l'emporte ;
//   - R est independant de H (r <= 0,14) et second dans les guides (l'arme, l'objectif) ;
//   - M est anti-correle a V et E (r = -0,5 a -0,7 : un couvert proche, c'est moins de vue) :
//     il equilibre la paire V/E, pas plus.
func ReglageGeoV1() Reglage {
	return Reglage{
		PoidsH: 0.30, PoidsV: 0.20, PoidsE: 0.15, PoidsR: 0.20, PoidsM: 0.15,
		PorteeRessourceM:     20.0,
		QuantileSeuil:        0.90,
		TailleMiniComposante: 12,
		MaxPositions:         8,
		RayonMaximumLocalM:   3.0,
		RayonPositionM:       4.0,
	}
}

// Normalise ramene une serie dans [0, 1] par min-max robuste (p5..p95). Une serie plate
// rend 0,5 partout : elle ne departage rien, et ne doit pas faire semblant.
func Normalise(vals []float64) []float64 {
	out := make([]float64, len(vals))
	finis := make([]float64, 0, len(vals))
	for _, v := range vals {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			finis = append(finis, v)
		}
	}
	if len(finis) == 0 {
		return out
	}
	sort.Float64s(finis)
	lo, hi := quantile(finis, 0.05), quantile(finis, 0.95)
	for i, v := range vals {
		switch {
		case math.IsNaN(v) || math.IsInf(v, 0):
			out[i] = 0
		case hi <= lo:
			out[i] = 0.5
		default:
			out[i] = math.Max(0, math.Min(1, (v-lo)/(hi-lo)))
		}
	}
	return out
}

// quantile lit un quantile d'une serie TRIEE.
func quantile(tries []float64, q float64) float64 {
	if len(tries) == 0 {
		return math.NaN()
	}
	pos := q * float64(len(tries)-1)
	i := int(math.Floor(pos))
	if i+1 >= len(tries) {
		return tries[len(tries)-1]
	}
	f := pos - float64(i)
	return tries[i]*(1-f) + tries[i+1]*f
}

// Proximite rend 1 - min(d, portee) / portee ; 0 pour une distance infinie.
func Proximite(d, portee float64) float64 {
	if math.IsInf(d, 1) || math.IsNaN(d) || portee <= 0 {
		return 0
	}
	return 1 - math.Min(d, portee)/portee
}

// Score remplit `Norm` et `Score` de chaque noeud a partir de `Brut`, avec le reglage. R
// est recalcule depuis les distances avec la portee du reglage : c'est un poids, il vit ici.
func Score(noeuds []NoeudMesure, r Reglage) {
	n := len(noeuds)
	h, v, e, rr, m := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range noeuds {
		noeuds[i].Brut.R = Proximite(math.Min(noeuds[i].DArmeForte, noeuds[i].DObjectif), r.PorteeRessourceM)
		h[i], v[i], e[i], rr[i], m[i] = noeuds[i].Brut.H, noeuds[i].Brut.V, noeuds[i].Brut.E, noeuds[i].Brut.R, noeuds[i].Brut.M
	}
	h, v, e, rr, m = Normalise(h), Normalise(v), Normalise(e), Normalise(rr), Normalise(m)
	for i := range noeuds {
		noeuds[i].Norm = Variables{H: h[i], V: v[i], E: e[i], R: rr[i], M: m[i]}
		noeuds[i].Score = r.PoidsH*h[i] + r.PoidsV*v[i] - r.PoidsE*e[i] + r.PoidsR*rr[i] + r.PoidsM*m[i]
	}
}

// Correlation rend le coefficient de Pearson de deux series de meme taille (NaN si l'une
// est plate). Sert au rapport de calibrage : deux variables tres correlees ne portent pas
// deux informations.
func Correlation(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return math.NaN()
	}
	var ma, mb float64
	for i := range a {
		ma += a[i]
		mb += b[i]
	}
	ma, mb = ma/float64(len(a)), mb/float64(len(b))
	var sab, saa, sbb float64
	for i := range a {
		da, db := a[i]-ma, b[i]-mb
		sab += da * db
		saa += da * da
		sbb += db * db
	}
	if saa == 0 || sbb == 0 {
		return math.NaN()
	}
	return sab / math.Sqrt(saa*sbb)
}
