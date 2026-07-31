// tmp_widthtruth — TABLE DES LARGEURS VRAIES par composant, tiree de la verite terrain
// live (.ai/re_dump/ce_capture_delta.csv). THROWAWAY.
//
// PRINCIPE (aucune hypothese de grammaire de composant) : le hook CE est pose sur le SEUL
// site de dispatch des desers (FUN_14076cb60 : call [rax+28]) et enregistre le curseur du
// bitreader A L ENTREE de chaque deser. Donc, pour deux composants CONSECUTIFS du meme
// record, largeur(comp N) = curseur(N+1) - curseur(N). Aucun bit n'est consomme entre deux
// desers (le masque est lu integralement dans l'en-tete : mesure tmp_hdrtruth = 21 bits).
//
// Segmentation d'un record : la boucle FUN_14076cb60 parcourt les composants dans l'ordre
// croissant d'index. Un nouvel enregistrement s'ouvre donc des que eid change, OU que
// compIndex ne croit plus, OU que le curseur recule (frontiere de paquet).
//
// Le DERNIER composant d'un record n'a pas de successeur intra-record : sa largeur est
// reconstruite via l'en-tete du record suivant (21 + 6*count si count<=7, sinon 82 = dense).
// Cette voie-la EST une hypothese de grammaire : elle est publiee separement (colonne TAIL)
// et jamais melangee a la mesure directe.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_widthtruth [csv] [filmdir]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defCSV  = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_capture_delta.csv`
	defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	idLow   = 14 // mesure tmp_hdrtruth sur CETTE capture (en-tete eparse = 21 = 7 + idLow)
)

type hit struct{ eid, ti, ci, p4, cur int }

