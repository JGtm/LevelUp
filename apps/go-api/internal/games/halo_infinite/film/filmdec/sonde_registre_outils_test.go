package filmdec

// sonde_registre_outils_test.go — L'OUTILLAGE PARTAGE des sondes du lot F : compter des
// valeurs, les journaliser, mesurer une densite autour d'evenements, ecrire un TSV.
//
// IL VIT A PART parce que les verdicts (`sonde_registre_verdicts_test.go`) et le moteur
// (`sonde_registre_scan_test.go`) sont deux lectures differentes de la meme recolte, et que ces
// helpers servent aux deux. Les laisser dans le fichier des verdicts poussait celui-ci au-dela
// du seuil de 500 lignes du depot, et melangeait « ce qu'on conclut » avec « comment on
// compte » — deux choses qu'une relecture doit pouvoir separer.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// probeValeur est une valeur observee et son compte.
type probeValeur struct {
	cle     uint64
	libelle string
	n       int
}

func probeFiltre(es []probeEmission, comp ProbeComponent) []probeEmission {
	var out []probeEmission
	for _, e := range es {
		if e.comp == comp {
			out = append(out, e)
		}
	}
	return out
}

// probeCle reduit un n-uplet de valeurs a une cle : les composants sondes n'en portent qu'une,
// mais la signature du hook en admet plusieurs et une cle qui ignorerait les suivantes
// confondrait deux emissions differentes.
func probeCle(vals []uint64) uint64 {
	var k uint64 = 1469598103934665603
	for _, v := range vals {
		k = (k ^ v) * 1099511628211
	}
	return k
}

func probeLibelle(vals []uint64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, "/")
}

// probeCompteValeurs agrege les emissions par valeur, triees par frequence decroissante.
func probeCompteValeurs(es []probeEmission) []probeValeur {
	n := map[uint64]int{}
	lib := map[uint64]string{}
	for _, e := range es {
		k := probeCle(e.vals)
		n[k]++
		lib[k] = probeLibelle(e.vals)
	}
	out := make([]probeValeur, 0, len(n))
	for k, c := range n {
		out = append(out, probeValeur{cle: k, libelle: lib[k], n: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].libelle < out[j].libelle
	})
	return out
}

func probeJournaliseValeurs(t *testing.T, vals []probeValeur, total, top int) {
	t.Helper()
	for i, v := range vals {
		if i >= top {
			t.Logf("      ... et %d autres valeurs", len(vals)-top)
			break
		}
		t.Logf("      %-24s %7d  %5.2f %%", v.libelle, v.n, 100*float64(v.n)/float64(maxI(1, total)))
	}
}

func probeJournaliseDeltas(t *testing.T, dts []int64) {
	t.Helper()
	if len(dts) == 0 {
		return
	}
	sort.Slice(dts, func(i, j int) bool { return dts[i] < dts[j] })
	classes := map[int64]int{}
	for _, d := range dts {
		classes[d]++
	}
	keys := make([]int64, 0, len(classes))
	for k := range classes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return classes[keys[i]] > classes[keys[j]] })
	t.Logf("  ecarts entre emissions (ms) : mediane %d · p95 %d · max %d · %d valeurs distinctes",
		dts[len(dts)/2], dts[minI(len(dts)-1, 95*len(dts)/100)], dts[len(dts)-1], len(classes))
	for i, k := range keys {
		if i >= 6 {
			break
		}
		t.Logf("      ecart %4d ms : %7d fois (%5.2f %%)", k, classes[k],
			100*float64(classes[k])/float64(len(dts)))
	}
}

// probeProche dit si un instant tombe dans la fenetre d'un evenement.
func probeProche(instants []int, tMS int) bool {
	i := sort.SearchInts(instants, tMS-probeFenetreMS)
	return i < len(instants) && instants[i] <= tMS+probeFenetreMS
}

// probeSecondes rend le nombre de secondes couvertes par les fenetres, et le reste du match. Les
// fenetres se RECOUVRENT : les additionner sans les fusionner gonflerait le denominateur et
// ferait passer un canal quelconque pour un canal concentre.
func probeSecondes(instants []int, es []probeEmission) (float64, float64) {
	if len(instants) == 0 || len(es) == 0 {
		return 0, 0
	}
	lo, hi := math.MaxInt, math.MinInt
	for _, e := range es {
		if e.tMS < 0 {
			continue
		}
		lo, hi = minI(lo, e.tMS), maxI(hi, e.tMS)
	}
	if lo > hi {
		return 0, 0
	}
	var couvert int
	debut, fin := instants[0]-probeFenetreMS, instants[0]+probeFenetreMS
	for _, t := range instants[1:] {
		if t-probeFenetreMS > fin {
			couvert += fin - debut
			debut, fin = t-probeFenetreMS, t+probeFenetreMS
			continue
		}
		fin = t + probeFenetreMS
	}
	couvert += fin - debut
	total := hi - lo
	return float64(couvert) / 1000, float64(maxI(0, total-couvert)) / 1000
}

func probeDensite(dedans, dehors int, secIn, secOut float64) string {
	if dehors == 0 {
		if dedans == 0 {
			return "—"
		}
		return "hors fenetre nul"
	}
	d := (float64(dedans) / secIn) / (float64(dehors) / secOut)
	return fmt.Sprintf("%.2fx", d)
}

func probeRapport(a, b int) string {
	if b == 0 {
		if a == 0 {
			return "0/0"
		}
		return "fantome NUL"
	}
	return fmt.Sprintf("%.2fx", float64(a)/float64(b))
}

// probeEcrisValeursTSV depose la table complete des valeurs d'un composant.
func probeEcrisValeursTSV(t *testing.T, dir, tag string, vals []probeValeur, total int) {
	t.Helper()
	out := os.Getenv(probeTSVEnv)
	if out == "" {
		return
	}
	var b strings.Builder
	b.WriteString("valeur\tcompte\tpart\n")
	for _, v := range vals {
		fmt.Fprintf(&b, "%s\t%d\t%.6f\n", v.libelle, v.n, float64(v.n)/float64(maxI(1, total)))
	}
	path := filepath.Join(out, filepath.Base(dir)+"_"+tag+"_valeurs.tsv")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture TSV : %v", err)
	}
	t.Logf("    TSV : %s", path)
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}
