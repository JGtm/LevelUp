package filmdec

// navpoint_ti12_onebomb_test.go — INSPECTION STRUCTURELLE DES SERIES i14 EN ONE BOMB.
//
// # POURQUOI CE FICHIER
//
// Le plancher (navpoint_ti12_plancher_test.go) etablit que l'anneau du marqueur ti=12 i14 est la
// jauge d'armement sur Neutral Bomb (13/13, delai 4,93 s, CV 0,016, 0/1000 tirages nuls aussi
// bons) et Husky Raid (4/4, 5,1 s, CV 0,016, 0/1000). Sur les trois films One Bomb, la MEME
// lecture rend 11/11 mais delai median 17,2 s, CV 0,725, et 87/1000 tirages nuls font aussi
// bien : le signal ne tient pas. AVANT de proposer une autre lecture, ce fichier IMPRIME la
// structure brute autour de chaque explosion — il ne decide rien, il montre.
//
// # LES HYPOTHESES, et le critere de chacune, ECRITS AVANT LA MESURE
//
//	H1 MAUVAISE FAMILLE   deux familles d'anneaux coexistent (paires de slots a +12, un par
//	                      camp ; ou anneau d'un autre role du mode) et la lecture tous-slots
//	                      retient la mauvaise. TIENT si une sous-famille de slots porte un
//	                      delai court (~5 s) constant sur les explosions qu'elle couvre,
//	                      la montee retenue (la plus recente toutes familles) venant d'ailleurs.
//	H2 RAMPE FRAGMENTEE   la rampe d'armement existe mais la contiguite (trou <= 500 ms) ou les
//	                      seuils (>= 3 ech., amplitude >= 16 quanta) la cassent. TIENT si pour
//	                      au moins 9 explosions sur 11 une sequence croissante d'amplitude
//	                      >= 16 existe dans les 10 s avant l'explosion SANS qu'aucune montee
//	                      contigue valide ne s'y termine.
//	H3 ANNEAU DE RETOUR   un anneau de retour de bombe (comme au CTF) pollue la population.
//	                      TIENT si des montees valides existent loin de toute explosion
//	                      (aucune dans les 120 s qui suivent leur fin), en nombre comparable
//	                      a celui des montees d'armement.
//	NEGATIF DE STRUCTURE  aucune lecture croissante dans les 10 s avant la plupart des
//	                      explosions : One Bomb n'expose pas l'armement par ce canal.
//
// REGIME : garde ASSAUT_CACHE. Aucune base, aucun reseau, sentinelle memoire armee, un seul
// decodage a la fois (LockProcessDecode).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run NavpointTi12OneBombInspection -v -timeout 30m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// obFilms : les trois films One Bomb et leurs explosions (sous-ensemble de ti12Explosions,
// garde par TestNavpointTi12OracleFige).
var obFilms = []struct {
	id   string
	exps []int32
}{
	{"9f57c612", []int32{83322, 298489, 353160, 469057}},
	{"c75f33b8", []int32{109549, 395724, 450833}},
	{"df8fcbef", []int32{255767, 309284, 485860, 778033}},
}

const (
	obFenAvantMS = 45000 // fenetre d'inspection avant l'explosion
	obFenApresMS = 2000  // et un peu apres, pour voir la retombee
)

// obSeg est un segment CONTIGU (trou <= NavpointRiseMaxGapMS) d'une serie d'un slot, sans exigence de
// monotonie : c'est le fait brut, la montee est une interpretation.
type obSeg struct {
	slot       uint32
	t0, t1     int32
	q0, q1     uint8
	qmin, qmax uint8
	n          int
	gapAvant   int32 // ms depuis l'echantillon precedent du slot, -1 si premier
}

// TestNavpointTi12OneBombInspection imprime la structure des trois films One Bomb.
func TestNavpointTi12OneBombInspection(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer tpSentinelle(t)()
	release := LockProcessDecode()
	defer release()

	for _, f := range obFilms {
		series, ok := obCharger(t, cache, f.id)
		if !ok {
			t.Fatalf("%s : film indispensable absent", f.id)
		}
		t.Logf("=========================================== FILM %s", f.id)
		obVueGlobale(t, series)
		segs := obTousSegments(series)
		for _, e := range f.exps {
			obVueExplosion(t, segs, e)
		}
		obVueOrphelines(t, series, f.exps)
	}
}

// obFinsContigues rend les fins des montees contigues VALIDES d'une serie, par la detection
// de PRODUCTION (`NavpointContiguousRises`, portee le 2026-09-01 depuis l'instrument
// `tpMonteesContigues` — a l'identique, le plancher A/B/C en temoigne) : l'inspection
// One Bomb juge ainsi le code qui publie, pas une copie.
func obFinsContigues(slot uint32, s []ti12Ech) []int32 {
	reads := make([]NavpointRadialRead, 0, len(s))
	for _, e := range s {
		reads = append(reads, NavpointRadialRead{Slot: slot, TMS: e.tMS, Q: e.q})
	}
	var fins []int32
	for _, m := range NavpointContiguousRises(reads) {
		fins = append(fins, m.EndMS)
	}
	return fins
}

// obCharger balaye UN film et rend les series i14 triees par slot.
func obCharger(t *testing.T, cache, id string) (map[uint32][]ti12Ech, bool) {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	if CountFilmChunks(dir) == 0 {
		return nil, false
	}
	clk, ok := ti12Horloge(dir)
	if !ok {
		return nil, false
	}
	sc, err := ScanFilmNavpointRadial(dir, clk.startMS)
	if err != nil {
		return nil, false
	}
	series := map[uint32][]ti12Ech{}
	for _, r := range sc.Reads {
		series[r.Slot] = append(series[r.Slot], ti12Ech{r.TMS, r.Q})
	}
	for _, s := range series {
		sort.Slice(s, func(i, j int) bool { return s[i].tMS < s[j].tMS })
	}
	return series, true
}

