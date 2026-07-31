// tmp_killweapon_csv — THROWAWAY. Analyse la capture CE filmdec_killweapon.csv.
// But : (1) prouver que weaponHandle/weaponDefId encodent le TUEUR (pas l'arme),
// (2) extraire de façon fiable la paire X/Y (tueur/victime) par kill,
// (3) sortir la matrice tueur->victime + totaux pour valider vs la DB.
//
// Colonnes : weaponDefId,weaponHandle,attacker,p00,p04,p08,p0C,p10,...,p4C
//
//	p04 = event+0x04 = victime ; p08 = event+0x08 = tueur (cf. memory deadstate).
//	Index joueur = (val - base) / 0x10002 (espacement constaté).
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const csvPath = `C:/Users/Guillaume/Downloads/filmdec_killweapon.csv`
const step = 0x10002 // 65538 : espacement des handles joueur

type row struct {
	weaponDefID uint32
	weaponHnd   uint32
	attacker    uint32
	p           []uint32 // p00..p4C (20 valeurs)
}

func main() {
	rows := parse()
	fmt.Printf("=== %d lignes parsées ===\n\n", len(rows))

	// base = min des valeurs p04/p08 (handles joueur)
	base := ^uint32(0)
	for _, r := range rows {
		for _, v := range []uint32{r.p[1], r.p[2]} { // p04, p08
			if v < base {
				base = v
			}
		}
	}
	idx := func(v uint32) int {
		if v < base || (v-base)%step != 0 {
			return -1 // pas un handle joueur aligné
		}
		return int((v - base) / step)
	}

	// --- 1. weaponHandle est-il aligné joueur, et == tueur (p08) ? ---
	hndBase := ^uint32(0)
	for _, r := range rows {
		if r.weaponHnd < hndBase {
			hndBase = r.weaponHnd
		}
	}
	hndIdx := func(v uint32) int {
		if v < hndBase || (v-hndBase)%step != 0 {
			return -1
		}
		return int((v - hndBase) / step)
	}
	distinctHnd := map[uint32]bool{}
	matchKiller, hndAligned := 0, 0
	for _, r := range rows {
		distinctHnd[r.weaponHnd] = true
		hi := hndIdx(r.weaponHnd)
		if hi >= 0 {
			hndAligned++
		}
		if hi == idx(r.p[2]) { // p08 = tueur
			matchKiller++
		}
	}
	fmt.Printf("--- weaponHandle ([R13+0x538]) ---\n")
	fmt.Printf("  valeurs distinctes : %d (espacées de 0x10002 si =8)\n", len(distinctHnd))
	fmt.Printf("  alignées index joueur : %d/%d\n", hndAligned, len(rows))
	fmt.Printf("  == index TUEUR (p08) : %d/%d  >>> si 100%%, event+0x538 = entité du tueur, PAS l'arme\n\n", matchKiller, len(rows))

	// --- 2. weaponDefId : distribution (corrélée au tueur ?) ---
	defByKiller := map[int]map[uint32]int{}
	for _, r := range rows {
		k := idx(r.p[2])
		if defByKiller[k] == nil {
			defByKiller[k] = map[uint32]int{}
		}
		defByKiller[k][r.weaponDefID]++
	}
	fmt.Printf("--- weaponDefId (FUN_14049d198(handle)) par index tueur ---\n")
	var ks []int
	for k := range defByKiller {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		parts := []string{}
		for d, c := range defByKiller[k] {
			parts = append(parts, fmt.Sprintf("%d×%d", int32(d), c))
		}
		fmt.Printf("  tueur idx%2d : %s\n", k, strings.Join(parts, " "))
	}
	fmt.Println("  >>> si def-id constant par tueur (et = -1/0), il n'encode AUCUNE arme.")

	// --- 3. Matrice tueur(p08) -> victime(p04) + totaux ---
	mat := map[[2]int]int{}
	killTot := map[int]int{}
	deathTot := map[int]int{}
	misaligned := 0
	for _, r := range rows {
		k, v := idx(r.p[2]), idx(r.p[1])
		if k < 0 || v < 0 {
			misaligned++
			continue
		}
		mat[[2]int{k, v}]++
		killTot[k]++
		deathTot[v]++
	}
	fmt.Printf("\n--- Matrice TUEUR (ligne) -> VICTIME (col), base=%d, %d non-alignés ---\n", base, misaligned)
	fmt.Printf("        ")
	for v := 0; v < 8; v++ {
		fmt.Printf(" v%-2d", v)
	}
	fmt.Printf("  | KILLS\n")
	for k := 0; k < 8; k++ {
		fmt.Printf("  k%-2d : ", k)
		for v := 0; v < 8; v++ {
			c := mat[[2]int{k, v}]
			if c == 0 {
				fmt.Printf("  . ")
			} else {
				fmt.Printf(" %2d ", c)
			}
		}
		fmt.Printf("  |  %2d\n", killTot[k])
	}
	fmt.Printf("  DEATHS ")
	for v := 0; v < 8; v++ {
		fmt.Printf(" %2d ", deathTot[v])
	}
	fmt.Println()

	total := 0
	for _, c := range killTot {
		total += c
	}
	fmt.Printf("\n  total kills alignés : %d (CSV brut = %d ; le surplus = outro)\n", total, len(rows))

	// --- 3b. Profil de chaque colonne event p00..p4C : porte-t-elle une ARME ? ---
	// Une colonne "arme/source de dégât" aurait : peu de valeurs distinctes (~types d'armes
	// du match, 5-15), NON déterminée par le seul tueur ni la seule victime, et NON constante.
	names := []string{"p00", "p04", "p08", "p0C", "p10", "p14", "p18", "p1C", "p20", "p24",
		"p28", "p2C", "p30", "p34", "p38", "p3C", "p40", "p44", "p48", "p4C"}
	fmt.Printf("\n--- Profil colonnes event (candidat ARME = peu de valeurs, indép. tueur/victime) ---\n")
	fmt.Printf("  %-5s %-8s %-22s %-22s %s\n", "col", "distinct", "déterminé-par-tueur?", "déterminé-par-victime?", "note")
	for ci, nm := range names {
		distinct := map[uint32]bool{}
		byK := map[int]map[uint32]bool{}
		byV := map[int]map[uint32]bool{}
		for _, r := range rows {
			val := r.p[ci]
			distinct[val] = true
			k, v := idx(r.p[2]), idx(r.p[1])
			if byK[k] == nil {
				byK[k] = map[uint32]bool{}
			}
			if byV[v] == nil {
				byV[v] = map[uint32]bool{}
			}
			byK[k][val] = true
			byV[v][val] = true
		}
		detK := true
		for _, s := range byK {
			if len(s) > 1 {
				detK = false
			}
		}
		detV := true
		for _, s := range byV {
			if len(s) > 1 {
				detV = false
			}
		}
		note := ""
		switch {
		case len(distinct) == 1:
			note = "CONSTANT"
		case nm == "p04" || nm == "p08":
			note = "= victime/tueur (connu)"
		case detK:
			note = "fonction du TUEUR"
		case detV:
			note = "fonction de la VICTIME"
		case len(distinct) <= 20:
			note = "<<< CANDIDAT ARME (peu de valeurs, indépendant)"
		default:
			note = "trop de valeurs (entité/pos/float)"
		}
		fmt.Printf("  %-5s %-8d %-22v %-22v %s\n", nm, len(distinct), detK, detV, note)
	}

	// --- 3c. p0C = catégorie de dégât ? Décodage high16/low16 par tueur ---
	// Roster résolu par bijection film<->DB (tmp_killvalidate).
	roster := map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm",
		3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}
	// catégorie supposée par high16 (memory dead-state +0x0c)
	catName := func(p0c uint32) string {
		switch p0c >> 16 {
		case 4:
			return "MÊLÉE"
		case 1:
			return "distance/lancé(1)"
		case 2:
			return "cat2"
		case 0:
			return "cat0"
		default:
			return fmt.Sprintf("high%d", p0c>>16)
		}
	}
	fmt.Printf("\n--- p0C décodé par tueur (high16=catégorie ? low16=modif) ---\n")
	fmt.Printf("  Validation : IKE ILYA (idx4, 'marteau') doit être ~MÊLÉE (high=4)\n")
	for k := 0; k < 8; k++ {
		hi := map[uint32]int{}
		raw := map[uint32]int{}
		n := 0
		for _, r := range rows {
			if idx(r.p[2]) != k {
				continue
			}
			n++
			hi[r.p[3]>>16]++
			raw[r.p[3]]++
		}
		parts := []string{}
		for h, c := range hi {
			parts = append(parts, fmt.Sprintf("high%d×%d", h, c))
		}
		fmt.Printf("  idx%d %-18s (%2d kills) : %s\n", k, roster[k], n, strings.Join(parts, " "))
	}
	fmt.Printf("\n--- Toutes les valeurs p0C distinctes (high|low) ---\n")
	seen := map[uint32]int{}
	for _, r := range rows {
		seen[r.p[3]]++
	}
	var vals []uint32
	for v := range seen {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	for _, v := range vals {
		fmt.Printf("  0x%05X (high=%d low=%d) ×%-3d  [%s]\n", v, v>>16, v&0xffff, seen[v], catName(v))
	}

	// --- 4. Ordre temporel : les N derniers kills (candidats outro) ---
	fmt.Printf("\n--- 8 derniers kills (ordre de capture = ordre replay) ---\n")
	startAt := len(rows) - 8
	if startAt < 0 {
		startAt = 0
	}
	for i := startAt; i < len(rows); i++ {
		r := rows[i]
		fmt.Printf("  #%2d : tueur idx%d -> victime idx%d  (p00=%d p0C=%d p1C=%d)\n",
			i+1, idx(r.p[2]), idx(r.p[1]), r.p[3], r.p[4], r.p[7])
	}
}

func parse() []row {
	f, err := os.Open(csvPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var out []row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "weaponDefId") {
			continue
		}
		f := strings.Split(line, ",")
		if len(f) < 23 {
			continue
		}
		u := func(s string) uint32 { n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64); return uint32(n) }
		r := row{weaponDefID: u(f[0]), weaponHnd: u(f[1]), attacker: u(f[2])}
		for i := 3; i < 23; i++ {
			r.p = append(r.p, u(f[i]))
		}
		out = append(out, r)
	}
	return out
}
