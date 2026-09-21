//go:build research

package grammar

// mouvement_5_7_5_oracles_research_test.go — LES DEUX ORACLES PHYSIQUES DU JEU, POSES COMME
// VERITE TERRAIN (lot 5.7.5).
//
// # LA DOCTRINE DE L UTILISATEUR, QUI FAIT AUTORITE
//
// « Tous les Spartans sautent la meme hauteur (petite variable) et courent a la meme vitesse ; on
// a la velocite et la position en Z, ca permet de controler quand un saut ou un sprint est
// entame. »
//
// Cela renverse la methode des passes precedentes. On ne cherche plus un champ replique en
// esperant qu il ressemble a un saut : ON ETIQUETTE D ABORD LES SAUTS ET LES SPRINTS PAR LA
// PHYSIQUE, puis on demande a chaque champ replique de coincider avec cette etiquette. Un champ
// qui coincide est nomme ; un champ qui ne coincide pas rend un SCORE, jamais un negatif.
//
//	(A) LE SAUT A UNE HAUTEUR. Un episode aerien commence quand la vitesse verticale devient
//	    positive et finit quand elle repasse a zero ou dessous ; la HAUTEUR de la montee est
//	    l integrale de `vz` sur cette phase. Si les Spartans sautent tous pareil, la distribution
//	    de ces hauteurs porte un PIC ETROIT a une valeur H — et les rampes, canons a homme et
//	    chutes sont AILLEURS dans la meme distribution.
//	(B) LE SPRINT A UNE VITESSE. La vitesse au sol d un jeu ou l on marche et court a vitesse
//	    CONSTANTE ne prend pas n importe quelle valeur : elle tient des PLATEAUX. La
//	    distribution des plateaux, ponderee par leur DUREE, doit porter deux bosses — `Vm` la
//	    marche, `Vs` le sprint.
//
// # POURQUOI L INTEGRALE DE `vz` ET PAS LE Z DES POSITIONS
//
// `PositionSample.Vec` n est une coordonnee ABSOLUE que sous un accumulateur de monde, qu aucun
// balayage n installe ; sur le chemin delta c est un DELTA borne. La composante verticale de
// `i1`, elle, est en m/s et son unite est VALIDEE (oracle independant du lot 5.3.5 : deplacement
// de la meme vie contre vitesse decodee, dispersion p90/p10 de 1,7 et 2,3, meme facteur d unite
// sur deux films). L integrale d une vitesse validee est une hauteur ; une somme de deltas de
// position dont l unite n est pas etablie n en est pas une.
//
// UN FLUX DE DELTAS SE LIT EN VALEUR TENUE : `i1` ne voyage que quand la vitesse change, donc la
// vitesse d une vie a l instant `t` est celle de sa derniere lecture a ou avant `t`. C est cette
// fonction en escalier que les deux oracles integrent et segmentent.
//
// # LA POPULATION EST CELLE DE LA PORTE PROPRE (lot 5.7.4)
//
// Aucune lecture d essai : la porte des etats de mouvement est inscrite dans
// `neutraliserEtatsDeMouvement` depuis le 5.7.4, et le controle du § 5.7.4.c dit que ce qu elle
// publie egale EXACTEMENT ce que les records retenus declarent.
//
// Rejouable :
//
//	MOUV57_FILM=<dir> MOUV57_CARTE=<carte> MOUV57_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement575Oracles$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m575Episode est UN episode aerien d une vie : son instant d amorce, sa vitesse verticale
// initiale, la hauteur montee et sa duree.
type m575Episode struct {
	slot    uint32
	t0      uint64
	vz0     float64
	hauteur float64
	dureeS  float64
}

// m575Plateau est UN plateau de vitesse au sol tenu par une vie.
type m575Plateau struct {
	slot   uint32
	t0, t1 uint64
	valeur float64
	dureeS float64
}

