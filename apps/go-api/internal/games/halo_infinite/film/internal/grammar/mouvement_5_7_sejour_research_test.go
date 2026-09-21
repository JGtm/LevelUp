//go:build research

package grammar

// mouvement_5_7_sejour_research_test.go — LE TEST QUI NE DEPEND PAS DE L APPARIEMENT : LE SEJOUR
// ET LES TRANSITIONS DES QUATRE TAGS D `i55` (lot 5.7).
//
// # POURQUOI UN SECOND TEST
//
// Le test de la montee (`mouvement_5_7_posture_research_test.go`) apparie une lecture d `i55` a
// une lecture d `i1` de la MEME vie dans les deux ticks. Mesure faite : il n apparie que 15 a
// 29 % des lectures, parce qu `i1` ne voyage que quand la vitesse CHANGE assez — la fenetre est
// donc vide la plupart du temps, et un taux calcule sur un cinquieme des occurrences ne tranche
// rien.
//
// Ce fichier pose deux tests qui n ont PAS ce defaut.
//
//	(1) LE SEJOUR. Un etat AERIEN est court par nature : un saut de Spartan dure quelques
//	    dixiemes de seconde. Un etat AU SOL est long. Un etat VEHICULE est long ET rare par vie.
//	    Le sejour d un tag = le temps jusqu a la lecture d `i55` SUIVANTE de la meme vie.
//	(2) LES TRANSITIONS. Un etat aerien s inscrit entre deux etats au sol : sa matrice de
//	    transition doit etre dominee par « X -> aerien -> X ». Un etat vehicule, lui, ne
//	    s intercale pas.
//
// Et la vitesse revient, mais en VALEUR TENUE : dans un flux de deltas un etat vaut jusqu a la
// prochaine transmission, donc la vitesse d une vie a l instant `t` est celle de sa DERNIERE
// lecture d `i1` a ou avant `t`. Cela donne une valeur a CHAQUE lecture d `i55`, sans fenetre.
//
// Rejouable : memes variables que `TestMouvement57Posture`.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m57Sejour est UN sejour : un tag, sa duree jusqu a la lecture suivante de la meme vie, et la
// vitesse TENUE au moment de la lecture.
type m57Sejour struct {
	tag      uint64
	suivant  uint64
	dureeUS  uint64
	aSuivant bool
	vz, sol  float64
	aVit     bool
}

// m57Sejours construit les sejours par vie, dans l ordre du temps.
func m57Sejours(rec *m57Rec) []m57Sejour {
	parSlot := map[uint32][]m57Post{}
	for _, p := range rec.post {
		parSlot[p.slot] = append(parSlot[p.slot], p)
	}
	vits := m57ParSlot(rec.vit)
	var out []m57Sejour
	for slot, ps := range parSlot {
		sort.Slice(ps, func(i, j int) bool { return ps[i].ts < ps[j].ts })
		for i, p := range ps {
			s := m57Sejour{tag: p.tag}
			if i+1 < len(ps) {
				s.suivant, s.dureeUS, s.aSuivant = ps[i+1].tag, ps[i+1].ts-p.ts, true
			}
			if v, ok := m57VitTenue(vits[slot], p.ts); ok {
				s.vz, s.sol, s.aVit = v.vz, v.sol, true
			}
			out = append(out, s)
		}
	}
	return out
}

// m57VitTenue rend la DERNIERE lecture d `i1` a ou avant `ts` — la valeur tenue du flux de
// deltas. Les lectures sont supposees triees (m57ParSlot le garantit).
func m57VitTenue(vits []m57Vit, ts uint64) (m57Vit, bool) {
	idx := -1
	for i, v := range vits {
		if v.ts > ts {
			break
		}
		idx = i
	}
	if idx < 0 {
		return m57Vit{}, false
	}
	return vits[idx], true
}

// TestMouvement57Sejour publie les deux tests et la vitesse tenue par tag.
func TestMouvement57Sejour(t *testing.T) {
	rec, ok := m57Passe(t)
	if !ok {
		return
	}
	sej := m57Sejours(rec)
	m57RendreSejour(t, sej)
	m57RendreTransitions(t, sej)
	m57RendreVitesseTenue(t, sej)
}

