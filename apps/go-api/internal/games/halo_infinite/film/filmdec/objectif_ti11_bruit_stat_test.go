package filmdec

// objectif_ti11_bruit_stat_test.go — LA MACHINERIE STATISTIQUE DU LOT « BRUIT » : une epreuve,
// deux temoins, et rien qui se choisisse apres coup.
//
// Le CRITERE est ecrit dans l'en-tete d'`objectif_ti11_bruit_test.go` ; ce fichier ne fait que
// le calculer. Trois regles le gouvernent, et elles valent pour toutes les epreuves :
//
//	UNE SEULE FENETRE      [5 s, 60 s[ avant l'explosion, figee avant la mesure. Aucun balayage
//	                       de fenetres : chercher la meilleure des vingt en trouverait toujours
//	                       une, et ce serait le resultat du chercheur, pas du film.
//	UN TEMOIN QUI GARDE    le temoin B (decalage circulaire) preserve l'ordre et les amas des
//	                       lectures ; le temoin A (uniforme) ne preserve que leur nombre. Les
//	                       deux sont publies, le B fait foi.
//	UNE P-VALEUR MAJOREE   `(sup + 1) / (reps + 1)` — un tirage fini ne peut pas produire zero.
//
// HORS LIGNE, en memoire, sur des bilans deja digeres : aucun film n'est relu ici.

import (
	"math"
	"math/rand"
	"sort"
)

// btRepetitions est le nombre de tirages temoins par epreuve, dans le cas nominal.
const btRepetitions = 999

// btPlafondTirages borne le cout total d'une epreuve (lectures x repetitions). Au-dela, le
// nombre de repetitions est REDUIT, et la reduction est annoncee dans le journal — la machine a
// connu quatre sinistres memoire, et une epreuve qui diverge se garde, elle ne s'espere pas.
const btPlafondTirages = 10_000_000

// btMinRepetitions est le plancher : sous cent tirages, une p-valeur empirique ne veut rien dire.
const btMinRepetitions = 99

// btMesure est ce qu'une passe rend : le compte dans la fenetre, le nombre d'explosions en
// exces, l'histogramme des delais, et les lectures sans explosion apres elles.
type btMesure struct {
	fenetre int
	exces   int
	histo   [btSeaux]int
	sans    int
}

// btEpreuve designe UN champ dans UNE voie sur UN corpus, eventuellement restreint a une valeur.
type btEpreuve struct {
	bs     []*btFilmBilan
	voie   int
	champ  int
	filtre func(uint64) bool
	reps   int
}

// btNouvelleEpreuve regle le nombre de repetitions sur la taille de l'epreuve.
func btNouvelleEpreuve(bs []*btFilmBilan, voie, champ int, filtre func(uint64) bool) btEpreuve {
	e := btEpreuve{bs: bs, voie: voie, champ: champ, filtre: filtre, reps: btRepetitions}
	n := e.tirages()
	if n > 0 && n*e.reps > btPlafondTirages {
		e.reps = btPlafondTirages / n
	}
	if e.reps < btMinRepetitions {
		e.reps = btMinRepetitions
	}
	return e
}

// tirages rend le nombre total de lectures retenues par l'epreuve.
func (e btEpreuve) tirages() int {
	n := 0
	for _, b := range e.bs {
		n += len(e.suite(b))
	}
	return n
}

// suite rend les instants d'un film pour cette epreuve, dans l'ordre du temps.
func (e btEpreuve) suite(b *btFilmBilan) []int32 {
	v := b.voies[e.voie][e.champ]
	out := make([]int32, 0, len(v.ech))
	for _, x := range v.ech {
		if e.filtre != nil && !e.filtre(x.v) {
			continue
		}
		out = append(out, x.tMS)
	}
	return out
}

// prep precalcule les suites d'instants, UNE FOIS par epreuve : elles sont relues des centaines
// de fois par les temoins, et les refiltrer a chaque repetition ne changerait rien au resultat
// tout en multipliant le cout par le nombre de tirages.
func (e btEpreuve) prep() [][]int32 {
	out := make([][]int32, len(e.bs))
	for i, b := range e.bs {
		out[i] = e.suite(b)
	}
	return out
}

