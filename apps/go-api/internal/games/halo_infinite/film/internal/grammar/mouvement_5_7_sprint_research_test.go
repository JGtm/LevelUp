//go:build research

package grammar

// mouvement_5_7_sprint_research_test.go — LE SPRINT ET LE SAUT SUR LA POPULATION RETENUE
// (lot 5.7.4).
//
// # POURQUOI CE FICHIER EXISTE, ET CE QU IL CORRIGE
//
// Les deux tests du § 5.7.2 (montee de vz, sejour) portaient sur la porte POLLUEE : de 14 a 152
// lectures d essai pour une lecture retenue. Depuis que la porte s inscrit dans
// `neutraliserEtatsDeMouvement`, la population est celle des records RETENUS — et les deux
// questions se reposent proprement :
//
//	(a) LE SPRINT. La vitesse AU SOL du bipede a-t-elle DEUX BOSSES ? Un jeu ou l on marche a
//	    ~2,3 m/s et ou l on sprinte plus vite doit montrer un second mode. Le lot 5.3.5 disait
//	    « un seul mode a 2-3 m/s, 0,08 % au-dela de 4 m/s » ; `i1` etait le composant le MOINS
//	    pollue (x 1,27), donc son verdict tenait deja — ici il est repose sur la population
//	    exacte, ET PAR SLOT : une vie qui sprinte se voit dans SA distribution, pas dans la
//	    moyenne du film.
//	(b) LE SAUT. Les quatre tags d `i55` contre la montee de la composante verticale, sur les
//	    seules lectures retenues.
//
// L instrument ne branche la porte qu a travers la marche : aucune relecture, aucun second
// decodage. Les records retenus publient, les essais non.
//
// Rejouable : memes variables que `TestMouvement57Posture`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m57SprintBinMS est la largeur d un casier de l histogramme de vitesse au sol, en m/s.
const m57SprintBinMS = 0.5

// m57SprintMinLectures est le nombre de lectures d `i1` sous lequel une vie ne porte pas de
// distribution : en dessous, un p95 est un maximum deguise.
const m57SprintMinLectures = 200

// TestMouvement57Sprint repose les deux questions sur la population retenue.
func TestMouvement57Sprint(t *testing.T) {
	rec, ok := m57PasseRetenue(t)
	if !ok {
		return
	}
	m57SprintDistribution(t, rec)
	m57SprintParSlot(t, rec)
	m57SautParTag(t, rec)
}