// m57RendreSejour publie, par tag, la mediane et les deciles du sejour.
func m57RendreSejour(t *testing.T, sej []m57Sejour) {
	t.Helper()
	t.Logf("SEJOUR PAR TAG (temps jusqu a la lecture d `i55` suivante de la MEME vie) — un etat " +
		"AERIEN doit etre court (quelques dixiemes de seconde), un etat AU SOL long :")
	for tag := uint64(0); tag < 4; tag++ {
		var d []float64
		for _, s := range sej {
			if s.tag == tag && s.aSuivant {
				d = append(d, float64(s.dureeUS)/1e6)
			}
		}
		if len(d) == 0 {
			t.Logf("  tag %d : aucun sejour borne", tag)
			continue
		}
		sort.Float64s(d)
		t.Logf("  tag %d : %d sejours · p10 %.3f s · mediane %.3f s · p90 %.3f s · max %.1f s",
			tag, len(d), m57Quantile(d, 0.10), m57Quantile(d, 0.50), m57Quantile(d, 0.90),
			d[len(d)-1])
	}
}

// m57RendreTransitions publie la matrice des transitions de tag, par vie.
func m57RendreTransitions(t *testing.T, sej []m57Sejour) {
	t.Helper()
	m := map[[2]uint64]int{}
	depart := map[uint64]int{}
	for _, s := range sej {
		if !s.aSuivant {
			continue
		}
		m[[2]uint64{s.tag, s.suivant}]++
		depart[s.tag]++
	}
	t.Logf("MATRICE DES TRANSITIONS (de -> vers, par vie) — un etat AERIEN s inscrit ENTRE deux " +
		"etats au sol, donc sa ligne et sa colonne sont dominees par le meme tag :")
	for de := uint64(0); de < 4; de++ {
		var parts []string
		for vers := uint64(0); vers < 4; vers++ {
			n := m[[2]uint64{de, vers}]
			parts = append(parts, fmt.Sprintf("-> %d : %d (%.1f %%)",
				vers, n, m533bPart(n, depart[de])))
		}
		t.Logf("  tag %d (%d sorties) : %s", de, depart[de], strings.Join(parts, " · "))
	}
}

// m57RendreVitesseTenue publie, par tag, la vitesse VERTICALE et au SOL tenue a l instant de la
// lecture. Un etat aerien doit porter une vz non nulle bien plus souvent que les autres.
func m57RendreVitesseTenue(t *testing.T, sej []m57Sejour) {
	t.Helper()
	t.Logf("VITESSE TENUE A L INSTANT DE LA LECTURE (derniere lecture d `i1` de la meme vie a ou " +
		"avant) :")
	for tag := uint64(0); tag < 4; tag++ {
		var vz, sol []float64
		var monte int
		for _, s := range sej {
			if s.tag != tag || !s.aVit {
				continue
			}
			vz = append(vz, s.vz)
			sol = append(sol, s.sol)
			if s.vz >= m57SeuilMontee {
				monte++
			}
		}
		if len(vz) == 0 {
			t.Logf("  tag %d : aucune vitesse tenue", tag)
			continue
		}
		sort.Float64s(vz)
		sort.Float64s(sol)
		t.Logf("  tag %d : %d lectures · vz mediane %+.3f m/s (p10 %+.3f · p90 %+.3f) · "+
			"vz >= %.1f sur %d (%.1f %%) · sol median %.3f m/s",
			tag, len(vz), m57Quantile(vz, 0.50), m57Quantile(vz, 0.10),
			m57Quantile(vz, 0.90), m57SeuilMontee, monte, m533bPart(monte, len(vz)),
			m57Quantile(sol, 0.50))
	}
}

// m57Quantile rend le quantile d une tranche DEJA TRIEE.
func m57Quantile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	i := int(q * float64(len(xs)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(xs) {
		i = len(xs) - 1
	}
	return xs[i]
}