// mesurer applique la mesure au corpus. `gen`, s'il n'est pas nil, remplace les instants reels
// par ceux d'un temoin (meme film, meme nombre).
func (e btEpreuve) mesurer(pre [][]int32, gen func(*btFilmBilan, []int32) []int32) btMesure {
	var m btMesure
	for k, b := range e.bs {
		if len(b.expl) == 0 {
			continue
		}
		inst := pre[k]
		if gen != nil {
			inst = gen(b, inst)
		}
		h, sans := btHisto(inst, b.expl)
		f, par := btFenetre(inst, b.expl)
		m.fenetre += f
		m.sans += sans
		m.exces += btExces(par, btAttendus(b, len(inst)))
		for i := range h {
			m.histo[i] += h[i]
		}
	}
	return m
}

// btHisto range les delais « lecture -> explosion SUIVANTE » dans les seaux, et compte les
// lectures qu'aucune explosion ne suit (fin de film, ou manche sans explosion).
func btHisto(inst []int32, expl []int) ([btSeaux]int, int) {
	var h [btSeaux]int
	sans := 0
	for _, t := range inst {
		d, ok := btDelaiSuivant(int(t), expl)
		if !ok {
			sans++
			continue
		}
		h[btSeau(d)]++
	}
	return h, sans
}

// btDelaiSuivant rend le delai jusqu'a la premiere explosion a partir de t. `expl` est TRIEE.
func btDelaiSuivant(t int, expl []int) (int, bool) {
	i := sort.SearchInts(expl, t)
	if i >= len(expl) {
		return 0, false
	}
	return expl[i] - t, true
}

// btSeau rend l'index du seau d'un delai.
func btSeau(d int) int {
	for i := btSeaux - 1; i >= 0; i-- {
		if d >= btBornesMS[i] {
			return i
		}
	}
	return 0
}

// btFenetre compte les lectures tombant dans la fenetre d'AU MOINS une explosion (une lecture
// n'est comptee qu'une fois, meme quand deux explosions sont a moins de 60 s l'une de l'autre)
// et, separement, le compte PAR explosion.
func btFenetre(inst []int32, expl []int) (int, []int) {
	par := make([]int, len(expl))
	total := 0
	for _, t := range inst {
		dedans := false
		for i, e := range expl {
			d := e - int(t)
			if d >= btFenetreBasMS && d < btFenetreHautMS {
				par[i]++
				dedans = true
			}
		}
		if dedans {
			total++
		}
	}
	return total, par
}

// btAttendus rend, par explosion, le nombre de lectures attendu si la densite etait UNIFORME sur
// le support du film. La fenetre est rognee aux bornes du film : une explosion survenue dans la
// premiere minute n'a pas 55 secondes de fenetre derriere elle.
func btAttendus(b *btFilmBilan, n int) []float64 {
	out := make([]float64, len(b.expl))
	span := float64(b.finMS - b.debutMS)
	if span <= 0 || n == 0 {
		return out
	}
	for i, e := range b.expl {
		bas := math.Max(float64(e-btFenetreHautMS), float64(b.debutMS))
		haut := math.Min(float64(e-btFenetreBasMS), float64(b.finMS))
		if haut > bas {
			out[i] = float64(n) * (haut - bas) / span
		}
	}
	return out
}

// btExces compte les explosions dont la fenetre porte PLUS de lectures que son attendu.
func btExces(par []int, att []float64) int {
	n := 0
	for i := range par {
		if float64(par[i]) > att[i] {
			n++
		}
	}
	return n
}

// btUniforme est le TEMOIN A : n instants uniformes sur le support du film.
func btUniforme(rng *rand.Rand, b *btFilmBilan, n int) []int32 {
	out := make([]int32, n)
	span := int(b.finMS - b.debutMS)
	if span <= 0 {
		for i := range out {
			out[i] = b.debutMS
		}
		return out
	}
	for i := range out {
		out[i] = b.debutMS + int32(rng.Intn(span))
	}
	return out
}