// obVueGlobale imprime une ligne par slot : volume, etendue, montees contigues et non contigues.
func obVueGlobale(t *testing.T, series map[uint32][]ti12Ech) {
	t.Helper()
	slots := make([]uint32, 0, len(series))
	for s := range series {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	t.Logf("--- VUE GLOBALE : %d slot(s)", len(slots))
	for _, slot := range slots {
		s := series[slot]
		mc := obFinsContigues(slot, s)
		mn := ti12Montees(slot, s)
		t.Logf("    slot %-5d %5d ech. de %7.1fs a %7.1fs · %2d montee(s) contigue(s), "+
			"%2d non contigue(s)", slot, len(s), float64(s[0].tMS)/1000,
			float64(s[len(s)-1].tMS)/1000, len(mc), len(mn))
	}
}

// obTousSegments segmente toutes les series par contiguite (trou <= NavpointRiseMaxGapMS).
func obTousSegments(series map[uint32][]ti12Ech) []obSeg {
	var out []obSeg
	for slot, s := range series {
		out = append(out, obSegmenter(slot, s)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].t0 != out[j].t0 {
			return out[i].t0 < out[j].t0
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// obSegmenter decoupe UNE serie triee en segments contigus, sans exigence de monotonie.
func obSegmenter(slot uint32, s []ti12Ech) []obSeg {
	var out []obSeg
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1].tMS-s[j].tMS <= NavpointRiseMaxGapMS {
			j++
		}
		seg := obSeg{slot: slot, t0: s[i].tMS, t1: s[j].tMS, q0: s[i].q, q1: s[j].q,
			qmin: s[i].q, qmax: s[i].q, n: j - i + 1, gapAvant: -1}
		for k := i; k <= j; k++ {
			if s[k].q < seg.qmin {
				seg.qmin = s[k].q
			}
			if s[k].q > seg.qmax {
				seg.qmax = s[k].q
			}
		}
		if i > 0 {
			seg.gapAvant = s[i].tMS - s[i-1].tMS
		}
		out = append(out, seg)
		i = j + 1
	}
	return out
}

// obVueExplosion imprime les segments qui intersectent la fenetre autour d'une explosion.
func obVueExplosion(t *testing.T, segs []obSeg, exp int32) {
	t.Helper()
	lo, hi := exp-obFenAvantMS, exp+obFenApresMS
	t.Logf("--- EXPLOSION %d ms (fenetre %.1fs .. %.1fs)", exp, float64(lo)/1000, float64(hi)/1000)
	n := 0
	for _, g := range segs {
		if g.t1 < lo || g.t0 > hi {
			continue
		}
		n++
		gap := "premier"
		if g.gapAvant >= 0 {
			gap = fmt.Sprintf("trou %6d ms", g.gapAvant)
		}
		forme := obForme(g)
		t.Logf("    slot %-5d %8.1fs -> %8.1fs  q %3d->%3d (min %3d, max %3d) n=%-3d %s · "+
			"fin a %+7.1fs de l'explosion · %s",
			g.slot, float64(g.t0)/1000, float64(g.t1)/1000, g.q0, g.q1, g.qmin, g.qmax, g.n,
			forme, float64(g.t1-exp)/1000, gap)
	}
	if n == 0 {
		t.Logf("    (aucun segment dans la fenetre)")
	}
}

// obForme resume la forme d'un segment : plat, montee, descente, ou mixte.
func obForme(g obSeg) string {
	switch {
	case g.qmin == g.qmax:
		return "PLAT     "
	case g.q1 > g.q0 && g.qmin == g.q0 && g.qmax == g.q1:
		return "MONTEE   "
	case g.q1 < g.q0 && g.qmax == g.q0 && g.qmin == g.q1:
		return "DESCENTE "
	default:
		return "MIXTE    "
	}
}

// obVueOrphelines recense les montees contigues valides NON suivies d'une explosion (H3) et
// celles qui en precedent une, avec le delai.
func obVueOrphelines(t *testing.T, series map[uint32][]ti12Ech, exps []int32) {
	t.Helper()
	type fin struct {
		slot uint32
		fMS  int32
	}
	var fins []fin
	for slot, s := range series {
		for _, f := range obFinsContigues(slot, s) {
			fins = append(fins, fin{slot, f})
		}
	}
	sort.Slice(fins, func(i, j int) bool { return fins[i].fMS < fins[j].fMS })
	suivies, orphelines := 0, 0
	var sb strings.Builder
	for _, f := range fins {
		d := int32(-1)
		for _, e := range exps {
			if e >= f.fMS && (d < 0 || e-f.fMS < d) {
				d = e - f.fMS
			}
		}
		if d >= 0 && d <= tpSensMaxMS {
			suivies++
			fmt.Fprintf(&sb, " [slot %d fin %.1fs -> expl +%.1fs]", f.slot,
				float64(f.fMS)/1000, float64(d)/1000)
		} else {
			orphelines++
			fmt.Fprintf(&sb, " [slot %d fin %.1fs ORPHELINE]", f.slot, float64(f.fMS)/1000)
		}
	}
	t.Logf("--- MONTEES CONTIGUES VALIDES : %d suivie(s) d'une explosion sous %d s, "+
		"%d orpheline(s)", suivies, tpSensMaxMS/1000, orphelines)
	for _, l := range strings.Split(sb.String(), "] ") {
		if l == "" {
			continue
		}
		t.Logf("    %s]", strings.TrimSuffix(strings.TrimSpace(l), "]"))
	}
}
