package filmdec

// ti47_annonces_valeurs_test.go — LES RAPPORTS DE VALEUR de l'instrument ti=47 (phase 1, item
// 1.1 du plan `.ai/V7.5/replay2d/PLAN_TI47_ANNONCES_ZONE.md`) : cardinalite, profil de bits,
// structure temporelle, rythme par slot. La confrontation aux oracles est dans
// `ti47_annonces_oracle_test.go`.
//
// L'ORDRE DES TROIS LECTURES N'EST PAS ARBITRAIRE. On regarde d'abord COMBIEN de valeurs
// distinctes (une annonce a un petit alphabet), puis QUELS BITS bougent (un champ de 45 bits dont
// vingt positions ne varient jamais est une structure, pas un scalaire), puis A QUEL RYTHME par
// entite (une annonce tombe quelques dizaines de fois par match ; une valeur repliquee tombe a
// cadence fixe). La troisieme lecture est celle qui a tranche ce lot.

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
)

// ti47JournaliseValeurs publie la cardinalite et la structure des valeurs lues (phase 1).
func ti47JournaliseValeurs(t *testing.T, m *ti47Moisson, width int) {
	t.Helper()
	t.Logf("VALEURS (largeur %d bits · champ analyse [%d:%d[, soit %d bits) : %d records annoncent"+
		" i%d · %d refuses (payload trop court) · %d emissions retenues · %d valeurs distinctes",
		width, m.champLo, m.champHi, m.champHi-m.champLo, m.luesTotal, m.b.iPersonal,
		m.luesRefusee, len(m.emissions), len(m.valeurs))
	if m.valeursDebordees > 0 || m.emissionsDebordees > 0 {
		t.Logf("   PLAFOND ATTEINT : %d valeurs et %d emissions ecartees",
			m.valeursDebordees, m.emissionsDebordees)
	}
	type vc struct {
		v uint64
		n int
	}
	tri := make([]vc, 0, len(m.valeurs))
	for v, n := range m.valeurs {
		tri = append(tri, vc{v, n})
	}
	sort.Slice(tri, func(i, j int) bool {
		if tri[i].n != tri[j].n {
			return tri[i].n > tri[j].n
		}
		return tri[i].v < tri[j].v
	})
	var cumul int
	for i, e := range tri {
		if i >= 16 {
			break
		}
		cumul += e.n
		t.Logf("   %-12d %7d  %5.2f %%", e.v, e.n,
			100*float64(e.n)/float64(maxI(1, len(m.emissions))))
	}
	t.Logf("   les 16 premieres valeurs couvrent %.2f %% des emissions",
		100*float64(cumul)/float64(maxI(1, len(m.emissions))))
	ti47ProfilBits(t, m)
	ti47Deciles(t, m)
	ti47StructureValeurs(t, m)
	ti47ParSlot(t, m)
}

// ti47Deciles publie la distribution de la valeur RAMENEE A SON ECHELLE (fraction de 2^n).
//
// SANS ECHELLE, UN ENTIER DE 25 BITS NE DIT RIEN. Ramene a sa pleine echelle, il se lit : une
// valeur qui vit a 0,999 avec des excursions vers 0,4 est une fraction normalisee, pas un
// identifiant ni une horloge — et c'est une lecture que la donnee porte, pas une interpretation.
func ti47Deciles(t *testing.T, m *ti47Moisson) {
	t.Helper()
	if len(m.emissions) == 0 {
		return
	}
	n := uint(m.champHi - m.champLo)
	plein := float64(uint64(1) << n)
	var h [10]int
	sature := 0
	for _, e := range m.emissions {
		f := float64(e.val) / plein
		h[minI(9, int(f*10))]++
		if f >= 0.999 {
			sature++
		}
	}
	var parts []string
	for k, c := range h {
		if c == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%.1f-%.1f[ %.1f %%", float64(k)/10, float64(k+1)/10,
			100*float64(c)/float64(len(m.emissions))))
	}
	t.Logf("   echelle (valeur / 2^%d) : %s · au plafond (>= 0,999) : %.1f %%",
		n, strings.Join(parts, " · "), 100*float64(sature)/float64(len(m.emissions)))
}