type rec struct {
	ti    int
	comps []hit
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

func main() {
	csvPath, filmDir := defCSV, defFilm
	if len(os.Args) > 1 {
		csvPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		filmDir = os.Args[2]
	}

	// noms de composants : le registre est BIT-A-BIT identique d'un film a l'autre
	// (FNV des 1067 slots verifie, cf. registry.go) -> utilisable pour nommer la capture live.
	var names []string
	if reg, err := filmdec.ParseRegistryChunk(inflate(filmDir + "/chunk_00.bin")); err == nil {
		if a, ok := reg.Archetype(35); ok {
			names = a.Components
		}
	}

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
	fmt.Printf("hits lus : %d\n", len(hits))

	// --- segmentation en records -------------------------------------------------
	var recs []rec
	var cur rec
	flush := func() {
		if len(cur.comps) > 0 {
			recs = append(recs, cur)
		}
		cur = rec{}
	}
	for _, h := range hits {
		if len(cur.comps) > 0 {
			p := cur.comps[len(cur.comps)-1]
			if h.eid != p.eid || h.ci <= p.ci || h.cur <= p.cur {
				flush()
			}
		}
		if len(cur.comps) == 0 {
			cur.ti = h.ti
		}
		cur.comps = append(cur.comps, h)
	}
	flush()
	fmt.Printf("records segmentes : %d\n", len(recs))

	// --- largeurs directes (intra-record, sans hypothese) -------------------------
	type key struct{ ti, ci int }
	direct := map[key]map[int]int{}
	tail := map[key]map[int]int{}
	p4hist := map[key]map[int]int{}
	for ri, r := range recs {
		for i, h := range r.comps {
			k := key{r.ti, h.ci}
			if p4hist[k] == nil {
				p4hist[k] = map[int]int{}
			}
			p4hist[k][h.p4]++
			if i+1 < len(r.comps) {
				if direct[k] == nil {
					direct[k] = map[int]int{}
				}
				direct[k][r.comps[i+1].cur-h.cur]++
				continue
			}
			// dernier composant : reconstruction via l'en-tete du record suivant.
			if ri+1 >= len(recs) {
				continue
			}
			n := recs[ri+1]
			c0 := n.comps[0].cur
			if c0 <= h.cur { // frontiere de paquet : le curseur a recule
				continue
			}
			hdr := 7 + idLow + 6*len(n.comps)
			if len(n.comps) > 7 {
				hdr = 1 + idLow + 2 + 1 + 64
			}
			w := c0 - hdr - h.cur
			if tail[k] == nil {
				tail[k] = map[int]int{}
			}
			tail[k][w]++
		}
	}

	type row struct {
		ci                   int
		name                 string
		n, mode, modeN       int
		top                  string
		tn, tmode, tmodeN    int
		p4mode, p4modeN, p4n int
	}
	modeOf := func(h map[int]int) (int, int, int, string) {
		type kv struct{ k, v int }
		var a []kv
		tot := 0
		for k, v := range h {
			a = append(a, kv{k, v})
			tot += v
		}
		sort.Slice(a, func(i, j int) bool {
			if a[i].v != a[j].v {
				return a[i].v > a[j].v
			}
			return a[i].k < a[j].k
		})
		if len(a) == 0 {
			return 0, 0, 0, ""
		}
		var sb []string
		for i := 0; i < len(a) && i < 4; i++ {
			sb = append(sb, fmt.Sprintf("%d:%.0f%%", a[i].k, 100*float64(a[i].v)/float64(tot)))
		}
		return a[0].k, a[0].v, tot, strings.Join(sb, " ")
	}

	tis := map[int]bool{}
	for k := range direct {
		tis[k.ti] = true
	}
	// on publie l'archetype bipede en priorite, puis les autres archetypes couverts.
	order := []int{35}
	var rest []int
	for ti := range tis {
		if ti != 35 {
			rest = append(rest, ti)
		}
	}
	sort.Ints(rest)
	order = append(order, rest...)
	for _, ti := range order {
		if !tis[ti] {
			continue
		}
		names := names
		if a, ok := regArch(filmDir, ti); ok {
			names = a
		}
		var rows []row
		for k, h := range direct {
			if k.ti != ti {
				continue
			}
			m, mn, tot, top := modeOf(h)
			r := row{ci: k.ci, mode: m, modeN: mn, n: tot, top: top}
			if k.ci < len(names) {
				r.name = names[k.ci]
			}
			if th, ok := tail[k]; ok {
				r.tmode, r.tmodeN, r.tn, _ = modeOf(th)
			}
			if ph, ok := p4hist[k]; ok {
				r.p4mode, r.p4modeN, r.p4n, _ = modeOf(ph)
			}
			rows = append(rows, r)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ci < rows[j].ci })
		fmt.Printf("\n=== VERITE TERRAIN — LARGEURS PAR COMPOSANT, archetype ti=%d ===\n", ti)
		fmt.Printf("%-4s %-46s %6s %6s %7s  %-30s %6s %6s %7s %5s\n",
			"i", "composant", "nDir", "mode", "part", "top valeurs (directes)", "nTail", "tMode", "tPart", "p4")
		for _, r := range rows {
			tp := ""
			if r.tn > 0 {
				tp = fmt.Sprintf("%.0f%%", 100*float64(r.tmodeN)/float64(r.tn))
			}
			p4 := ""
			if r.p4n > 0 {
				p4 = fmt.Sprintf("%d/%.0f%%", r.p4mode, 100*float64(r.p4modeN)/float64(r.p4n))
			}
			fmt.Printf("i%02d  %-46s %6d %6d %6.1f%%  %-30s %6d %6d %7s %5s\n",
				r.ci, trunc(r.name, 46), r.n, r.mode, 100*float64(r.modeN)/float64(r.n), r.top,
				r.tn, r.tmode, tp, p4)
		}
	}

	// resume des autres archetypes (pour situer la couverture)
	fmt.Println("\n=== autres archetypes (couverture) ===")
	var others []int
	for ti := range tis {
		if ti != 35 {
			others = append(others, ti)
		}
	}
	sort.Ints(others)
	for _, ti := range others {
		n := 0
		for k, h := range direct {
			if k.ti == ti {
				for _, v := range h {
					n += v
				}
			}
		}
		fmt.Printf("  ti=%-3d  %d mesures directes\n", ti, n)
	}
}

var regCache *filmdec.Registry

// regArch rend la liste ordonnee des composants d'un archetype (registre film-invariant).
func regArch(filmDir string, ti int) ([]string, bool) {
	if regCache == nil {
		r, err := filmdec.ParseRegistryChunk(inflate(filmDir + "/chunk_00.bin"))
		if err != nil {
			return nil, false
		}
		regCache = r
	}
	a, ok := regCache.Archetype(ti)
	if !ok {
		return nil, false
	}
	return a.Components, true
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
