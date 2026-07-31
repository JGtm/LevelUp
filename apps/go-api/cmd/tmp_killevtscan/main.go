// tmp_killevtscan — THROWAWAY, OEIL NEUF : localiser la LISTE de records kill-event dans le film.
// Grammaire (deser FUN_14104bd08, vérifiée fraîche en Ghidra) :
//
//	victim   = R(1) present(bit==0) + [R(5)]
//	killer   = R(1) present(bit==0) + [R(5)]
//	R(32) brut
//	R(1) bool
//	assistant= R(1) present(bit==0) + [R(5)]
//	R(32) brut
//	[tail conditionnel feature-gated, supposé absent en retail]
//
// Hypothèse fraîche : un format de replay scalable stocke les kills en LISTE CONTIGUË. On cherche
// donc l'offset bit d'où N records kill-event consécutifs se décodent (victim+killer présents,
// distincts, indices < maxIdx). Un run long par hasard est ~impossible. On scanne les octets bruts
// inflatés de chaque chunk (indépendant de la boucle ECS / World / reach).
//
// Bit reader MSB-first standalone (convention confirmée par FUN_1406cf008 : top-bit puis shift-left).
//
// Usage : tmp_killevtscan [maxIdx] [minRun] [withTail0|1]
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

// MSB-first bit read at absolute bit position. Pas de struct : on passe le curseur.
func bit(d []byte, bp int) int {
	if bp>>3 >= len(d) {
		return -1
	}
	return int((d[bp>>3] >> uint(7-(bp&7))) & 1)
}
func bits(d []byte, bp, n int) (int, int) { // returns (value, newBp); -1 si dépasse
	v := 0
	for i := 0; i < n; i++ {
		b := bit(d, bp)
		if b < 0 {
			return -1, bp
		}
		v = (v << 1) | b
		bp++
	}
	return v, bp
}

// readOpt5 : R(1) present (bit==0) -> R(5) ; sinon -1 (absent). Retourne (val, newBp, present).
func readOpt5(d []byte, bp int) (int, int, bool) {
	b := bit(d, bp)
	if b < 0 {
		return -1, bp, false
	}
	bp++
	if b == 0 {
		v, nbp := bits(d, bp, 5)
		return v, nbp, true
	}
	return -1, bp, false // absent
}

// decodeRecord : tente 1 record kill-event à bp. Retourne (ok, newBp, victim, killer, assist).
func decodeRecord(d []byte, bp, maxIdx int, withTail bool) (bool, int, int, int, int) {
	v, bp, pv := readOpt5(d, bp)
	if !pv || v >= maxIdx {
		return false, bp, 0, 0, 0
	}
	k, bp, pk := readOpt5(d, bp)
	if !pk || k >= maxIdx || k == v {
		return false, bp, 0, 0, 0
	}
	_, bp = bits(d, bp, 32) // u32a
	if bp < 0 {
		return false, bp, 0, 0, 0
	}
	bb := bit(d, bp) // bool R(1)
	if bb < 0 {
		return false, bp, 0, 0, 0
	}
	bp++
	a, bp, _ := readOpt5(d, bp) // assistant (peut être absent)
	if a >= maxIdx {            // si présent, doit être valide
		return false, bp, 0, 0, 0
	}
	_, bp = bits(d, bp, 32) // u32b
	if bp < 0 || bp>>3 >= len(d) {
		return false, bp, 0, 0, 0
	}
	if withTail {
		// tail = 2×R(32) + R(4) (cas commun deser FUN_1431eb378), si feature-gated présent
		_, bp = bits(d, bp, 32)
		_, bp = bits(d, bp, 32)
		_, bp = bits(d, bp, 4)
		if bp < 0 || bp>>3 >= len(d) {
			return false, bp, 0, 0, 0
		}
	}
	return true, bp, v, k, a
}

type run struct {
	chunk, bp, length int
}

func main() {
	maxIdx := 16
	minRun := 8
	withTail := false
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxIdx)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &minRun)
	}
	if len(os.Args) >= 4 {
		var t int
		fmt.Sscanf(os.Args[3], "%d", &t)
		withTail = t != 0
	}
	fmt.Printf("=== scan kill-event : maxIdx=%d minRun=%d withTail=%v ===\n", maxIdx, minRun, withTail)

	var runs []run
	globalBest := run{}
	for c := 0; c <= 27; c++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		total := len(d) * 8
		chunkBest := 0
		for start := 0; start < total-78; start++ {
			// run depuis start
			bp := start
			n := 0
			for {
				ok, nbp, _, _, _ := decodeRecord(d, bp, maxIdx, withTail)
				if !ok || nbp <= bp {
					break
				}
				n++
				bp = nbp
				if n > 200 {
					break
				}
			}
			if n >= minRun {
				runs = append(runs, run{c, start, n})
			}
			if n > chunkBest {
				chunkBest = n
			}
			if n > globalBest.length {
				globalBest = run{c, start, n}
			}
		}
		if chunkBest >= 4 {
			fmt.Printf("  chunk%02d (%d o) : meilleur run = %d records\n", c, len(d), chunkBest)
		}
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].length > runs[j].length })
	fmt.Printf("\n=== %d runs >= %d ; meilleur global = %d records (chunk%02d bit%d) ===\n",
		len(runs), minRun, globalBest.length, globalBest.chunk, globalBest.bp)
	for i, r := range runs {
		if i >= 15 {
			fmt.Printf("  ... (%d runs au total)\n", len(runs))
			break
		}
		fmt.Printf("  chunk%02d bit%-9d : %d records\n", r.chunk, r.bp, r.length)
	}

	// dump du meilleur run : histogrammes killer/victim/assist
	if globalBest.length >= minRun {
		fmt.Printf("\n=== DUMP meilleur run (chunk%02d bit%d, %d records) ===\n", globalBest.chunk, globalBest.bp, globalBest.length)
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, globalBest.chunk))
		bp := globalBest.bp
		killer := map[int]int{}
		victim := map[int]int{}
		assist := map[int]int{}
		nAssist := 0
		for i := 0; i < globalBest.length; i++ {
			ok, nbp, v, k, a := decodeRecord(d, bp, maxIdx, withTail)
			if !ok {
				break
			}
			victim[v]++
			killer[k]++
			if a >= 0 {
				assist[a]++
				nAssist++
			}
			if i < 30 {
				fmt.Printf("  rec%-3d victim=%-2d killer=%-2d assist=%-3d\n", i, v, k, a)
			}
			bp = nbp
		}
		fmt.Printf("\n  histogramme KILLER (idx->count) : %s\n", histo(killer))
		fmt.Printf("  histogramme VICTIME (idx->count): %s\n", histo(victim))
		fmt.Printf("  histogramme ASSIST  (idx->count): %s  (%d records avec assist)\n", histo(assist), nAssist)
		fmt.Println("\n  >>> comparer aux comptes DB : kills=[14,14,14,13,12,10,8,8] deaths=[13,9,13,9,10,11,14,14] assists total=17")
	} else {
		fmt.Println("\n>>> Aucun run significatif. La liste kill-event n'est pas contiguë à ces contraintes (essayer maxIdx 8/32, withTail, ou records non contigus).")
	}
}

func histo(m map[int]int) string {
	type kv struct{ k, v int }
	var a []kv
	for k, v := range m {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].v != a[j].v {
			return a[i].v > a[j].v
		}
		return a[i].k < a[j].k
	})
	s := ""
	for _, e := range a {
		s += fmt.Sprintf("%d:%d ", e.k, e.v)
	}
	return s
}