// m57PasseRetenue marche le film avec la porte branchee — donc SOUS la porte de speculation — et
// rend les lectures d `i55` et d `i1` des records retenus, filtrees sur le bipede.
func m57PasseRetenue(t *testing.T) (*m57Rec, bool) {
	t.Helper()
	dir := os.Getenv("MOUV57_FILM")
	if dir == "" {
		t.Skip("MOUV57_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m57Contexte(t, film)
	reg, errR := fc.Registry()
	if errR != nil {
		t.Fatalf("registre : %v", errR)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	rec := &m57Rec{fautifs: map[int]int{}, etalon: map[int]int{}}
	var ts uint64
	cfg := fc.CadreDeBalayage()
	cfg.Obs = m57Observateur(rec, &ts)
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, okC := fc.ChunkAt(c)
		if !okC {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m57Paquet(pk, data, w, cfg, rec, &ts)
		}
	}
	m57Filtrer(t, rec, w)
	return rec, m57Oracle(t, rec)
}

// m57SprintDistribution publie l histogramme de la vitesse AU SOL, tous slots confondus, et
// compte ses MODES LOCAUX : deux bosses, c est un sprint observable ; une seule, non.
func m57SprintDistribution(t *testing.T, rec *m57Rec) {
	t.Helper()
	casiers := map[int]int{}
	sols := make([]float64, 0, len(rec.vit))
	for _, v := range rec.vit {
		sols = append(sols, v.sol)
		casiers[int(v.sol/m57SprintBinMS)]++
	}
	if len(sols) == 0 {
		t.Logf("SPRINT : aucune vitesse retenue")
		return
	}
	sort.Float64s(sols)
	var queue int
	for k, n := range casiers {
		if float64(k)*m57SprintBinMS >= 4 {
			queue += n
		}
	}
	t.Logf("SPRINT — VITESSE AU SOL DES RECORDS RETENUS (%d lectures) : mediane %.3f m/s "+
		"(p10 %.3f · p90 %.3f · p99 %.3f · max %.1f) · %d (%.2f %%) au-dela de 4 m/s",
		len(sols), m57Quantile(sols, 0.50), m57Quantile(sols, 0.10), m57Quantile(sols, 0.90),
		m57Quantile(sols, 0.99), sols[len(sols)-1], queue, m533bPart(queue, len(sols)))
	t.Logf("  HISTOGRAMME (casiers de %.1f m/s, jusqu a 12 m/s) : %s", m57SprintBinMS,
		strings.Join(m57Casiers(casiers), " · "))
	modes := m57ModesLocaux(casiers, len(sols))
	t.Logf("  MODES LOCAUX (> 0,5 %% des lectures, strictement au-dessus de leurs deux "+
		"voisins) : %d — %s", len(modes), strings.Join(modes, " · "))
}

// m57Casiers rend l histogramme jusqu a 12 m/s, en clair.
func m57Casiers(casiers map[int]int) []string {
	cles := make([]int, 0, len(casiers))
	for k := range casiers {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	for _, k := range cles {
		if k > 24 {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%.1f-%.1f[ %d", float64(k)*m57SprintBinMS,
			float64(k+1)*m57SprintBinMS, casiers[k]))
	}
	return parts
}

// m57ModesLocaux nomme les casiers strictement superieurs a leurs deux voisins et pesant plus
// d un demi pour cent des lectures.
func m57ModesLocaux(casiers map[int]int, total int) []string {
	cles := make([]int, 0, len(casiers))
	for k := range casiers {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var out []string
	for _, k := range cles {
		if casiers[k]*200 <= total {
			continue
		}
		if casiers[k] <= casiers[k-1] || casiers[k] <= casiers[k+1] {
			continue
		}
		out = append(out, fmt.Sprintf("%.1f-%.1f m/s (%d)", float64(k)*m57SprintBinMS,
			float64(k+1)*m57SprintBinMS, casiers[k]))
	}
	return out
}

// m57SlotVitesse resume la distribution de vitesse au sol d UNE vie.
type m57SlotVitesse struct {
	slot              uint32
	n                 int
	med, p95, rapport float64
}

// m57SprintParSlot cherche la seconde bosse LA OU ELLE SERAIT : dans la distribution d UNE vie.
func m57SprintParSlot(t *testing.T, rec *m57Rec) {
	t.Helper()
	parSlot := map[uint32][]float64{}
	for _, v := range rec.vit {
		parSlot[v.slot] = append(parSlot[v.slot], v.sol)
	}
	var ls []m57SlotVitesse
	for s, xs := range parSlot {
		if len(xs) < m57SprintMinLectures {
			continue
		}
		sort.Float64s(xs)
		med, p95 := m57Quantile(xs, 0.50), m57Quantile(xs, 0.95)
		r := 0.0
		if med > 0 {
			r = p95 / med
		}
		ls = append(ls, m57SlotVitesse{slot: s, n: len(xs), med: med, p95: p95, rapport: r})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].rapport > ls[j].rapport })
	t.Logf("SPRINT PAR VIE (%d vies a >= %d lectures) — le rapport p95/mediane d une vie qui "+
		"sprinte devrait depasser nettement celui d une vie qui marche :", len(ls),
		m57SprintMinLectures)
	for i, l := range ls {
		if i >= 8 {
			break
		}
		t.Logf("  slot %5d : %6d lectures · mediane %.3f · p95 %.3f · "+
			"rapport %.2f", l.slot, l.n, l.med, l.p95, l.rapport)
	}
	if len(ls) > 0 {
		t.Logf("  ... le plus faible des %d : rapport %.2f", len(ls), ls[len(ls)-1].rapport)
	}
}

// m57SautParTag repose le test du saut : le tag d `i55` contre la vitesse verticale TENUE.
func m57SautParTag(t *testing.T, rec *m57Rec) {
	t.Helper()
	parSlot := m57ParSlot(rec.vit)
	t.Logf("SAUT — LES QUATRE TAGS D `i55` SUR LA POPULATION RETENUE (%d lectures) :",
		len(rec.post))
	for tag := uint64(0); tag < 4; tag++ {
		var vz []float64
		var total, monte int
		for _, p := range rec.post {
			if p.tag != tag {
				continue
			}
			total++
			v, ok := m57VitTenue(parSlot[p.slot], p.ts)
			if !ok {
				continue
			}
			vz = append(vz, v.vz)
			if v.vz >= m57SeuilMontee {
				monte++
			}
		}
		sort.Float64s(vz)
		t.Logf("  tag %d : %4d lectures · %4d avec vitesse tenue · vz mediane "+
			"%+.3f m/s (p10 %+.3f · p90 %+.3f) · vz >= %.1f sur %d (%.1f %%)",
			tag, total, len(vz), m57Quantile(vz, 0.50), m57Quantile(vz, 0.10),
			m57Quantile(vz, 0.90), m57SeuilMontee, monte, m533bPart(monte, len(vz)))
	}
	m57MonteesSansCandidat(t, rec)
}
