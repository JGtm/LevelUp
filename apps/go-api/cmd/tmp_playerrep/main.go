// cmd/tmp_playerrep — ETAPE 1 du plan PLAN_REJEU_2D_FIABILISATION : LIRE le lien
// joueur -> entite, au lieu de le faire voter.
//
// CE QU'ON CHERCHE. L'archetype 5 est le JOUEUR (27 composants, i0 player-waypoint).
// Son composant i21 `player-representation-component` est, par son nom, l'entite qui
// REPRESENTE ce joueur dans le monde — donc son biped, donc son slot. Le deser est
// deja porte dans traverse.go (FUN_14111ec64 = R(32)) mais la valeur est JETEE.
//
// PASSE 1.1 (ce fichier) : recensement pur. Combien d'entites ti=5 dans les images-cles
// du film, a quels slots, quelles generations, quelle largeur d'emprise. Aucune
// conclusion sans son denominateur.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	playerTI = 5  // archetype JOUEUR (player-*-component)
	bipedTI  = 35 // archetype BIPED (le corps dans le monde)
)

// recSpan est l'emprise bit d'un record de keyframe, bornee par le record suivant.
type recSpan struct {
	slot, ti, gen int
	from, to      int
}

// spans borne chaque record d'un payload de keyframe par le debut du suivant.
func spans(pay []byte) []recSpan {
	recs := filmdec.WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	out := make([]recSpan, 0, len(recs))
	total := len(pay) * 8
	for i, r := range recs {
		to := total
		if i+1 < len(recs) {
			to = recs[i+1].Bit
		}
		out = append(out, recSpan{slot: r.Slot, ti: r.TI, gen: r.Gen, from: r.Bit, to: to})
	}
	return out
}

func bitAt(buf []byte, p int) uint32 {
	if idx := p >> 3; idx >= 0 && idx < len(buf) {
		return uint32(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

func bits(buf []byte, p, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | bitAt(buf, p+i)
	}
	return v
}

// kfView est une image-cle deja decoupee en records, avec son horodatage.
type kfView struct {
	chunk, packet int
	tsUS          uint64
	pay           []byte
	sp            []recSpan
}

// loadKeyframes lit tout le film et rend ses images-cles decoupees en records.
func loadKeyframes(dir string) []kfView {
	n := filmdec.CountFilmChunks(dir)
	var out []kfView
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			out = append(out, kfView{chunk: c, packet: p.Index, tsUS: p.TimestampUS,
				pay: pay, sp: spans(pay)})
		}
	}
	return out
}

// runCensus — PASSE 1.1 : l'archetype 5 est-il instancie, et combien d'entites porte-t-il ?
func runCensus(kfs []kfView) {
	fmt.Printf("=== 1.1 RECENSEMENT — %d images-cles\n\n", len(kfs))
	// slots distincts par archetype, sur tout le film
	slotsByTI := map[int]map[int]int{}
	for _, kf := range kfs {
		for _, s := range kf.sp {
			if slotsByTI[s.ti] == nil {
				slotsByTI[s.ti] = map[int]int{}
			}
			slotsByTI[s.ti][s.slot]++
		}
	}
	for _, ti := range []int{playerTI, bipedTI} {
		m := slotsByTI[ti]
		var ks []int
		for k := range m {
			ks = append(ks, k)
		}
		sort.Ints(ks)
		total := 0
		for _, v := range m {
			total += v
		}
		fmt.Printf("ti=%-2d : %d slots distincts, %d records au total\n", ti, len(ks), total)
		fmt.Printf("        slots = %v\n", ks)
	}
	fmt.Println()

	// emprise des records ti=5, image-cle par image-cle
	fmt.Printf("%-4s %-6s %-10s %-6s %-5s %-5s %-8s\n", "kf", "chunk", "t(us)", "slot", "gen", "ti", "largeur")
	nPlayer := 0
	widths := map[int]int{}
	for i, kf := range kfs {
		for _, s := range kf.sp {
			if s.ti != playerTI {
				continue
			}
			nPlayer++
			w := s.to - s.from
			widths[w]++
			if i < 3 { // detail sur les 3 premieres images-cles seulement
				fmt.Printf("%-4d %-6d %-10d %-6d %-5d %-5d %-8d\n",
					i+1, kf.chunk, kf.tsUS, s.slot, s.gen, s.ti, w)
			}
		}
	}
	fmt.Printf("\nrecords ti=5 sur tout le film : %d\n", nPlayer)
	fmt.Printf("largeurs d'emprise distinctes : %d\n", len(widths))
	var ws []int
	for w := range widths {
		ws = append(ws, w)
	}
	sort.Ints(ws)
	shown := 0
	for _, w := range ws {
		fmt.Printf("   %6d bits : %d records\n", w, widths[w])
		if shown++; shown >= 12 {
			fmt.Printf("   ... (%d largeurs supplementaires)\n", len(ws)-shown)
			break
		}
	}
}

func main() {
	dir := flag.String("dir", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`,
		"repertoire des chunks du film")
	mode := flag.String("mode", "census", "census | kfchain | delta | anchor | solve")
	flag.Parse()

	if *mode == "delta" {
		runDelta(*dir)
		return
	}
	kfs := loadKeyframes(*dir)
	if len(kfs) == 0 {
		fmt.Fprintf(os.Stderr, "aucune image-cle lisible dans %s\n", *dir)
		os.Exit(1)
	}
	switch *mode {
	case "census":
		runCensus(kfs)
	case "solve":
		runSolve(kfs)
	case "anchor":
		runAnchor(kfs)
	case "kfchain":
		runKFChain(kfs, *dir)
	default:
		fmt.Fprintf(os.Stderr, "mode inconnu: %s\n", *mode)
		os.Exit(2)
	}
}
