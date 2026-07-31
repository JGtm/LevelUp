// tmp_widthcmp2 — confrontation AFFINEE largeur VRAIE / largeur consommee par NOS desers,
// avec (a) le vrai param_4 par composant (colonne param4 de la capture CE), (b) la part de
// records ou chaque composant est PRESENT, (c) P(align) = probabilite que notre deser
// consomme exactement la largeur vraie. THROWAWAY.
//
// P(align) est LA mesure utile : meme si le curseur arrive au BON bit, un deser dont la
// grammaire fait dependre la largeur de bits qui n'en decident pas dans le jeu ne consomme
// la bonne largeur qu'une fraction du temps. Sur bits aleatoires, P(align) est exactement
// cette fraction. Un composant a P(align) faible ET a forte presence casse le chainage.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_widthcmp2 [csv] [filmdir]
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
	trials  = 4096
	bufLen  = 256
)

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
	arch, _ := reg.Archetype(35)

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

	truth := map[int]map[int]int{}
	p4mode := map[int]int{}
	p4h := map[int]map[int]int{}
	presence := map[int]int{}
	countHist := map[int]int{}
	bipedRecs := 0
	var recCur []hit
	flush := func() {
		if len(recCur) > 0 && recCur[0].ti == 35 {
			bipedRecs++
			countHist[len(recCur)]++
			for i, h := range recCur {
				presence[h.ci]++
				if p4h[h.ci] == nil {
					p4h[h.ci] = map[int]int{}
				}
				p4h[h.ci][h.p4]++
				if i+1 < len(recCur) {
					if truth[h.ci] == nil {
						truth[h.ci] = map[int]int{}
					}
					truth[h.ci][recCur[i+1].cur-h.cur]++
				}
			}
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
	for ci, h := range p4h {
		v, _, _ := modeOf(h)
		p4mode[ci] = v
	}

	fmt.Printf("records BIPEDE (ti=35) segmentes : %d\n", bipedRecs)
	fmt.Println("\n=== TAILLE DU MASQUE (nb de composants presents) — verite terrain ===")
	var cnts []int
	for k := range countHist {
		cnts = append(cnts, k)
	}
	sort.Ints(cnts)
	for _, k := range cnts {
		fmt.Printf("   %2d composants : %7d  (%.2f %%)\n", k, countHist[k], 100*float64(countHist[k])/float64(bipedRecs))
	}

	// ---- notre deser : histogramme + P(align) au vrai param_4 ----
	rng := rand.New(rand.NewSource(20260726))
	bufs := make([][]byte, trials)
	for i := range bufs {
		bufs[i] = make([]byte, bufLen)
		rng.Read(bufs[i])
	}
	type ourStat struct {
		hist   map[int]int
		ported bool
	}
	ours := map[int]ourStat{}
	for i, name := range arch.Components {
		filmdec.SetRecordStateParam(uint32(p4mode[i])) // vrai param_4 mesure pour CE composant
		st := ourStat{hist: map[int]int{}, ported: true}
		for _, b := range bufs {
			w, ported := filmdec.ProbeComponentConsumedWidth(name, 35, arch.Level(i), b)
			if !ported {
				st.ported = false
				break
			}
			st.hist[w]++
		}
		ours[i] = st
	}
	filmdec.SetRecordStateParam(0)

	var idx []int
	for ci := range truth {
		idx = append(idx, ci)
	}
	sort.Ints(idx)

	fmt.Println("\n=== CONFRONTATION (archetype bipede ti=35) ===")
	fmt.Printf("%-4s %-42s %7s %6s %6s %5s %8s %6s  %s\n",
		"i", "composant", "presence", "nVrai", "vrai", "part", "P(align)", "p4", "nos largeurs")
	fmt.Println(strings.Repeat("-", 150))
	for _, ci := range idx {
		name := arch.Components[ci]
		tw, tn, ttot := modeOf(truth[ci])
		st := ours[ci]
		pal, desc := -1.0, "-"
		if st.ported {
			tot := 0
			for _, v := range st.hist {
				tot += v
			}
			pal = float64(st.hist[tw]) / float64(tot)
			desc = topOf(st.hist, 4)
		}
		ps := ""
		if pal >= 0 {
			ps = fmt.Sprintf("%6.1f%%", 100*pal)
		} else {
			ps = "  NPORT"
		}
		fmt.Printf("i%02d  %-42s %6.2f%% %6d %6d %4.0f%% %8s %6d  %s\n",
			ci, trunc(name, 42), 100*float64(presence[ci])/float64(bipedRecs),
			ttot, tw, 100*float64(tn)/float64(ttot), ps, p4mode[ci], trunc(desc, 40))
	}
}

func modeOf(h map[int]int) (val, n, tot int) {
	first := true
	for k, v := range h {
		tot += v
		if first || v > n || (v == n && k < val) {
			val, n, first = k, v, false
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
