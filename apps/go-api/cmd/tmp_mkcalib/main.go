// tmp_mkcalib — THROWAWAY : génère une calibration de largeurs (nom_composant:largeur)
// depuis la capture CE filmdec_delta.csv + le registre. Résout (ti,ci)->nom via le
// registre, agrège par NOM la largeur modale (les largeurs sont universelles par type
// de composant). Sert à forcer l'alignement (skip à la largeur mesurée) et supprimer
// la cascade, pour prouver que le dead-state décode quand l'amont est exact.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_mkcalib <csv> <out_calib.txt>
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

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_mkcalib <csv> <out.txt>")
		return
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("csv: %v\n", err)
		return
	}
	defer f.Close()

	// (ti,ci) -> width -> count, en diffant les bitCursor consécutifs d'un même record.
	type key struct{ ti, ci int }
	width := map[key]map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	type row struct{ eid, ti, ci, cur int }
	var prev *row
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		p := strings.Split(line, ",")
		if len(p) < 6 {
			continue
		}
		eid, _ := strconv.Atoi(p[0])
		ti, _ := strconv.Atoi(p[1])
		ci, _ := strconv.Atoi(p[2])
		cur, _ := strconv.Atoi(p[4])
		r := row{eid, ti, ci, cur}
		if prev != nil && prev.eid == r.eid && prev.ti == r.ti && r.cur > prev.cur && r.ci > prev.ci {
			w := r.cur - prev.cur
			if w >= 0 && w < 4000 {
				k := key{prev.ti, prev.ci}
				if width[k] == nil {
					width[k] = map[int]int{}
				}
				width[k][w]++
			}
		}
		rr := r
		prev = &rr
	}

	// résoudre (ti,ci)->nom ; agréger par nom la largeur modale (somme des votes).
	byName := map[string]map[int]int{}
	for k, m := range width {
		arch, ok := reg.Archetype(k.ti)
		if !ok || k.ci >= len(arch.Components) {
			continue
		}
		name := arch.Components[k.ci]
		if byName[name] == nil {
			byName[name] = map[int]int{}
		}
		for w, n := range m {
			byName[name][w] += n
		}
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Printf("out: %v\n", err)
		return
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	fmt.Fprintf(w, "# calibration largeurs (nom:largeur_modale) ; %d composants\n", len(byName))
	var names []string
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		m := byName[name]
		bestW, bestN := -1, 0
		for ww, n := range m {
			if n > bestN {
				bestN, bestW = n, ww
			}
		}
		// confiance = part du modal (variable si < 0.9)
		tot := 0
		for _, n := range m {
			tot += n
		}
		conf := float64(bestN) / float64(tot)
		fmt.Fprintf(w, "%s:%d %.2f\n", name, bestW, conf)
	}
	w.Flush()
	fmt.Printf("%d composants -> %s\n", len(byName), os.Args[2])
}
