package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// delta.go — PASSE 1.2 / 1.3 du plan, sur le BON flux.
//
// POURQUOI PAS LES IMAGES-CLES. Le mode `kfchain` mesure la chaine de composants sur les
// records d'image-cle : elle n'atterrit sur la borne du record suivant pour AUCUN des 832
// records ti=5. C'est la meme refutation que celle deja consignee en tete de
// keyframe_loadout.go (« le masque lu a cette position n'est pas le vrai masque »).
// L'image-cle a une grammaire de record differente, non decodee a ce jour.
//
// LE FLUX QUI ATTEINT ti=5 est le flux DELTA des paquets de trame : pas de typeIndex, pas
// de default-state, seulement masque + composants presents, l'archetype venant du monde
// seme par une image-cle. C'est le chemin qui a livre le compteur de reapparition (ti=5
// i1) — donc un chemin deja valide sur cet archetype, pas une hypothese neuve.

// seedChunk : le chunk dont l'image-cle seme les liaisons slot -> archetype. Le parcours
// sequentiel commence au chunk suivant. Protocole publie de cmd/tmp_l0witness, repris tel
// quel par cmd/tmp_vitals : une liaison prise ailleurs desynchronise toute la trame.
const seedChunk = 2

// idLowBits est une valeur de RUNTIME (11 sur 000d5950, 14 sur le film de la capture live).
const idLowBits = 11

// seedWorld construit le monde a partir de l'image-cle du chunk de semence.
func seedWorld(dir string, reg *filmdec.Registry) (*filmdec.World, int, error) {
	w := filmdec.NewWorld(reg)
	kfChunk, err := filmdec.ReadFilmChunk(dir, seedChunk)
	if err != nil {
		return nil, 0, fmt.Errorf("chunk de semence %d : %w", seedChunk, err)
	}
	bound := 0
	for _, pk := range filmdec.WalkPackets(kfChunk) {
		if pk.Type != filmdec.PacketTypeKeyframe {
			continue
		}
		for _, b := range filmdec.WalkKeyframeWorld(pk.Payload(kfChunk)) {
			w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
			bound++
		}
		break
	}
	if bound == 0 {
		return nil, 0, fmt.Errorf("aucune liaison dans l'image-cle du chunk %d", seedChunk)
	}
	return w, bound, nil
}

// walkDeltas parcourt les paquets delta du film et appelle fn sur chaque record. Le
// payload est passe au rappel : les positions bit des composants (CompResult.StartBit) y
// renvoient, ce qui permet de RELIRE la valeur d'un composant que le dispatch traverse
// sans la capturer.
func walkDeltas(dir string, reg *filmdec.Registry, w *filmdec.World,
	fn func(ts uint64, pay []byte, r filmdec.FrameRecord)) {
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLowBits}
	n := filmdec.CountFilmChunks(dir)
	for c := seedChunk + 1; c <= n; c++ {
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
				fn(pk.TimestampUS, pay, r)
			}
		}
	}
}

