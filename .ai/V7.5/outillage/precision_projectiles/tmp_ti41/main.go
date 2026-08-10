// Commande tmp_ti41 — TEST T1 de `.ai/V7.5/film_re/NOTE_I0_TI41_POSITION_PROJECTILE.md`.
//
// UNE SEULE QUESTION, ET ELLE PEUT TUER TOUTE LA VOIE : les entites de l archetype 41 (le
// PROJECTILE) sont-elles seulement PRESENTES dans le monde replique du film ? Si le film ne
// porte pas d entite projectile, aucune trajectoire n en sortira, quelle que soit l exactitude
// de son composant i0 — et il n y a rien a porter.
//
// L instrument est `filmdec.WalkKeyframeWorld`, deja valide (249/250 entites, 8/8 bipedes) et
// deja utilise par le decodeur de loadout : on ne reecrit rien.
//
// Lecture seule sur le cache de films, plafonnee par -limit.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// projectileTI est l archetype du projectile (registre `chunk_00`, 22 composants).
const projectileTI = 41

func main() {
	films := flag.String("films", "", "racine du cache de films (lecture seule)")
	limit := flag.Int("limit", 10, "nombre maximum de films")
	top := flag.Int("top", 12, "nombre d archetypes affiches")
	flag.Parse()
	if *films == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_ti41 -films <dir> [-limit N]")
		os.Exit(2)
	}
	entries, err := os.ReadDir(*films)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cache illisible: %v\n", err)
		os.Exit(1)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	if *limit > 0 && len(dirs) > *limit {
		dirs = dirs[:*limit]
	}

	byTI := map[int]int{}
	slotsByTI := map[int]map[int]bool{}
	var films41, filmsTot int
	for _, d := range dirs {
		n := filmdec.CountFilmChunks(filepath.Join(*films, d))
		if n == 0 {
			continue
		}
		filmsTot++
		local := 0
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(filepath.Join(*films, d), c)
			if err != nil {
				continue
			}
			for _, p := range filmdec.WalkPackets(chunk) {
				if p.Type != filmdec.PacketTypeKeyframe {
					continue
				}
				for _, r := range filmdec.WalkKeyframeWorld(p.Payload(chunk)) {
					byTI[r.TI]++
					if slotsByTI[r.TI] == nil {
						slotsByTI[r.TI] = map[int]bool{}
					}
					slotsByTI[r.TI][r.Slot] = true
					if r.TI == projectileTI {
						local++
					}
				}
			}
		}
		if local > 0 {
			films41++
		}
		fmt.Fprintf(os.Stderr, "%s  ti=41 : %d records\n", d, local)
	}

	type line struct{ ti, n, slots int }
	var ls []line
	for ti, n := range byTI {
		ls = append(ls, line{ti, n, len(slotsByTI[ti])})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })

	fmt.Printf("\nfilms lus : %d — films portant au moins un record ti=41 : %d\n\n", filmsTot, films41)
	fmt.Printf("%-10s %12s %12s\n", "archetype", "records", "slots distincts")
	for i, l := range ls {
		if i >= *top && l.ti != projectileTI {
			continue
		}
		mark := ""
		if l.ti == projectileTI {
			mark = "  <- PROJECTILE"
		}
		fmt.Printf("ti=%-7d %12d %12d%s\n", l.ti, l.n, l.slots, mark)
	}
}
