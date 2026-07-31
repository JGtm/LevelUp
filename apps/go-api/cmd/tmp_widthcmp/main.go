// tmp_widthcmp — CONFRONTATION composant par composant : largeur VRAIE (verite terrain CE)
// contre largeur que NOS desers consomment. THROWAWAY.
//
// METHODE
//  1. Verite : .ai/V7.5/dumps/ce_capture_delta.csv, differences de curseurs entre composants
//     consecutifs d'un meme record (cf. cmd/tmp_widthtruth ; le hook CE est sur le SEUL
//     site de dispatch, la capture est donc exhaustive et les composants captures
//     consecutifs sont bien consecutifs dans le flux).
//  2. Nous : chaque deser porte est execute sur N flux de bits ALEATOIRES, ce qui explore
//     ses branches et donne son ENSEMBLE DE LARGEURS ATTEIGNABLES. Un deser a largeur fixe
//     rend toujours la meme valeur, quels que soient les bits — la comparaison est alors
//     valable MEME si notre walk est desynchronise sur le film (c'est tout l'interet :
//     elle ne depend d'aucun film, d'aucune carte, d'aucun alignement).
//
// VERDICT par composant :
//
//	IMPOSSIBLE : la largeur VRAIE (mode) n'est PAS atteignable par notre deser -> divergence
//	             certaine, quelle que soit l'entree.
//	FIXE-OK    : notre deser est a largeur fixe et elle egale la largeur vraie.
//	ATTEIGNABLE: largeur variable chez nous, la valeur vraie est dans notre ensemble
//	             (pas de preuve de faute par cette methode).
//	NON-PORTE  : consumeByName renvoie ported=false.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_widthcmp [csv] [filmdir]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defCSV  = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_capture_delta.csv`
	defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	trials  = 40000
	bufLen  = 512 // octets : 4096 bits, assez pour tous les desers du bipede
)

// mapDependent : composants dont la largeur depend de la CARTE (quantification de position
// peuplee au chargement de carte) et qui ne sont donc PAS comparables entre deux films.
// Source : registry.go (les largeurs d'i0 changent de carte en carte : 13/13/14 vs 15/15/15)
// et i0_layout.go (DetectI0Layout lit le decoupage dans le bitstream).
var mapDependent = map[string]string{
	"object-position-dynamic-precision-component":               "quantification de position (bornes du BSP de la carte)",
	"object-translational-velocity-dynamic-precision-component": "dynamic-precision derive des memes bornes",
	"object-angular-velocity-dynamic-precision-component":       "dynamic-precision derive des memes bornes",
	"unit-desired-aiming-vector-component":                      "dequant de direction (largeurs 0xc/0xb litterales : A VERIFIER, cf. rapport)",
}

type hit struct{ eid, ti, ci, p4, cur int }

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

