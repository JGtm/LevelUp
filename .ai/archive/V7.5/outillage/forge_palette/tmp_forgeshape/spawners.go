package main

// spawners.go — la question du rejeu : « comment reconnait-on un emplacement
// sur une carte qu'on n'a jamais vue ? »
//
// Une liste de `type_id` en dur est morte d'avance : `493070541` (Catalyst)
// n'etait pas dans les quatre du handoff, et les auteurs choisissent leurs
// composants selon le theme graphique de la carte. On mesure donc quels
// PREDICATS tiennent, et leur couverture reelle sur les 199 cartes :
//
//	P1  groupe de palette dans {weap, vehi, eqip}
//	P2  porte un delai de reapparition (/#8/#1[]/#4)
//	P3  emprise du modele egale a une signature d'emplacement connue
//
//	spawners <cls_all.csv> <mvar...>

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

type palEntry struct {
	group string
	gid   int32
	dx    float64
	dy    float64
	dz    float64
}

// loadPalette lit cls_all.csv : type_id -> groupe de tag, tag d'objet, emprise.
func loadPalette(path string) map[int32]palEntry {
	out := map[int32]palEntry{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 0; sc.Scan(); i++ {
		if i == 0 {
			continue
		}
		fs := strings.Split(sc.Text(), ",")
		if len(fs) < 6 {
			continue
		}
		id, err := strconv.ParseInt(strings.TrimSpace(fs[0]), 10, 64)
		if err != nil {
			continue
		}
		e := palEntry{group: fs[1]}
		if g, err := strconv.ParseInt(strings.TrimSpace(fs[2]), 10, 64); err == nil {
			e.gid = int32(g)
		}
		e.dx, _ = strconv.ParseFloat(fs[3], 64)
		e.dy, _ = strconv.ParseFloat(fs[4], 64)
		e.dz, _ = strconv.ParseFloat(fs[5], 64)
		out[int32(id)] = e
	}
	return out
}

// footprint rend la signature d'emprise arrondie au dix-millieme — deux
// entrees de palette batties sur le meme modele la partagent exactement.
func footprint(e palEntry) string {
	return fmt.Sprintf("%.4f/%.4f/%.4f", e.dx, e.dy, e.dz)
}

func cmdSpawners(paths []string, palCSV string) {
	pal := loadPalette(palCSV)

	// Signatures d'emprise des emplacements etablis (les 5 types connus).
	knownSig := map[string]bool{}
	for _, id := range []int32{1486653438, -1062552774, 1882451900, 801517767, 493070541} {
		if e, ok := pal[id]; ok {
			knownSig[footprint(e)] = true
		}
	}

	type stat struct {
		objs, withDelay int
		maps            map[string]bool
	}
	byType := map[int32]*stat{}
	sigTypes := map[string]map[int32]bool{}
	mapsWithSpawner := map[string]int{}
	delaysByGroup := map[string]int{}

	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			continue
		}
		objs, ok := root.Field(3)
		if !ok {
			continue
		}
		base := filepath.Base(p)
		for i, o := range v.Objects {
			hasDelay := false
			if bag, ok := objs.Items[i].Field(8); ok {
				if lst, ok := bag.Field(1); ok && len(lst.Items) > 0 {
					_, hasDelay = lst.Items[0].Field(4)
				}
			}
			if !hasDelay {
				continue
			}
			s := byType[o.TypeID]
			if s == nil {
				s = &stat{maps: map[string]bool{}}
				byType[o.TypeID] = s
			}
			s.withDelay++
			s.maps[base] = true
			mapsWithSpawner[base]++
			e := pal[o.TypeID]
			g := e.group
			if g == "" {
				g = "(irresolu)"
			}
			delaysByGroup[g]++
			sig := footprint(e)
			if sigTypes[sig] == nil {
				sigTypes[sig] = map[int32]bool{}
			}
			sigTypes[sig][o.TypeID] = true
		}
	}

	fmt.Printf("cartes balayees : %d ; cartes portant au moins un objet a delai : %d\n",
		len(paths), len(mapsWithSpawner))
	fmt.Printf("type_id distincts portant un delai de reapparition : %d\n\n", len(byType))

	fmt.Println("=== P1+P2 : groupe de palette des objets a delai ===")
	gs := make([]string, 0, len(delaysByGroup))
	for g := range delaysByGroup {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return delaysByGroup[gs[i]] > delaysByGroup[gs[j]] })
	for _, g := range gs {
		fmt.Printf("  %-12s %d objets\n", g, delaysByGroup[g])
	}

	fmt.Println("\n=== P3 : signatures d'emprise, triees par nombre de type_id ===")
	sigs := make([]string, 0, len(sigTypes))
	for s := range sigTypes {
		sigs = append(sigs, s)
	}
	sort.Slice(sigs, func(i, j int) bool {
		if len(sigTypes[sigs[i]]) != len(sigTypes[sigs[j]]) {
			return len(sigTypes[sigs[i]]) > len(sigTypes[sigs[j]])
		}
		return sigs[i] < sigs[j]
	})
	for _, s := range sigs {
		ids := make([]int, 0, len(sigTypes[s]))
		total, nmaps := 0, map[string]bool{}
		for id := range sigTypes[s] {
			ids = append(ids, int(id))
			total += byType[id].withDelay
			for m := range byType[id].maps {
				nmaps[m] = true
			}
		}
		sort.Ints(ids)
		mark := "  "
		if knownSig[s] {
			mark = "->"
		}
		shown := ids
		if len(shown) > 8 {
			shown = shown[:8]
		}
		fmt.Printf("%s %-26s %2d type_id  %5d objets  %3d cartes  %v\n",
			mark, s, len(ids), total, len(nmaps), shown)
	}
	fmt.Println("\n(-> = signature portee par l'un des 5 emplacements deja etablis)")
}
