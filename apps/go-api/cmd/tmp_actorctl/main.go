// cmd/tmp_actorctl — LE HANDLE DE `unit-actor-control-component` DESIGNE-T-IL LE JOUEUR ?
//
// D'OU VIENT CETTE PISTE. Ghidra, sur demande de l'utilisateur. Deux lectures decisives :
//
//	FUN_14111ec64  (player-representation-component, i21 du JOUEUR)
//	  -> FUN_14080dec4(param_2, "representation-name", ...)
//	  C'est un NOM, pas une entite. L'etape 1 du plan cherchait donc la mauvaise chose, et
//	  c'est pourquoi l'ancrage par signature ne trouvait rien : il n'y a pas de handle a
//	  trouver dans ce composant.
//
//	FUN_1408f0778  (unit-actor-control-component, i19 du BIPED)
//	  R(3) sel ; si sel==1 -> fin
//	  R(1) present ; si pose -> FUN_141015740 = R(32)   <- UN HANDLE COMPLET
//	  ...
//	  « actor control » sur une unite : qui la pilote. Candidat direct au lien biped -> joueur.
//
// CE QUE CE PROGRAMME MESURE, sans rien modifier en production. Le composant est deja porte
// (`consumeUnitActorControl`) mais sa valeur est jetee. On relit donc les 32 bits a la position
// du composant — `CompResult.StartBit` + 4 bits d'en-tete (R(3) sel + R(1) present) — et on
// regarde ou pointe le handle.
//
// LE CRITERE, ecrit AVANT la mesure : les entites JOUEUR de ce film occupent les slots 52 a 83
// (recensement de cmd/tmp_playerrep, 32 entites, 8 actives en 52..59). Si le handle designe le
// joueur, `handle & 0x3fffffff` doit tomber dans cette plage — et pour les bipeds vivants,
// dans 52..59. S'il tombe ailleurs, la piste est refutee et il faut le dire.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	bipedTI     = 35
	playerSlotL = 52 // borne basse des entites JOUEUR, mesuree sur ce film
	playerSlotH = 83
)

func main() {
	repo := flag.String("repo", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, "racine des films")
	match := flag.String("match", "000d5950", "match")
	seed := flag.Int("seed", 2, "chunk dont l'image-cle seme le monde")
	flag.Parse()
	dir := filepath.Join(*repo, "data", "cache", "film_chunks", *match)

	reg, err := filmdec.ParseRegistryChunk(mustChunk(dir, 0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "registre:", err)
		os.Exit(1)
	}
	arch, ok := reg.Archetype(bipedTI)
	if !ok {
		fmt.Fprintln(os.Stderr, "archetype biped absent")
		os.Exit(1)
	}
	idx := -1
	for i, c := range arch.Components {
		if c == "unit-actor-control-component" {
			idx = i
		}
	}
	fmt.Printf("=== unit-actor-control-component : rang i%d de l'archetype biped (%d composants)\n\n",
		idx, len(arch.Components))
	if idx < 0 {
		fmt.Println("composant absent de l'archetype : rien a mesurer.")
		return
	}

	w := filmdec.NewWorld(reg)
	kf, err := filmdec.ReadFilmChunk(dir, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chunk de semence:", err)
		os.Exit(1)
	}
	bound := 0
	for _, pk := range filmdec.WalkPackets(kf) {
		if pk.Type != filmdec.PacketTypeKeyframe {
			continue
		}
		for _, b := range filmdec.WalkKeyframeWorld(pk.Payload(kf)) {
			w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
			bound++
		}
		break
	}
	fmt.Printf("monde seme : %d liaisons\n\n", bound)

	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
	n := filmdec.CountFilmChunks(dir)

	var nRec, nReached, nPresent int
	inPlayerRange, inActiveRange := 0, 0
	slotHist := map[int]int{}
	genHist := map[int]int{}
	var samples []string

	for c := *seed + 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				if r.TypeIndex != bipedTI {
					continue
				}
				nRec++
				for _, comp := range r.Trace.Comps {
					if comp.Name != "unit-actor-control-component" {
						continue
					}
					nReached++
					// R(3) sel : si 1, le composant s'arrete la.
					if bits(pay, comp.StartBit, 3) == 1 {
						continue
					}
					// R(1) present : si 0, pas de handle.
					if bits(pay, comp.StartBit+3, 1) == 0 {
						continue
					}
					nPresent++
					h := bits(pay, comp.StartBit+4, 32)
					slot := int(h & 0x3fffffff)
					gen := int(h >> 30)
					genHist[gen]++
					slotHist[slot]++
					if slot >= playerSlotL && slot <= playerSlotH {
						inPlayerRange++
						if slot <= 59 {
							inActiveRange++
						}
					}
					if len(samples) < 15 {
						samples = append(samples, fmt.Sprintf(
							"  biped slot %-5d -> handle 0x%08x  (gen %d, slot %d)", r.Slot, h, gen, slot))
					}
				}
			}
		}
	}

	fmt.Printf("records de biped rencontres        : %d\n", nRec)
	fmt.Printf("i%d atteint                        : %d\n", idx, nReached)
	fmt.Printf("handle PRESENT                     : %d\n\n", nPresent)
	if nPresent == 0 {
		fmt.Println("aucun handle lu : la piste n'est pas evaluable sur ce parcours.")
		return
	}
	fmt.Println("echantillons :")
	for _, s := range samples {
		fmt.Println(s)
	}
	fmt.Printf("\nCRITERE (ecrit avant la mesure) : le slot pointe doit tomber dans 52..83\n")
	fmt.Printf("  dans 52..83 (entites JOUEUR)     : %d / %d (%.1f %%)\n",
		inPlayerRange, nPresent, 100*float64(inPlayerRange)/float64(nPresent))
	fmt.Printf("  dans 52..59 (joueurs ACTIFS)     : %d / %d (%.1f %%)\n",
		inActiveRange, nPresent, 100*float64(inActiveRange)/float64(nPresent))
	fmt.Printf("  generations lues                 : %s\n", fmtMap(genHist))
	fmt.Printf("  slots distincts pointes          : %d\n", len(slotHist))
	fmt.Printf("  les 12 plus frequents            : %s\n", topN(slotHist, 12))
}

func mustChunk(dir string, i int) []byte {
	b, err := filmdec.ReadFilmChunk(dir, i)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chunk", i, ":", err)
		os.Exit(1)
	}
	return b
}

func bits(buf []byte, p, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		q := p + i
		var b uint32
		if idx := q >> 3; idx >= 0 && idx < len(buf) {
			b = uint32(buf[idx]>>(7-uint(q&7))) & 1
		}
		v = v<<1 | b
	}
	return v
}

func fmtMap(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}

func topN(m map[int]int, n int) string {
	type kv struct{ k, v int }
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	s := ""
	for i := 0; i < n && i < len(l); i++ {
		s += fmt.Sprintf("%d(x%d) ", l[i].k, l[i].v)
	}
	return s
}