func main() {
	csvPath, filmDir := defCSV, defFilm
	if len(os.Args) > 1 {
		csvPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		filmDir = os.Args[2]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(filmDir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	arch, ok := reg.Archetype(35)
	if !ok {
		panic("archetype 35 absent")
	}

	// ---------- verite terrain ----------
	f, err := os.Open(csvPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var hits []hit
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" || strings.HasPrefix(ln, "#") || strings.HasPrefix(ln, "eid") {
			continue
		}
		p := strings.Split(ln, ",")
		if len(p) < 5 {
			continue
		}
		a, _ := strconv.Atoi(p[0])
		b, _ := strconv.Atoi(p[1])
		c, _ := strconv.Atoi(p[2])
		d, _ := strconv.Atoi(p[3])
		e, _ := strconv.Atoi(p[4])
		hits = append(hits, hit{a, b, c, d, e})
	}
	truth := map[int]map[int]int{} // ci -> largeur -> n   (archetype 35 seulement)
	p4of := map[int]map[int]int{}
	var recCur []hit
	flush := func() {
		for i := 0; i+1 < len(recCur); i++ {
			if recCur[i].ti != 35 {
				continue
			}
			ci := recCur[i].ci
			if truth[ci] == nil {
				truth[ci] = map[int]int{}
			}
			truth[ci][recCur[i+1].cur-recCur[i].cur]++
		}
		for _, h := range recCur {
			if h.ti != 35 {
				continue
			}
			if p4of[h.ci] == nil {
				p4of[h.ci] = map[int]int{}
			}
			p4of[h.ci][h.p4]++
		}
		recCur = recCur[:0]
	}
	for _, h := range hits {
		if n := len(recCur); n > 0 {
			p := recCur[n-1]
			if h.eid != p.eid || h.ci <= p.ci || h.cur <= p.cur {
				flush()
			}
		}
		recCur = append(recCur, h)
	}
	flush()

	// ---------- notre largeur : ensemble atteignable sur bits aleatoires ----------
	rng := rand.New(rand.NewSource(20260726))
	buf := make([]byte, bufLen)
	type ourStat struct {
		hist   map[int]int
		ported bool
	}
	ours := map[int]ourStat{}
	for i, name := range arch.Components {
		st := ourStat{hist: map[int]int{}}
		for t := 0; t < trials; t++ {
			for j := range buf {
				buf[j] = byte(rng.Intn(256))
			}
			w, ported := filmdec.ProbeComponentConsumedWidth(name, 35, arch.Level(i), buf)
			st.ported = ported
			if !ported {
				break
			}
			st.hist[w]++
		}
		ours[i] = st
	}

	// ---------- confrontation ----------
	var idx []int
	for ci := range truth {
		idx = append(idx, ci)
	}
	sort.Ints(idx)

	fmt.Printf("%-4s %-44s %6s %6s %6s  %-24s %-11s %s\n",
		"i", "composant", "nVrai", "vrai", "part", "nos largeurs (top)", "verdict", "carte?")
	fmt.Println(strings.Repeat("-", 140))
	firstDiv := -1
	for _, ci := range idx {
		name := ""
		if ci < len(arch.Components) {
			name = arch.Components[ci]
		}
		tw, tn, ttot := modeOf(truth[ci])
		st := ours[ci]
		verdict, ourDesc := "?", ""
		switch {
		case !st.ported:
			verdict, ourDesc = "NON-PORTE", "-"
		default:
			ourDesc = topOf(st.hist, 3)
			if _, hit := st.hist[tw]; !hit {
				verdict = "IMPOSSIBLE"
				if firstDiv < 0 {
					firstDiv = ci
				}
			} else if len(st.hist) == 1 {
				verdict = "FIXE-OK"
			} else {
				verdict = "ATTEIGNABLE"
			}
		}
		md := ""
		if r, ok := mapDependent[name]; ok {
			md = "OUI (" + r + ")"
		}
		fmt.Printf("i%02d  %-44s %6d %6d %5.1f%%  %-24s %-11s %s\n",
			ci, trunc(name, 44), ttot, tw, 100*float64(tn)/float64(ttot), trunc(ourDesc, 24), verdict, md)
	}

	fmt.Println()
	if firstDiv >= 0 {
		fmt.Printf("PREMIER ECART (ordre des index) : i%02d %s\n", firstDiv, arch.Components[firstDiv])
	} else {
		fmt.Println("AUCUN ecart de type IMPOSSIBLE detecte.")
	}

	// param_4 reel par composant : la valeur runtime dont dependent i10/i19/i20/i23.
	fmt.Println("\n=== param_4 (recordStateParam) REEL par composant, archetype 35 ===")
	for _, ci := range idx {
		if p4of[ci] == nil {
			continue
		}
		name := ""
		if ci < len(arch.Components) {
			name = arch.Components[ci]
		}
		fmt.Printf("i%02d  %-46s %s\n", ci, trunc(name, 46), topOf(p4of[ci], 4))
	}
}

func modeOf(h map[int]int) (val, n, tot int) {
	for k, v := range h {
		tot += v
		if v > n || (v == n && k < val) {
			val, n = k, v
		}
	}
	return
}

func topOf(h map[int]int, k int) string {
	type kv struct{ k, v int }
	var a []kv
	tot := 0
	for x, y := range h {
		a = append(a, kv{x, y})
		tot += y
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].v != a[j].v {
			return a[i].v > a[j].v
		}
		return a[i].k < a[j].k
	})
	var sb []string
	for i := 0; i < len(a) && i < k; i++ {
		sb = append(sb, fmt.Sprintf("%d:%.0f%%", a[i].k, 100*float64(a[i].v)/float64(tot)))
	}
	return strings.Join(sb, " ")
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