// runDelta — 1.2 et 1.3 : ou la chaine de l'archetype 5 s'arrete-t-elle, et atteint-elle i21 ?
func runDelta(dir string) {
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "registre illisible: %v\n", err)
		os.Exit(1)
	}
	arch, ok := reg.Archetype(playerTI)
	if !ok {
		fmt.Fprintf(os.Stderr, "archetype %d absent du registre\n", playerTI)
		os.Exit(1)
	}
	w, bound, err := seedWorld(dir, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("=== 1.2/1.3 CHAINE DE COMPOSANTS ti=%d sur le flux DELTA\n", playerTI)
	fmt.Printf("monde seme par l'image-cle du chunk %d : %d liaisons\n\n", seedChunk, bound)

	var nRec, nClean int
	desyncAt := map[int]int{}        // rang du premier composant non porte
	presence := map[int]int{}        // rang -> nb de records ou le composant est present
	reached := map[int]int{}         // rang -> nb de records ou le composant est ATTEINT (decode)
	widthOf := map[int]map[int]int{} // rang -> largeur -> compte
	slots := map[uint32]int{}
	var rep []repObs
	walkDeltas(dir, reg, w, func(ts uint64, pay []byte, r filmdec.FrameRecord) {
		if r.TypeIndex != playerTI {
			return
		}
		nRec++
		slots[r.Slot]++
		for _, c := range r.Trace.Comps {
			if c.Name != "player-representation-component" {
				continue
			}
			v := bits(pay, c.StartBit, 32)
			rep = append(rep, repObs{ts: ts, pslot: r.Slot, word: v})
		}
		for i := 0; i < 27; i++ {
			if r.Trace.Mask&(uint64(1)<<uint(i)) != 0 {
				presence[i]++
			}
		}
		for k, c := range r.Trace.Comps {
			reached[c.Index]++
			end := r.Trace.EndBit
			if k+1 < len(r.Trace.Comps) {
				end = r.Trace.Comps[k+1].StartBit
			}
			if widthOf[c.Index] == nil {
				widthOf[c.Index] = map[int]int{}
			}
			widthOf[c.Index][end-c.StartBit]++
		}
		if r.Trace.DesyncAt == -1 {
			nClean++
		} else {
			desyncAt[r.Trace.DesyncAt]++
		}
	})

	fmt.Printf("records ti=%d dans le flux delta : %d\n", playerTI, nRec)
	fmt.Printf("records traverses sans desync    : %d (%.1f %%)\n", nClean, pct(nClean, nRec))
	fmt.Printf("slots ti=5 rencontres            : %v\n\n", sortedSlots(slots))

	fmt.Printf("--- ou la chaine s'arrete (rang du premier composant non porte) ---\n")
	var ds []int
	for d := range desyncAt {
		ds = append(ds, d)
	}
	sort.Ints(ds)
	for _, d := range ds {
		name := "<hors archetype>"
		if d >= 0 && d < len(arch.Components) {
			name = arch.Components[d]
		}
		fmt.Printf("  i%-3d %6d records   %s\n", d, desyncAt[d], name)
	}

	fmt.Printf("\n--- 1.2 LARGEURS : presence, atteinte, largeur consommee (rang i0..i26) ---\n")
	fmt.Printf("%-5s %-9s %-9s %-24s %s\n", "rang", "present", "atteint", "largeur (bits:records)", "composant")
	for i := 0; i < len(arch.Components); i++ {
		fmt.Printf("i%-4d %-9d %-9d %-24s %s\n", i, presence[i], reached[i],
			topWidths(widthOf[i], 3), arch.Components[i])
	}
	dumpRepresentation(rep)
}

// repObs est une lecture de player-representation-component (i21) : l'instant, le slot du
// record joueur porteur, et le mot de 32 bits lu.
type repObs struct {
	ts    uint64
	pslot uint32
	word  uint32
}

// dumpRepresentation publie les valeurs d'i21 effectivement lues — 1.4 : la valeur
// designe-t-elle une entite ? Un handle se lit gen = mot>>30, slot = mot & 0x3fffffff.
func dumpRepresentation(rep []repObs) {
	fmt.Printf("\n--- 1.4 VALEUR LUE de player-representation-component (i21) ---\n")
	if len(rep) == 0 {
		fmt.Println("  aucune occurrence atteinte : rien a publier.")
		return
	}
	sort.Slice(rep, func(i, j int) bool { return rep[i].ts < rep[j].ts })
	genDist := map[int]int{}
	zero := 0
	fmt.Printf("%-14s %-8s %-12s %-6s %s\n", "t(us)", "joueur", "mot", "gen", "slot")
	for _, o := range rep {
		gen := int(o.word >> 30)
		slot := int(o.word & 0x3fffffff)
		genDist[gen]++
		if o.word == 0 {
			zero++
		}
		fmt.Printf("%-14d %-8d 0x%08x   %-6d %d\n", o.ts, o.pslot, o.word, gen, slot)
	}
	fmt.Printf("\noccurrences : %d ; mots nuls : %d ; generations : %s\n",
		len(rep), zero, fmtCount(genDist))
}

// topWidths rend les trois largeurs les plus frequentes d'un composant.
func topWidths(m map[int]int, n int) string {
	if len(m) == 0 {
		return "-"
	}
	type kv struct{ w, c int }
	var l []kv
	for w, c := range m {
		l = append(l, kv{w, c})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].c != l[j].c {
			return l[i].c > l[j].c
		}
		return l[i].w < l[j].w
	})
	s := ""
	for i := 0; i < n && i < len(l); i++ {
		s += fmt.Sprintf("%d:%d ", l[i].w, l[i].c)
	}
	if len(l) > n {
		s += fmt.Sprintf("(+%d)", len(l)-n)
	}
	return s
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func sortedSlots(m map[uint32]int) []uint32 {
	var ks []uint32
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}