// ti47ParSlot publie, pour chaque slot de la bande, le rythme et la variation de sa valeur.
//
// C'EST CE QUI SEPARE UNE ANNONCE D'UNE REPLICATION. Une annonce tombe quelques dizaines de fois
// par match, a des instants irreguliers ; une valeur repliquee tombe a cadence fixe. Le lot C
// comptait des « annonces » au sens du MASQUE (le composant est annonce dans le masque du
// record), ce qui ne dit rien du rythme : ce tableau le dit.
func ti47ParSlot(t *testing.T, m *ti47Moisson) {
	t.Helper()
	parSlot := map[uint32][]ti47Emission{}
	for _, e := range m.emissions {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	slots := make([]uint32, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return len(parSlot[slots[i]]) > len(parSlot[slots[j]]) })
	t.Logf("PAR SLOT (rythme et variation ; un slot = une entite de l'archetype) :")
	for i, s := range slots {
		if i >= 12 {
			t.Logf("   ... et %d autres slots", len(slots)-12)
			break
		}
		es := parSlot[s]
		sort.SliceStable(es, func(x, y int) bool {
			if es[x].chunk != es[y].chunk {
				return es[x].chunk < es[y].chunk
			}
			return es[x].tMS < es[y].tMS
		})
		var ecarts []int
		var deltas []int64
		distinct := map[uint64]bool{}
		for k, e := range es {
			distinct[e.val] = true
			if k == 0 || e.tMS < 0 || es[k-1].tMS < 0 {
				continue
			}
			if d := e.tMS - es[k-1].tMS; d >= 0 {
				ecarts = append(ecarts, d)
			}
			deltas = append(deltas, absI64(int64(e.val)-int64(es[k-1].val)))
		}
		sort.Ints(ecarts)
		sort.Slice(deltas, func(x, y int) bool { return deltas[x] < deltas[y] })
		med, p90 := ti47Mediane(ecarts), ti47Percentile(ecarts, 90)
		var medDelta, p99Delta int64
		if len(deltas) > 0 {
			medDelta = deltas[len(deltas)/2]
			p99Delta = deltas[minI(len(deltas)-1, 99*len(deltas)/100)]
		}
		lo, hi := es[0].val, es[0].val
		for _, e := range es {
			if e.val < lo {
				lo = e.val
			}
			if e.val > hi {
				hi = e.val
			}
		}
		t.Logf("   slot %-5d %6d emissions · %5d valeurs distinctes · ecart median %4d ms"+
			" (p90 %5d) · |variation| mediane %d (p99 %d) · plage [%d..%d] · derive %d",
			s, len(es), len(distinct), med, p90, medDelta, p99Delta, lo, hi,
			int64(es[len(es)-1].val)-int64(es[0].val))
	}
}

func absI64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ti47Mediane rend la mediane d'une suite DEJA TRIEE, ou -1 si elle est vide.
func ti47Mediane(v []int) int { return ti47Percentile(v, 50) }

// ti47Percentile rend le percentile p d'une suite DEJA TRIEE, ou -1 si elle est vide.
func ti47Percentile(v []int, p int) int {
	if len(v) == 0 {
		return -1
	}
	return v[minI(len(v)-1, p*len(v)/100)]
}

// ti47ProfilBits publie, position par position, la part d'emissions ou le bit vaut 1.
//
// C'EST CE QUI DIT SI LA LARGEUR EST UN SCALAIRE OU UNE STRUCTURE. Un champ de 45 bits dont dix
// positions ne bougent jamais n'est pas un entier de 45 bits : c'est un en-tete constant suivi
// d'une valeur, et le decoupage se lit directement dans ce profil — sans binaire et sans
// hypothese.
func ti47ProfilBits(t *testing.T, m *ti47Moisson) {
	t.Helper()
	n := len(m.emissions)
	if n == 0 || m.largeur == 0 {
		return
	}
	var ligne, constants strings.Builder
	nbConst := 0
	for k := 0; k < m.largeur; k++ {
		p := float64(m.bits[k]) / float64(n)
		switch {
		case p == 0:
			ligne.WriteByte('.')
			constants.WriteByte('0')
			nbConst++
		case p == 1:
			ligne.WriteByte('#')
			constants.WriteByte('1')
			nbConst++
		default:
			ligne.WriteByte("0123456789"[minI(9, int(p*10))])
			constants.WriteByte('-')
		}
	}
	t.Logf("   profil des bits (position 0 = premier bit apres le masque ; '.' = toujours 0," +
		" '#' = toujours 1, chiffre = dizaine de la part a 1) :")
	t.Logf("      %s", ligne.String())
	t.Logf("      constants : %s  (%d bits sur %d ne varient JAMAIS)",
		constants.String(), nbConst, m.largeur)
}

// ti47StructureValeurs distingue une ENUMERATION d'un continuum : une horloge ou une rampe suit
// le temps, un identifiant de message ne le suit pas.
func ti47StructureValeurs(t *testing.T, m *ti47Moisson) {
	t.Helper()
	if len(m.emissions) < 8 {
		return
	}
	es := append([]ti47Emission(nil), m.emissions...)
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].chunk != es[j].chunk {
			return es[i].chunk < es[j].chunk
		}
		return es[i].tMS < es[j].tMS
	})
	croissant := 0
	for i := 1; i < len(es); i++ {
		if es[i].val > es[i-1].val {
			croissant++
		}
	}
	var sx, sy, sxy, sxx, syy float64
	n := float64(len(es))
	for _, e := range es {
		x, y := float64(e.tMS), float64(e.val)
		sx, sy, sxy, sxx, syy = sx+x, sy+y, sxy+x*y, sxx+x*x, syy+y*y
	}
	den := (n*sxx - sx*sx) * (n*syy - sy*sy)
	r := 0.0
	if den > 0 {
		r = (n*sxy - sx*sy) / math.Sqrt(den)
	}
	t.Logf("   structure : %.1f %% de transitions croissantes · correlation valeur/instant"+
		" r = %.3f · %d valeurs distinctes pour %d emissions",
		100*float64(croissant)/(n-1), r, len(m.valeurs), len(es))
	switch {
	case len(m.valeurs) <= 64:
		t.Logf("   -> ENUMERATION : moins de 64 valeurs distinctes.")
	case r >= 0.90:
		t.Logf("   -> la valeur SUIT LE TEMPS : horloge ou compteur, pas un identifiant.")
	default:
		t.Logf("   -> ni enumeration courte ni horloge : continuum ou treillis (comme i1).")
	}
}
