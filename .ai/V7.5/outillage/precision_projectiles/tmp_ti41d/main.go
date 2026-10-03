// Commande tmp_ti41d — TEST T1' : les entites `ti=41` (PROJECTILE) dans le flux DELTA.
//
// T1 (keyframes) a montre que le projectile EST une entite repliquee, mais un keyframe tombe
// toutes les ~20 s et un projectile vit une fraction de seconde : le compte y est un artefact
// d echantillonnage. La trajectoire, si elle existe, est dans le flux DELTA. C est ce que ce
// test mesure.
//
// TOUT EST REUTILISE, RIEN N EST REECRIT : `ParseRegistryChunk` pour le registre,
// `WalkKeyframeWorld` + `World.BindFull` pour le binding (meme regle que `killsource`),
// `DecodeFrameInfer` pour la marche — cette derniere INFERE l archetype des entites
// transitoires absentes du binding, ce qui est exactement le cas du projectile.
//
// ⚠ CE QUE CE CHIFFRE EST : une BORNE INFERIEURE. `DecodeFrameInfer` demarre au bit 0 du
// payload ; les paquets qui portent une liste d evenements avant la boucle de records
// desynchronisent tot (c est pour cela que `killsource` a un localisateur, qui n est pas
// exporte). Un ti=41 trouve est donc un vrai ti=41 ; un film a zero ne prouve rien.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const projectileTI = 41

func main() {
	films := flag.String("films", "", "racine du cache de films (lecture seule)")
	limit := flag.Int("limit", 6, "nombre maximum de films")
	flag.Parse()
	if *films == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_ti41d -films <dir> [-limit N]")
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

	totByTI := map[uint32]int{}
	slotByTI := map[uint32]map[uint32]bool{}
	var filmsOK int
	for _, d := range dirs {
		dir := filepath.Join(*films, d)
		raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
		if err != nil {
			continue
		}
		reg, err := filmdec.ParseRegistryChunk(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s : registre illisible (%v)\n", d, err)
			continue
		}
		w := filmdec.NewWorld(reg)
		cfg := filmdec.DefaultFrameConfig()

		n := filmdec.CountFilmChunks(dir)
		var deltaPkts, recs41, recsTot int
		local41 := map[uint32]bool{}
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range filmdec.WalkPackets(chunk) {
				pay := p.Payload(chunk)
				if p.Type == filmdec.PacketTypeKeyframe {
					for _, r := range filmdec.WalkKeyframeWorld(pay) {
						w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
					}
					continue
				}
				if p.Type != filmdec.PacketTypeDelta || len(pay) < 2 {
					continue
				}
				deltaPkts++
				rs, _ := filmdec.DecodeFrameInfer(pay, w, cfg)
				for _, r := range rs {
					recsTot++
					ti := r.TypeIndex
					if ti == 0 {
						if a, ok := w.ArchetypeForSlot(r.Slot); ok {
							ti = a
						}
					}
					totByTI[ti]++
					if slotByTI[ti] == nil {
						slotByTI[ti] = map[uint32]bool{}
					}
					slotByTI[ti][r.Slot] = true
					if ti == projectileTI {
						recs41++
						local41[r.Slot] = true
					}
				}
			}
		}
		filmsOK++
		fmt.Fprintf(os.Stderr, "%s  paquets delta=%5d  records=%7d  ti=41 : %5d records / %4d slots\n",
			d, deltaPkts, recsTot, recs41, len(local41))
	}

	type line struct {
		ti    uint32
		n     int
		slots int
	}
	var ls []line
	for ti, n := range totByTI {
		ls = append(ls, line{ti, n, len(slotByTI[ti])})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })
	fmt.Printf("\nfilms lus : %d — FLUX DELTA (borne inferieure)\n\n", filmsOK)
	fmt.Printf("%-10s %12s %12s\n", "archetype", "records", "slots distincts")
	for i, l := range ls {
		if i >= 12 && l.ti != projectileTI {
			continue
		}
		mark := ""
		if l.ti == projectileTI {
			mark = "  <- PROJECTILE"
		}
		fmt.Printf("ti=%-7d %12d %12d%s\n", l.ti, l.n, l.slots, mark)
	}
}