// m575FenetreMaxUS borne la duree qu une valeur tenue peut representer. `i1` ne voyage que sur
// changement, donc un silence de dix secondes ne veut pas dire « la meme vitesse pendant dix
// secondes » : il veut dire que la vie n a rien transmis, souvent parce qu elle est morte ou hors
// de la trame localisee. Integrer un tel silence fabriquerait des hauteurs et des plateaux.
const m575FenetreMaxUS = 250000

// m575SeuilAirMS est la vitesse verticale au-dela de laquelle une montee commence. 0,5 m/s ecarte
// le bruit de quantification (le pas du quantum log/exp au voisinage de zero) sans couper un
// saut, dont l amorce est de plusieurs m/s.
const m575SeuilAirMS = 0.5

// m575ToleranceH est la demi-largeur de la fenetre d etiquetage « saut », en fraction de H.
const m575ToleranceH = 0.10

// m575HauteurMiniM : sous cette hauteur, un episode aerien n est pas un saut mais une
// oscillation de marche. Cf. le commentaire de `m575RendreSauts` — la valeur est lue dans la
// mesure, pas choisie.
const m575HauteurMiniM = 0.3

// maxInt rend le plus grand de deux entiers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// m575PlateauMinS / m575PlateauTol : un plateau tient au moins ce temps, a cette tolerance
// relative pres.
const (
	m575PlateauMinS = 0.5
	m575PlateauTol  = 0.05
)

// TestMouvement575Oracles pose les deux verites terrain et les publie.
func TestMouvement575Oracles(t *testing.T) {
	rec, ok := m57PasseRetenue(t)
	if !ok {
		return
	}
	eps := m575Episodes(rec)
	m575RendreSauts(t, eps)
	pls := m575Plateaux(rec)
	m575RendrePlateaux(t, pls, rec)
}

// m575Episodes construit les episodes aeriens de chaque vie, par integration de la vitesse
// verticale TENUE.
func m575Episodes(rec *m57Rec) []m575Episode {
	var out []m575Episode
	for slot, vs := range m57ParSlot(rec.vit) {
		var cur *m575Episode
		for i := 0; i < len(vs); i++ {
			dt := m575DureeTenue(vs, i)
			switch {
			case vs[i].vz >= m575SeuilAirMS && cur == nil:
				cur = &m575Episode{slot: slot, t0: vs[i].ts, vz0: vs[i].vz}
			case vs[i].vz < m575SeuilAirMS && cur != nil:
				out = append(out, *cur)
				cur = nil
				continue
			}
			if cur != nil {
				cur.hauteur += vs[i].vz * dt
				cur.dureeS += dt
			}
		}
		if cur != nil {
			out = append(out, *cur)
		}
	}
	return out
}

// m575DureeTenue rend la duree, en secondes, pendant laquelle la lecture `i` vaut — bornee par
// `m575FenetreMaxUS` (cf. la constante).
func m575DureeTenue(vs []m57Vit, i int) float64 {
	if i+1 >= len(vs) {
		return 0
	}
	d := vs[i+1].ts - vs[i].ts
	if d > m575FenetreMaxUS {
		return 0
	}
	return float64(d) / 1e6
}

