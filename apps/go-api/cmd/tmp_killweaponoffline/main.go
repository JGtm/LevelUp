// tmp_killweaponoffline — THROWAWAY. VERROU FINAL : l'arme-par-kill 100% OFFLINE.
//
// Acquis (Ghidra FUN_14080c1f8 + sonde tmp_r5attacker) : le record de degat porte
// l'ATTAQUANT en clair = R5 (5 bits au bit 36, lu AVANT le slot). R5 in {0,2,..,14} = 8
// joueurs ; joueur = R5>>1. + la FAMILLE d'arme (variant_name) + le temps.
//
// Ici on CROISE avec le kill-feed (chunk_27) : pour chaque kill (tueur,victime,t), le tick
// de degat le plus proche AVANT t apprend le mapping tueur_xuid -> R5. Bijection propre 8<->8
// => l'attaquant offline est PROUVE, et l'arme du kill = la famille de ce tick.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const deserStartBit = 36
const variantSuffix = uint32(0x42c9679f)

var h32name = map[uint32]string{}

var piXuid = map[int]uint64{
	0: 2535467794760703, 1: 2535437947245250, 2: 2533274823110022, 3: 2533274980284321,
	4: 2533274815845110, 5: 2535444178793711, 6: 2533274882097883, 7: 2533274826120416,
}
var xuidName = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

func nm(x uint64) string {
	if g, ok := xuidName[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}
func xuidPi(x uint64) int {
	for pi, xu := range piXuid {
		if xu == x {
			return pi
		}
	}
	return -1
}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}
func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }

type dmg struct {
	tms int
	r5  uint64
	fam string
}

func decodeDmg() []dmg {
	build()
	var out []dmg
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		if len(d) == 0 {
			continue
		}
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			pl := d[off+16 : off+16+size]
			off += 16 + size
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			br := filmdec.NewBitReader(pl)
			br.Skip(deserStartBit)
			r5 := br.ReadBits(5) // ATTAQUANT (lu en premier, ordre deser)
			if br.ReadBit() {
			} else {
				br.ReadBits(2)
			}
			fam32 := uint32(br.ReadBits(32))
			low := uint32(br.ReadBits(32))
			fam, ok := h32name[fam32]
			if !ok || low != variantSuffix {
				continue
			}
			out = append(out, dmg{int((int64(ts) - int64(t0Us)) / 1000), r5, fam})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tms < out[j].tms })
	return out
}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
	}
}

type ev struct {
	xuid uint64
	t    int
}
type kfRow struct {
	killer, victim uint64
	t              int
}

func killFeed() []kfRow {
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	usedD := make([]bool, len(deaths))
	var feed []kfRow
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.xuid == k.xuid {
				continue
			}
			dt := k.t - d.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			usedD[best] = true
			feed = append(feed, kfRow{k.xuid, deaths[best].xuid, k.t})
		}
	}
	return feed
}

