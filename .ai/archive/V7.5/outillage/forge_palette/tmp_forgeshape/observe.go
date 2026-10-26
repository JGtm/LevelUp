package main

// observe.go — la table d'OBSERVATION.
//
// Correction de protocole apportee par l'utilisateur le 2026-08-02 : les socles
// d'emplacement sont visuellement GENERIQUES. Demander « lequel de ces deux
// socles est-ce ? » n'a pas de reponse en jeu. Ce qui s'observe, c'est
// l'objet qui apparait DESSUS et sa cadence de reapparition.
//
// On sort donc, groupe par `Representation Name` (l'identifiant stable et
// encore non nomme), toutes les positions ou l'observer nommerait ce hachage
// d'un coup — carte, position, cadence, famille de degat.
//
//	observe <cls_all.csv> <mvar...>            toutes les cartes
//	observe <cls_all.csv> --seules <liste> <mvar...>   filtre par nom de fichier
//
// La signature d'emprise 0.1306/0.1308/0.2617 est le predicat de selection
// (cf. etat de l'art Q1.0-octies) : elle capte les cinq types d'emplacement et
// eux seuls, sans liste de type_id a maintenir.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// emplacementSig — l'emprise du modele de socle, au dix-millieme.
const emplacementSig = "0.1306/0.1308/0.2617"

type spot struct {
	file      string
	typeID    int32
	pos       mapvar.Vec3
	respawn   int
	variant   int32
	wallMount bool
}

func cmdObserve(paths []string, palCSV string) {
	pal := loadPalette(palCSV)
	byRep := map[int32][]spot{}

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
			e, known := pal[o.TypeID]
			if !known || footprint(e) != emplacementSig {
				continue
			}
			rep, ok := representationName(objs.Items[i])
			if !ok {
				continue
			}
			s := spot{file: base, typeID: o.TypeID, pos: o.Pos, wallMount: o.Up.Z < 0.35}
			if cv, ok := crateVariant(objs.Items[i]); ok {
				s.variant = cv
			}
			if bag, ok := objs.Items[i].Field(8); ok {
				if lst, ok := bag.Field(1); ok && len(lst.Items) > 0 {
					if d, ok := lst.Items[0].Field(4); ok {
						s.respawn = int(d.Uint)
					}
				}
			}
			byRep[rep] = append(byRep[rep], s)
		}
	}

	reps := make([]int32, 0, len(byRep))
	for r := range byRep {
		reps = append(reps, r)
	}
	sort.Slice(reps, func(i, j int) bool {
		return len(byRep[reps[i]]) > len(byRep[reps[j]])
	})

	fmt.Printf("%d Representation Name distincts sur les emplacements\n\n", len(reps))
	for _, r := range reps {
		spots := byRep[r]
		// Doublons de position : un meme emplacement est souvent decrit par deux
		// objets superposes. On regroupe au decimetre pour ne pas faire croire a
		// deux socles la ou il n'y en a qu'un.
		seen := map[string]bool{}
		uniq := spots[:0:0]
		for _, s := range spots {
			k := fmt.Sprintf("%s|%.1f|%.1f|%.1f", s.file, s.pos.X, s.pos.Y, s.pos.Z)
			if seen[k] {
				continue
			}
			seen[k] = true
			uniq = append(uniq, s)
		}
		maps := map[string]bool{}
		for _, s := range uniq {
			maps[s.file] = true
		}
		fmt.Printf("=== Representation Name %d — %d socles sur %d cartes ===\n",
			r, len(uniq), len(maps))
		sort.Slice(uniq, func(i, j int) bool {
			if uniq[i].file != uniq[j].file {
				return uniq[i].file < uniq[j].file
			}
			return uniq[i].pos.Z > uniq[j].pos.Z
		})
		fmt.Printf("  %-38s %-8s %-8s %-8s %-9s %-20s %s\n",
			"carte", "x", "y", "z", "cadence", "famille de degat", "pose")
		for _, s := range uniq {
			pose := "au sol"
			if s.wallMount {
				pose = "AU MUR"
			}
			cad := "(aucune)"
			if s.respawn > 0 {
				cad = fmt.Sprintf("%d s", s.respawn)
			}
			fmt.Printf("  %-38s %-8.1f %-8.1f %-8.1f %-9s %-20s %s\n",
				strings.TrimSuffix(s.file, ".mvar"), s.pos.X, s.pos.Y, s.pos.Z,
				cad, variantNames[s.variant], pose)
		}
		fmt.Println()
	}
}