// m575RendreSauts publie l histogramme des hauteurs, la valeur H du pic, la part d episodes
// etiquetes et la duree mediane.
func m575RendreSauts(t *testing.T, eps []m575Episode) {
	t.Helper()
	if len(eps) == 0 {
		t.Logf("SAUT : aucun episode aerien")
		return
	}
	casiers := map[int]int{} // casiers de 0,1 m
	hs := make([]float64, 0, len(eps))
	for _, e := range eps {
		hs = append(hs, e.hauteur)
		casiers[int(e.hauteur/0.1)]++
	}
	sort.Float64s(hs)
	// LE PIC SE CHERCHE AU-DESSUS DU BRUIT, ET LE SEUIL EST MESURE, PAS CHOISI : le casier
	// [0,0-0,1[ rassemble 453 des 841 episodes de `bfecd02b`, c est-a-dire les oscillations
	// verticales d une marche (un quantum de `vz` au voisinage de zero integre sur un tick).
	// Un saut de Spartan ne fait pas dix centimetres ; chercher le pic dans ce casier revient a
	// mesurer le pas de quantification. Le seuil ecarte les episodes sous
	// `m575HauteurMiniM` et les compte a part.
	var micro int
	pic, n := -1, 0
	for k, c := range casiers {
		if float64(k)*0.1 < m575HauteurMiniM {
			micro += c
			continue
		}
		if c > n {
			pic, n = k, c
		}
	}
	if pic < 0 {
		t.Logf("SAUT : %d episodes, tous sous %.1f m — aucun pic a chercher", len(eps),
			m575HauteurMiniM)
		return
	}
	h := (float64(pic) + 0.5) * 0.1
	var etiquetes int
	var durees []float64
	for _, e := range eps {
		if e.hauteur >= h*(1-m575ToleranceH) && e.hauteur <= h*(1+m575ToleranceH) {
			etiquetes++
			durees = append(durees, e.dureeS)
		}
	}
	sort.Float64s(durees)
	t.Logf("SAUT — %d episodes aeriens sur %d vies : hauteur mediane %.3f m "+
		"(p10 %.3f · p90 %.3f · max %.2f)", len(eps), m575NbVies(eps),
		m57Quantile(hs, 0.50), m57Quantile(hs, 0.10), m57Quantile(hs, 0.90), hs[len(hs)-1])
	t.Logf("  HISTOGRAMME DES HAUTEURS (casiers de 0,1 m, jusqu a 3 m) : %s",
		strings.Join(m575CasiersH(casiers), " · "))
	var voisinG, voisinD int
	if c, ok := casiers[pic-1]; ok {
		voisinG = c
	}
	if c, ok := casiers[pic+1]; ok {
		voisinD = c
	}
	t.Logf("  PIC AU-DESSUS DE %.1f m : H = %.2f m · %d episodes dans son casier contre %d et "+
		"%d chez ses deux voisins (rapport %.1f) · %d episodes (%.1f %% du total, %.1f %% des "+
		"episodes >= %.1f m) dans +/- %.0f %% de H · duree mediane de ceux-la %.3f s · "+
		"%d micro-episodes sous %.1f m ecartes",
		m575HauteurMiniM, h, n, voisinG, voisinD,
		float64(2*n)/float64(maxInt(1, voisinG+voisinD)), etiquetes,
		m533bPart(etiquetes, len(eps)), m533bPart(etiquetes, len(eps)-micro),
		m575HauteurMiniM, m575ToleranceH*100, m57Quantile(durees, 0.50), micro,
		m575HauteurMiniM)
}

// m575NbVies compte les vies distinctes d une tranche d episodes.
func m575NbVies(eps []m575Episode) int {
	s := map[uint32]bool{}
	for _, e := range eps {
		s[e.slot] = true
	}
	return len(s)
}