func main() {
	dmgs := decodeDmg()
	feed := killFeed()
	fmt.Printf("=== %d ticks de degat (attaquant+arme+temps) ; %d kills apparies ===\n", len(dmgs), len(feed))

	// plages temporelles (verifier que les deux bases sont alignees).
	if len(dmgs) > 0 {
		fmt.Printf("    ticks tms: [%d..%d]ms ; kills t: [%d..%d]ms\n",
			dmgs[0].tms, dmgs[len(dmgs)-1].tms, feed[0].t, feed[len(feed)-1].t)
	}

	// DIAGNOSTIC sparsite : pour chaque kill, existe-t-il UN tick (n'importe quel R5) proche ?
	// Si oui haute couverture -> records OK, le pb est le mapping R5->joueur (permutation).
	// Si non -> records 0xd2 trop epars / pas alignes sur les morts.
	for _, w := range []int{300, 700, 1500, 3000} {
		cov := 0
		for _, k := range feed {
			for _, d := range dmgs {
				ad := d.tms - k.t
				if ad < 0 {
					ad = -ad
				}
				if ad <= w {
					cov++
					break
				}
			}
		}
		fmt.Printf("    [sparsite] kills avec UN tick (tout R5) a +-%4dms : %d/%d\n", w, cov, len(feed))
	}

	// MODE drift : pour chaque kill, tick R5==tueur le plus proche (fenetre LARGE +-25s) ;
	// affiche (kill_time, delta) trie -> revele si l'offset derive sur le match + couverture max.
	if len(os.Args) > 1 && os.Args[1] == "drift" {
		type row struct {
			kt, delta int
			fam       string
		}
		var rows []row
		for _, k := range feed {
			pi := xuidPi(k.killer)
			if pi < 0 {
				continue
			}
			bd, bf, found := 1<<30, "", false
			for _, r := range dmgs {
				if r.r5 != uint64(pi*2) {
					continue
				}
				d := r.tms - k.t
				ad := d
				if ad < 0 {
					ad = -ad
				}
				if ad < bd {
					bd, bf, found = ad, r.fam, true
				}
			}
			if found {
				dd := 0
				for _, r := range dmgs {
					if r.r5 == uint64(pi*2) {
						d := r.tms - k.t
						ad := d
						if ad < 0 {
							ad = -ad
						}
						if ad == bd {
							dd = d
							break
						}
					}
				}
				rows = append(rows, row{k.t, dd, bf})
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].kt < rows[j].kt })
		fmt.Println("\n=== DRIFT : (kill_time, delta=tick-kill) pour le tick R5==tueur le plus proche (+-25s) ===")
		within := map[int]int{}
		for _, r := range rows {
			fmt.Printf("  t=%6.1fs  delta=%+6dms  %s\n", float64(r.kt)/1000, r.delta, r.fam)
			for _, w := range []int{1000, 2000, 4000, 8000} {
				ad := r.delta
				if ad < 0 {
					ad = -ad
				}
				if ad <= w {
					within[w]++
				}
			}
		}
		fmt.Printf("\n  couverture (tick R5==tueur, |delta|<=) : 1s=%d 2s=%d 4s=%d 8s=%d  (sur %d kills)\n",
			within[1000], within[2000], within[4000], within[8000], len(feed))
		return
	}

	// MODE teams : inferer les 2 equipes (2-coloration du graphe kill) puis split attribue/non.
	if len(os.Args) > 1 && os.Args[1] == "teams" {
		// adjacence "ennemi" : tueur<->victime sont adverses.
		enemy := map[int]map[int]int{}
		for _, k := range feed {
			a, b := xuidPi(k.killer), xuidPi(k.victim)
			if a < 0 || b < 0 {
				continue
			}
			if enemy[a] == nil {
				enemy[a] = map[int]int{}
			}
			if enemy[b] == nil {
				enemy[b] = map[int]int{}
			}
			enemy[a][b]++
			enemy[b][a]++
		}
		team := map[int]int{0: 0} // BFS 2-coloration depuis pi0
		queue := []int{0}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for q := range enemy[p] {
				if _, ok := team[q]; !ok {
					team[q] = 1 - team[p]
					queue = append(queue, q)
				}
			}
		}
		fmt.Println("\n=== EQUIPES inferees (2-coloration du kill-feed) ===")
		for pi := 0; pi < 8; pi++ {
			fmt.Printf("  pi%d %-16s -> equipe %c\n", pi, nm(piXuid[pi]), 'A'+byte(team[pi]))
		}
		// split attribue/non par equipe du TUEUR (offset -20300, R5==tueur).
		type cnt struct{ att, non int }
		byTeam := map[int]*cnt{0: {}, 1: {}}
		killsByTeam := map[int]int{}
		for _, k := range feed {
			pi := xuidPi(k.killer)
			if pi < 0 {
				continue
			}
			t := team[pi]
			killsByTeam[t]++
			if _, ok := nearestFamSigned(dmgs, k.t-20300, 800, uint64(pi*2)); ok {
				byTeam[t].att++
			} else {
				byTeam[t].non++
			}
		}
		fmt.Println("\n=== attribution par equipe du tueur ===")
		for t := 0; t < 2; t++ {
			fmt.Printf("  equipe %c : %d kills total ; attribues=%d ; non=%d\n",
				'A'+byte(t), killsByTeam[t], byTeam[t].att, byTeam[t].non)
		}
		// idem mais R5 = VICTIME (si les records suivent la victime/equipe encaissante).
		fmt.Println("\n=== meme split mais en matchant R5 == VICTIME ===")
		vt := map[int]*cnt{0: {}, 1: {}}
		for _, k := range feed {
			vp := xuidPi(k.victim)
			tp := xuidPi(k.killer)
			if vp < 0 || tp < 0 {
				continue
			}
			if _, ok := nearestFamSigned(dmgs, k.t-20300, 800, uint64(vp*2)); ok {
				vt[team[tp]].att++
			} else {
				vt[team[tp]].non++
			}
		}
		for t := 0; t < 2; t++ {
			fmt.Printf("  equipe %c (tueur) : attribues(via victime)=%d ; non=%d\n", 'A'+byte(t), vt[t].att, vt[t].non)
		}
		return
	}

	// MODE diag N : dump des ticks (R5=2N) + kills du joueur pi=N, pour voir l'alignement.
	if len(os.Args) > 2 && os.Args[1] == "diag" {
		var pi int
		fmt.Sscanf(os.Args[2], "%d", &pi)
		want := uint64(pi * 2)
		fmt.Printf("\n=== DIAG pi%d %s : ticks R5=%d (time, arme) ===\n", pi, nm(piXuid[pi]), want)
		for _, d := range dmgs {
			if d.r5 == want {
				fmt.Printf("  t=%7.2fs  %s\n", float64(d.tms)/1000, d.fam)
			}
		}
		fmt.Printf("\n=== kills de pi%d (time, victime) ===\n", pi)
		for _, k := range feed {
			if xuidPi(k.killer) == pi {
				fmt.Printf("  t=%7.2fs  -> %s\n", float64(k.t)/1000, nm(k.victim))
			}
		}
		return
	}

	// TEST DUAL : R5 = attaquant (tueur) OU victime ? Pour chaque kill, tick le plus proche
	// (signe) dont R5>>1 == tueur_pi vs == victime_pi. L'hypothese gagnante a des deltas serres.
	fmt.Println("\n=== TEST R5 = TUEUR vs VICTIME (delta signe du tick le plus proche, |delta|<=1500ms) ===")
	for _, hyp := range []string{"tueur", "victime"} {
		tight, tot := 0, 0
		var sum int
		for _, k := range feed {
			pi := xuidPi(k.killer)
			if hyp == "victime" {
				pi = xuidPi(k.victim)
			}
			if pi < 0 {
				continue
			}
			d, found := nearestSigned(dmgs, k.t, uint64(pi*2), 1500)
			if !found {
				continue
			}
			tot++
			sum += d
			if d <= 0 && d >= -1500 {
				tight++
			}
		}
		avg := 0
		if tot > 0 {
			avg = sum / tot
		}
		fmt.Printf("  R5=%-8s : %d/%d kills ont un tick |delta|<=1500ms ; dont %d AVANT le kill ; delta moyen=%dms\n",
			hyp, tot, len(feed), tight, avg)
	}

	// SCAN D'OFFSET LARGE (le decalage est ~-20s, hors de la plage initiale). Mapping = IDENTITE
	// (R5>>1==pi, ancre live IKE=slot4=pi4). On scanne l'offset qui maximise la couverture
	// (kill avec tick R5==pi(role) a |delta|<=400ms), pour role=tueur et role=victime.
	cov := func(useVictim bool, off, tol int) int {
		c := 0
		for _, k := range feed {
			pi := xuidPi(k.killer)
			if useVictim {
				pi = xuidPi(k.victim)
			}
			if pi < 0 {
				continue
			}
			if d, ok := nearestSigned(dmgs, k.t+off, uint64(pi*2), tol+200); ok {
				ad := d
				if ad < 0 {
					ad = -ad
				}
				if ad <= tol {
					c++
				}
			}
		}
		return c
	}
	fmt.Println("\n=== SCAN OFFSET LARGE (identite R5>>1==pi, |delta|<=400ms) ===")
	bestOff, bestCov, useVictim := 0, -1, false
	for _, uv := range []bool{false, true} {
		bo, bc := 0, -1
		for off := -30000; off <= 5000; off += 100 {
			if c := cov(uv, off, 400); c > bc {
				bc, bo = c, off
			}
		}
		lbl := "tueur"
		if uv {
			lbl = "victime"
		}
		fmt.Printf("  role=%-8s : meilleur offset=%+7dms -> %d/%d kills couverts\n", lbl, bo, bc, len(feed))
		if bc > bestCov {
			bestCov, bestOff, useVictim = bc, bo, uv
		}
	}
	role := "tueur"
	if useVictim {
		role = "victime"
	}
	fmt.Printf("\n>>> R5 = %s, offset=%+dms, couverture=%d/%d\n", role, bestOff, bestCov, len(feed))
	for _, tol := range []int{200, 400, 800, 1500} {
		fmt.Printf("    |delta|<=%4dms : %d/%d\n", tol, cov(useVictim, bestOff, tol), len(feed))
	}
	perm := map[int]uint64{0: 0, 1: 2, 2: 4, 3: 6, 4: 8, 5: 10, 6: 12, 7: 14} // identite pi->R5=pi*2

	// ATTRIBUTION : pour chaque kill, arme = famille du tick R5==pi(role)*2 le plus proche.
	fmt.Println("\n=== ARME PAR KILL (offline) — 30 premiers ===")
	var killers []uint64
	seen := map[uint64]bool{}
	for _, k := range feed {
		if !seen[k.killer] {
			seen[k.killer] = true
			killers = append(killers, k.killer)
		}
	}
	sort.Slice(killers, func(i, j int) bool { return xuidPi(killers[i]) < xuidPi(killers[j]) })
	weaponByKiller := map[uint64]map[string]int{}
	shown, attributed := 0, 0
	for _, k := range feed {
		pi := xuidPi(k.killer)
		if useVictim {
			pi = xuidPi(k.victim)
		}
		fam := "?"
		if pi >= 0 {
			if f, ok := nearestFamSigned(dmgs, k.t+bestOff, 800, perm[pi]); ok {
				fam, attributed = f, attributed+1
			}
		}
		if weaponByKiller[k.killer] == nil {
			weaponByKiller[k.killer] = map[string]int{}
		}
		weaponByKiller[k.killer][fam]++
		if shown < 30 {
			fmt.Printf("  t=%6.1fs  %-16s --[%-22s]--> %-16s\n", float64(k.t)/1000, nm(k.killer), fam, nm(k.victim))
			shown++
		}
	}
	fmt.Printf("\n=== COUVERTURE : %d/%d kills avec arme à feu attribuée ===\n", attributed, len(feed))
	fmt.Println("\n=== armes par tueur (totaux offline) ===")
	for _, x := range killers {
		fmt.Printf("  %-18s : %s\n", nm(x), famLine(weaponByKiller[x]))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// nearestFamSigned : famille du tick R5 le plus proche (signe) du temps t, |delta|<=win.
func nearestFamSigned(d []dmg, t, win int, r5 uint64) (string, bool) {
	bd, fam := win+1, ""
	for _, r := range d {
		if r.r5 != r5 {
			continue
		}
		ad := r.tms - t
		if ad < 0 {
			ad = -ad
		}
		if ad < bd {
			bd, fam = ad, r.fam
		}
	}
	return fam, fam != ""
}

// nearestSigned : tick le plus proche (dans le temps) avec le R5 donne ; renvoie delta = tick-kill.
func nearestSigned(d []dmg, t int, r5 uint64, win int) (int, bool) {
	best, bd, found := 0, win+1, false
	for _, r := range d {
		if r.r5 != r5 {
			continue
		}
		dt := r.tms - t
		ad := dt
		if ad < 0 {
			ad = -ad
		}
		if ad < bd {
			bd, best, found = ad, dt, true
		}
	}
	return best, found
}

func nearestBefore(d []dmg, t, win int) (uint64, bool) {
	best, bd, found := uint64(0), win+1, false
	for _, r := range d {
		if r.tms > t {
			break
		}
		dt := t - r.tms
		if dt <= win && dt < bd {
			bd, best, found = dt, r.r5, true
		}
	}
	return best, found
}

func nearestFamBefore(d []dmg, t, win int, r5 uint64) string {
	best, bd, fam := -1, win+1, "?"
	for _, r := range d {
		if r.tms > t {
			break
		}
		if r.r5 != r5 {
			continue
		}
		dt := t - r.tms
		if dt <= win && dt < bd {
			bd, best, fam = dt, r.tms, r.fam
		}
	}
	_ = best
	return fam
}

func mark(b bool) string {
	if b {
		return "OK"
	}
	return "x"
}
func famLine(m map[string]int) string {
	type fc struct {
		f string
		c int
	}
	var fcs []fc
	for f, c := range m {
		fcs = append(fcs, fc{f, c})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].c > fcs[j].c })
	s := ""
	for _, e := range fcs {
		s += fmt.Sprintf("%s:%d  ", e.f, e.c)
	}
	return s
}
