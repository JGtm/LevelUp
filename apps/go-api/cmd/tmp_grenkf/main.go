// tmp_grenkf — les identifiants de grenade sont-ils dans l'INVENTAIRE des keyframes, au meme
// titre que les armes ?
//
// CE QUI MOTIVE LA SONDE (intuition utilisateur, 2026-07-26) : « t'as rien trouve en te basant
// dans la section du joueur, du loadout d'arme, un truc approchant ou se trouvent les id des
// grenades ? ». Mesure preliminaire en Python : les 4 identifiants de grenade apparaissent
// 12 fois dans les keyframes et 70 fois dans les paquets delta, TOUJOURS suivis du suffixe
// 0x42C9679F — celui-la meme qui marque une « vraie arme » dans le chemin loadout. Les grenades
// sont donc encodees EXACTEMENT comme les armes.
//
// CE QUE CETTE SONDE TRANCHE : l'archetype du record QUI CONTIENT chaque occurrence.
//
//	ti=35 (bipede)      -> c'est l'inventaire d'un joueur           => exploitable
//	ti=42 (arme au sol) -> c'est une grenade posee par la carte     => autre chose, mais utile
//
// La distinction est decisive : le meme identifiant au meme endroit ne veut pas dire la meme
// chose selon le porteur. Le decodeur d'armes portees ecarte deja ti=42 pour cette raison.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// Identifiants 32 bits des grenades (cf. filmdec/grenade_events.go).
var grenades = map[uint32]string{
	filmdec.GrenadeFragmentation: "Fragmentation",
	filmdec.GrenadePlasma:        "Plasma",
	filmdec.GrenadeShock:         "Dynamo",
	filmdec.GrenadeSpike:         "Spike",
}

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "dossier des chunks")
	flag.Parse()

	n := filmdec.CountFilmChunks(*dir)
	type hit struct {
		chunk, packet, bit int
		name               string
		ti                 int
		recBit, recWidth   int
	}
	var hits []hit
	byTI := map[int]int{}
	kfSeen, recTotal := 0, 0

	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kfSeen++
			pay := p.Payload(chunk)
			recs := filmdec.WalkKeyframeWorld(pay)
			recTotal += len(recs)
			sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })

			total := len(pay)*8 - 32
			for bp := 0; bp <= total; bp++ {
				v := uint32(peek(pay, bp, 32))
				name, ok := grenades[v]
				if !ok {
					continue
				}
				// record CONTENANT : le dernier dont le debut precede l'occurrence.
				ti, rb, rw := -1, -1, 0
				k := sort.Search(len(recs), func(i int) bool { return recs[i].Bit > bp }) - 1
				if k >= 0 {
					ti, rb = recs[k].TI, recs[k].Bit
					if k+1 < len(recs) {
						rw = recs[k+1].Bit - recs[k].Bit
					}
				}
				byTI[ti]++
				hits = append(hits, hit{c, p.Index, bp, name, ti, rb, rw})
			}
		}
	}

	fmt.Printf("GRENADES DANS LES KEYFRAMES — %s\n\n", *dir)
	fmt.Printf("  %d keyframes parcourus, %d records delimites\n", kfSeen, recTotal)
	fmt.Printf("  %d occurrences d'identifiant de grenade\n\n", len(hits))

	fmt.Println("  par archetype du record porteur :")
	tis := make([]int, 0, len(byTI))
	for k := range byTI {
		tis = append(tis, k)
	}
	sort.Ints(tis)
	for _, ti := range tis {
		lbl := ""
		switch ti {
		case 35:
			lbl = "  <-- BIPEDE : inventaire d'un joueur"
		case 42:
			lbl = "  <-- arme au sol : pose par la carte"
		case -1:
			lbl = "  <-- hors de tout record delimite"
		}
		fmt.Printf("    ti=%3d : %3d%s\n", ti, byTI[ti], lbl)
	}

	fmt.Println("\n  detail :")
	for i, h := range hits {
		if i >= 40 {
			fmt.Printf("    ... et %d autres\n", len(hits)-40)
			break
		}
		fmt.Printf("    chunk %2d paquet %4d bit %7d  %-14s ti=%3d (record @%d, %d bits)\n",
			h.chunk, h.packet, h.bit, h.name, h.ti, h.recBit, h.recWidth)
	}
	if len(hits) == 0 {
		fmt.Fprintln(os.Stderr, "aucune occurrence — verifier le dossier du film")
	}
}

// peek lit n bits MSB-first a la position bp (les bits hors tampon valent 0).
func peek(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}