// m575CasiersH rend l histogramme des hauteurs jusqu a 3 m.
func m575CasiersH(casiers map[int]int) []string {
	cles := make([]int, 0, len(casiers))
	for k := range casiers {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var out []string
	for _, k := range cles {
		if k > 30 {
			continue
		}
		out = append(out, fmt.Sprintf("[%.1f-%.1f[ %d", float64(k)*0.1, float64(k+1)*0.1,
			casiers[k]))
	}
	return out
}

// m575Plateaux segmente la vitesse au sol TENUE de chaque vie en plateaux.
func m575Plateaux(rec *m57Rec) []m575Plateau {
	var out []m575Plateau
	for slot, vs := range m57ParSlot(rec.vit) {
		i := 0
		for i < len(vs) {
			ref := vs[i].sol
			var duree float64
			var somme float64
			j := i
			for j < len(vs) && m575Proche(vs[j].sol, ref) {
				d := m575DureeTenue(vs, j)
				duree += d
				somme += vs[j].sol * d
				j++
			}
			if j == i {
				j = i + 1
			}
			if duree >= m575PlateauMinS && somme > 0 {
				out = append(out, m575Plateau{slot: slot, t0: vs[i].ts, t1: vs[j-1].ts,
					valeur: somme / duree, dureeS: duree})
			}
			i = j
		}
	}
	return out
}

// m575Proche dit si deux vitesses tiennent le meme plateau.
func m575Proche(a, ref float64) bool {
	if ref <= 0 {
		return a <= 0
	}
	return a >= ref*(1-m575PlateauTol) && a <= ref*(1+m575PlateauTol)
}

// m575RendrePlateaux publie l histogramme des plateaux PONDERE PAR LA DUREE, et nomme les deux
// bosses si elles existent.
func m575RendrePlateaux(t *testing.T, pls []m575Plateau, rec *m57Rec) {
	t.Helper()
	if len(pls) == 0 {
		t.Logf("SPRINT : aucun plateau tenu")
		return
	}
	casiers := map[int]float64{} // duree cumulee par casier de 0,25 m/s
	var totalS float64
	for _, p := range pls {
		casiers[int(p.valeur/0.25)] += p.dureeS
		totalS += p.dureeS
	}
	cles := make([]int, 0, len(casiers))
	for k := range casiers {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	var bosses []int
	for _, k := range cles {
		if k <= 40 {
			parts = append(parts, fmt.Sprintf("[%.2f-%.2f[ %.1f s", float64(k)*0.25,
				float64(k+1)*0.25, casiers[k]))
		}
		if casiers[k] > casiers[k-1] && casiers[k] > casiers[k+1] && casiers[k]*100 > totalS {
			bosses = append(bosses, k)
		}
	}
	t.Logf("SPRINT — %d plateaux tenus (>= %.1f s, tolerance %.0f %%) sur %d vies, %.0f s "+
		"cumulees", len(pls), m575PlateauMinS, m575PlateauTol*100, m575NbViesP(pls), totalS)
	t.Logf("  HISTOGRAMME PONDERE PAR LA DUREE (casiers de 0,25 m/s, jusqu a 10 m/s) : %s",
		strings.Join(parts, " · "))
	var noms []string
	for _, k := range bosses {
		noms = append(noms, fmt.Sprintf("%.2f-%.2f m/s (%.1f s, %.1f %%)", float64(k)*0.25,
			float64(k+1)*0.25, casiers[k], casiers[k]/totalS*100))
	}
	t.Logf("  BOSSES (> 1 %% du temps, strictement au-dessus de leurs deux voisins) : %d — %s",
		len(bosses), strings.Join(noms, " · "))
	m575RapportDesBosses(t, bosses, casiers, totalS)
	_ = rec
}

// m575NbViesP compte les vies distinctes d une tranche de plateaux.
func m575NbViesP(pls []m575Plateau) int {
	s := map[uint32]bool{}
	for _, p := range pls {
		s[p.slot] = true
	}
	return len(s)
}

// m575RapportDesBosses nomme Vm et Vs et rend leur rapport, s il y a au moins deux bosses.
func m575RapportDesBosses(t *testing.T, bosses []int, casiers map[int]float64, totalS float64) {
	t.Helper()
	if len(bosses) < 2 {
		t.Logf("  UNE SEULE BOSSE : la doctrine du jeu en attend DEUX (marche Vm, sprint Vs). "+
			"Sur cette population, la vitesse au sol ne porte pas de second plateau — score, "+
			"pas verdict : %d bosse(s) pour %.0f s cumulees.", len(bosses), totalS)
		return
	}
	// les deux bosses les plus lourdes, par duree
	sort.Slice(bosses, func(i, j int) bool { return casiers[bosses[i]] > casiers[bosses[j]] })
	a, b := bosses[0], bosses[1]
	if a > b {
		a, b = b, a
	}
	vm := (float64(a) + 0.5) * 0.25
	vs := (float64(b) + 0.5) * 0.25
	t.Logf("  Vm = %.2f m/s (%.1f s) · Vs = %.2f m/s (%.1f s) · RAPPORT Vs/Vm = %.2f",
		vm, casiers[a], vs, casiers[b], vs/vm)
}