// btDecale est le TEMOIN B : les instants REELS, decales d'un offset aleatoire modulo la duree du
// film. Il preserve exactement la structure d'amas des lectures et ne detruit que leur phase.
func btDecale(rng *rand.Rand, b *btFilmBilan, inst []int32) []int32 {
	out := make([]int32, len(inst))
	span := int(b.finMS - b.debutMS)
	if span <= 0 {
		copy(out, inst)
		return out
	}
	d := rng.Intn(span)
	for i, t := range inst {
		out[i] = b.debutMS + int32((int(t-b.debutMS)+d)%span)
	}
	return out
}

// btSerie accumule une statistique de temoin et sa p-valeur empirique.
type btSerie struct {
	obs          int
	n, sup       int
	somme, carre float64
}

// ajouter range une repetition.
func (s *btSerie) ajouter(v int) {
	s.n++
	s.somme += float64(v)
	s.carre += float64(v) * float64(v)
	if v >= s.obs {
		s.sup++
	}
}

// moyenne rend la moyenne des repetitions.
func (s *btSerie) moyenne() float64 {
	if s.n == 0 {
		return 0
	}
	return s.somme / float64(s.n)
}

// ecart rend l'ecart-type des repetitions.
func (s *btSerie) ecart() float64 {
	if s.n < 2 {
		return 0
	}
	m := s.moyenne()
	v := s.carre/float64(s.n) - m*m
	if v <= 0 {
		return 0
	}
	return math.Sqrt(v)
}

// p rend la p-valeur empirique, majoree de un au numerateur et au denominateur.
func (s *btSerie) p() float64 { return float64(s.sup+1) / float64(s.n+1) }

// enrich rend le facteur d'enrichissement de l'observe sur le temoin.
func (s *btSerie) enrich() float64 {
	m := s.moyenne()
	if m == 0 {
		return math.NaN()
	}
	return float64(s.obs) / m
}

// btResultat porte tout ce qu'une epreuve rend.
type btResultat struct {
	obs        btMesure
	fenA, fenB btSerie
	excA, excB btSerie
	histoB     [btSeaux]float64
	tirages    int
	reps       int
	// explosions est le denominateur de la consistance : le nombre d'explosions (ou de
	// pseudo-explosions, sur un temoin croise) que le corpus de l'epreuve porte.
	explosions int
}

// btEprouver lance l'observation puis les deux temoins.
func btEprouver(e btEpreuve) btResultat {
	pre := e.prep()
	r := btResultat{obs: e.mesurer(pre, nil), reps: e.reps}
	for k, b := range e.bs {
		r.explosions += len(b.expl)
		r.tirages += len(pre[k])
	}
	r.fenA.obs, r.fenB.obs = r.obs.fenetre, r.obs.fenetre
	r.excA.obs, r.excB.obs = r.obs.exces, r.obs.exces
	rng := rand.New(rand.NewSource(btGraine)) //nolint:gosec // temoin de mesure, pas de securite
	for i := 0; i < e.reps; i++ {
		a := e.mesurer(pre, func(b *btFilmBilan, in []int32) []int32 {
			return btUniforme(rng, b, len(in))
		})
		r.fenA.ajouter(a.fenetre)
		r.excA.ajouter(a.exces)
		d := e.mesurer(pre, func(b *btFilmBilan, in []int32) []int32 {
			return btDecale(rng, b, in)
		})
		r.fenB.ajouter(d.fenetre)
		r.excB.ajouter(d.exces)
		for k := range r.histoB {
			r.histoB[k] += float64(d.histo[k])
		}
	}
	for k := range r.histoB {
		r.histoB[k] /= float64(e.reps)
	}
	return r
}

// passe dit si le critere du chantier est rempli, au seuil de p donne (Bonferroni compris).
func (r btResultat) passe(seuil float64) bool {
	return r.obs.fenetre > 0 && r.fenB.enrich() >= btEnrichMin &&
		r.fenB.p() <= seuil && r.excB.p() <= seuil
}
