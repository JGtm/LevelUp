// tmp_kfdecode — décode le BUFFER KEYFRAME capturé en live (keyframe_buffer.bin,
// 11485 o, dumpé depuis le reader du jeu au 1er record du keyframe). C'est l'entrée
// EXACTE que la boucle de records du jeu lit → si ça décode en NEW records avec des
// positions d'entités, on tient le keyframe (état complet, TOUS les joueurs).
//
// Le World démarre VIDE : les records NEW lient les entités (bind au fil). On sweep le
// bit de départ (préambule ~20 bits observé en live : reader+0x2c=20 au 1er record).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfdecode <buffer.bin> [filmDirPourRegistre]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

// dumpRegistryBlock affiche les slots bruts d'un bloc registre (le film est complet ; on
// cherche pourquoi le parse tronque). Slot 260 o = [u32 kind][u32 flags][name ASCII NUL-pad].
func dumpRegistryBlock(raw []byte, block int) {
	fmt.Printf("=== BLOC REGISTRE %d (brut, slots 0..23) — offset %d ===\n", block, block*16640)
	base := block * 16640
	for s := 0; s < 24; s++ {
		off := base + s*260
		if off+260 > len(raw) {
			break
		}
		kind := binary.LittleEndian.Uint32(raw[off:])
		flags := binary.LittleEndian.Uint32(raw[off+4:])
		nraw := raw[off+8 : off+260]
		if z := bytes.IndexByte(nraw, 0); z >= 0 {
			nraw = nraw[:z]
		}
		printable := len(nraw) > 0
		for _, c := range nraw {
			if c < 0x20 || c > 0x7e {
				printable = false
				break
			}
		}
		name := "<vide>"
		if printable {
			name = string(nraw)
		} else if len(nraw) > 0 {
			name = fmt.Sprintf("<non-ascii %x>", raw[off+8:off+8+12])
		}
		fmt.Printf("  slot %2d @+%-6d kind=%-8d flags=%-3d name=%s\n", s, s*260, kind, flags, name)
	}
	fmt.Println()
}

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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
	bufPath := os.Args[1]
	film := defFilm
	if len(os.Args) > 2 {
		film = os.Args[2]
	}
	buf, err := os.ReadFile(bufPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("buffer keyframe = %d octets\n", len(buf))
	reg, err := filmdec.ParseRegistryChunk(inflate(film + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	filmdec.SetRecordStateParam(2)
	filmdec.SetInferChain(true)

	// === DÉROULÉ KEYFRAME (loop manuel record-par-record, tail=1) ===
	// Instrument d'autonomie : décode chaque record NEW (header type+id + TraverseEntity)
	// et s'arrête au 1er blocage en nommant l'archétype + le composant fautif. Guide le
	// portage : chaque archétype débloqué = +1 entité avec sa position.
	// DUMP BRUT du bloc registre 22 : le film est COMPLET (.exe dit ti=22 = 12 composants) ;
	// on cherche pourquoi mon parse n'en sort qu'1 (physics-state).
	dumpRegistryBlock(inflate(film+"/chunk_00.bin"), 22)

	// VÉRIF (signalée par un pair) : le registre chunk_00 est-il garbage au-delà de ti~46 ?
	fmt.Println("=== ARCHÉTYPES parsés (ti : nb composants : 1er nom) ===")
	valid, empty := 0, 0
	for ti := 0; ti <= 70; ti++ {
		arch, ok := reg.Archetype(ti)
		n, first := 0, ""
		if ok {
			n = len(arch.Components)
			if n > 0 {
				first = arch.Components[0]
			}
		}
		if n == 0 {
			empty++
		} else {
			valid++
		}
		if ti <= 40 || ti%2 == 0 || n > 0 {
			fmt.Printf("  ti=%-3d comps=%-3d %s\n", ti, n, first)
		}
	}
	fmt.Printf("-> %d archétypes non-vides, %d vides sur ti 0..70\n\n", valid, empty)

	// SCAN : où record1 (NEW petit slot) commence-t-il, et son ti est-il un archétype VALIDE ?
	// Si le seul E donnant NEW slot1 a un ti vide (63), alors ti=63 est réel-mais-absent du registre.
	// Si un autre E donne NEW slot1 avec un ti VALIDE, record0 a une autre largeur (physics-state mal porté).
	fmt.Println("=== SCAN début record1 (E -> NEW slot/ti/validité) ===")
	for E := 60; E <= 115; E++ {
		br := filmdec.NewBitReader(buf)
		br.SetBitPos(E)
		var typ int
		if br.ReadBit() {
			typ = 3
		} else {
			typ = int(br.ReadBits(2))
		}
		if typ != 1 {
			continue
		}
		slot := int(br.ReadBits(13))
		br.ReadBits(2)
		if slot < 1 || slot > 5 {
			continue
		}
		ti := int(br.ReadBits(6))
		arch, ok := reg.Archetype(ti)
		nc := 0
		if ok {
			nc = len(arch.Components)
		}
		flag := ""
		if nc > 0 {
			flag = "  <-- ti VALIDE"
		}
		fmt.Printf("  E=%-4d NEW slot=%d ti=%-3d comps=%d%s\n", E, slot, ti, nc, flag)
	}
	fmt.Println()

	filmdec.SetNewRecordTailBits(1)
	deroulerKeyframe(buf, reg)
	filmdec.SetNewRecordTailBits(0)
	probeDefaultState(buf, reg, 63) // crack empirique de ti=63 (record1) contre l'oracle

	// bipeds (ti=35) = joueurs+bots+respawns
	bipeds := map[uint32]bool{}

	// sweep du bit de départ.
	for _, skip := range []int{0, 1, 2, 3, 4, 8, 12, 16, 20, 24, 32} {
		for _, idlb := range []int{11, 13} {
			cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idlb}
			w := filmdec.NewWorld(reg)
			br := filmdec.NewBitReader(buf)
			br.Skip(skip)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			nNew, nBip, nClean, nDel := 0, 0, 0, 0
			for _, r := range recs {
				switch r.Type {
				case 1:
					nNew++
					if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && len(arch.Components) == 64 {
						nBip++ // biped a 64 composants
					}
				case 2:
					nDel++
				}
				if r.DesyncAt == -1 {
					nClean++
				}
			}
			fmt.Printf("skip=%-2d idlb=%d -> %d records (%d NEW, %d DEL, %d clean, ~%d biped), bits=%d/%d\n",
				skip, idlb, len(recs), nNew, nDel, nClean, nBip, br.BitPos(), len(buf)*8)
		}
	}
	_ = bipeds

	// ANCRE (validee par le workflow keyframe-deser-re) : preambule 2 bits + idLowBits=13
	// -> record0 = NEW slot0 id0x40000000 fin bit20 (reproduit la capture live). On decode
	// en record-loop depuis bit 2 et on trace OU + sur quel archetype ca desync = la brique
	// a porter (default-state vtable[0x60] par typeIndex, meme mur que les deltas).
	fmt.Println("\n--- ANCRE skip=2 idlb=13 (record-loop depuis bit 2) ---")
	w := filmdec.NewWorld(reg)
	br := filmdec.NewBitReader(buf)
	br.Skip(2)
	filmdec.SetNewRecordTailBits(1) // agent Ghidra : +1 bit terminal per-record NEW aligne record1
	recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{IDLowBits: 13, HasExtraFields: false})
	filmdec.SetNewRecordTailBits(0)
	typeName := map[int]string{0: "END", 1: "NEW", 2: "DEL", 3: "DELTA"}
	nClean := 0
	prevEnd := 2
	for i, r := range recs {
		name := ""
		if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && len(arch.Components) > 0 {
			name = arch.Components[0]
		}
		if r.DesyncAt == -1 {
			nClean++
		}
		end := r.Trace.EndBit
		width := 0
		if end > 0 {
			width = end - prevEnd
			prevEnd = end
		}
		if i < 70 {
			fmt.Printf("  #%-3d %-5s slot=%-6d ti=%-3d desync=%-4d w=%-4d end=%-6d comps=%d %s\n",
				i, typeName[r.Type], r.Slot, r.TypeIndex, r.DesyncAt, width, end, len(r.Trace.Comps), name)
		}
	}
	fmt.Printf("TOTAL %d records, %d clean, fin bit=%d/%d (%d octets)\n",
		len(recs), nClean, br.BitPos(), len(buf)*8, len(buf))
	if len(recs) > 0 {
		r0 := recs[0]
		fmt.Printf("ANCRE record0: type=%s slot=%d id=0x%08x  (attendu: NEW slot0 id0x40000000)\n",
			typeName[r0.Type], r0.Slot, r0.ID)
	}

	// PROBE : le default-state non-biped est Skip(NewDefaultStateBits). On sweepe cette
	// largeur pour trouver celle qui aligne record1+ en NEW à slots CROISSANTS (signal
	// keyframe = full-state all-NEW ascendant). Si un plateau émerge, le modèle record-loop
	// à largeur ~constante tient et on a la largeur du default-state dominant. Sinon =
	// largeurs par-archétype (chaque ti sa largeur) → il faut le deser générique.
	fmt.Println("\n--- PROBE largeur default-state (skip=2, idlb=13) ---")
	bestDsb, bestScore := -1, -1
	scoreOf := func(dsb int) (asc int, total int, firstSlots []uint32) {
		w := filmdec.NewWorld(reg)
		br := filmdec.NewBitReader(buf)
		br.Skip(2)
		recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{IDLowBits: 13, NewDefaultStateBits: dsb})
		total = len(recs)
		prev := int64(-1)
		for _, r := range recs {
			if r.Type == 1 && r.DesyncAt == -1 && int64(r.Slot) > prev {
				asc++
				prev = int64(r.Slot)
				if len(firstSlots) < 8 {
					firstSlots = append(firstSlots, r.Slot)
				}
			} else {
				break
			}
		}
		return
	}
	for dsb := 0; dsb <= 400; dsb++ {
		asc, _, _ := scoreOf(dsb)
		if asc > bestScore {
			bestScore, bestDsb = asc, dsb
		}
	}
	asc, total, slots := scoreOf(bestDsb)
	fmt.Printf("meilleur NewDefaultStateBits=%d -> %d NEW croissants consécutifs (sur %d records)\n",
		bestDsb, asc, total)
	fmt.Printf("  premiers slots croissants: %v\n", slots)

	// PROBE 3 : le keyframe est-il un record-loop à PAS CONSTANT W bits ? (record k = en-tête
	// à bit 2+k*W). On teste tout W et on compte la plus longue chaîne d'en-têtes NEW à slots
	// STRICTEMENT croissants. Chaîne longue -> modèle record-loop confirmé + W = largeur record.
	// Chaîne courte (<=2) -> largeurs variables par-archétype OU presence-word (pas de pas fixe).
	readHdr := func(atBit int) (typ int, slot int) {
		br := filmdec.NewBitReader(buf)
		br.SetBitPos(atBit)
		if br.ReadBit() {
			typ = 3 // DELTA
		} else {
			typ = int(br.ReadBits(2))
		}
		low := int(br.ReadBits(13))
		br.ReadBits(2) // tag
		return typ, low & 0x3fffffff
	}
	fmt.Println("\n--- PROBE 3 : record-loop à pas constant W (en-tête NEW slots croissants @ bit 2+k*W) ---")
	bestW, bestChain, bestSlots := 0, 0, []int{}
	for W := 16; W <= 2300; W++ {
		chain, prev, chSlots := 0, -1, []int{}
		for k := 0; k < 24; k++ {
			typ, slot := readHdr(2 + k*W)
			if typ != 1 || slot <= prev {
				break
			}
			prev = slot
			chain++
			if len(chSlots) < 10 {
				chSlots = append(chSlots, slot)
			}
		}
		if chain > bestChain {
			bestChain, bestW, bestSlots = chain, W, chSlots
		}
	}
	fmt.Printf("meilleur W=%d bits (%.1f o) -> chaîne de %d en-têtes NEW croissants: %v\n",
		bestW, float64(bestW)/8, bestChain, bestSlots)

	// PROBE 4 : sweep du tail terminal per-record NEW. ti=22 default-state=0 bit (agent Ghidra) ;
	// il manque ~1 bit à record0 pour aligner record1. On cherche le tail qui maximise la chaîne
	// de records NEW à slots croissants (each record = archétype potentiellement différent, donc
	// la chaîne est aussi bornée par le portage des composants — mais tail>0 gagnant = tail réel).
	fmt.Println("\n--- PROBE 4 : sweep tail terminal per-record NEW (default-state=0) ---")
	for tail := 0; tail <= 8; tail++ {
		filmdec.SetNewRecordTailBits(tail)
		w := filmdec.NewWorld(reg)
		br := filmdec.NewBitReader(buf)
		br.Skip(2)
		recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{IDLowBits: 13, NewDefaultStateBits: 0})
		a, prev := 0, int64(-1)
		var sl []uint32
		for _, r := range recs {
			if r.Type == 1 && r.DesyncAt == -1 && int64(r.Slot) > prev {
				a++
				prev = int64(r.Slot)
				if len(sl) < 12 {
					sl = append(sl, r.Slot)
				}
			} else {
				break
			}
		}
		fmt.Printf("  tail=%d -> %d NEW croissants (total %d) slots=%v\n", tail, a, len(recs), sl)
	}
	filmdec.SetNewRecordTailBits(0)
}

// deroulerKeyframe décode le keyframe record-par-record (le tail est réglé par l'appelant) et
// affiche chaque NEW + le 1er blocage (archétype/composant fautif) — l'instrument de portage.
func deroulerKeyframe(buf []byte, reg *filmdec.Registry) {
	fmt.Println("=== DÉROULÉ KEYFRAME (record-par-record, tail=1) ===")
	tn := map[int]string{0: "END", 1: "NEW", 2: "DEL", 3: "DELTA"}
	br := filmdec.NewBitReader(buf)
	br.Skip(2) // préambule 2 bits
	nOK := 0
	for k := 0; k < 3000; k++ {
		var typ int
		if br.ReadBit() {
			typ = 3
		} else {
			typ = int(br.ReadBits(2))
		}
		if typ == 0 {
			fmt.Printf("  #%d END @bit%d — fin\n", k, br.BitPos())
			break
		}
		slot := int(br.ReadBits(13)) // low ; tag R(2) masqué hors slot
		br.ReadBits(2)
		if typ != 1 {
			fmt.Printf("  #%d %s slot=%d @bit%d — attendu NEW, STOP\n", k, tn[typ], slot, br.BitPos())
			break
		}
		start := br.BitPos()
		t := filmdec.TraverseEntity(br, reg, 0)
		name, dcomp := "", ""
		if arch, ok := reg.Archetype(int(t.TypeIndex)); ok {
			if len(arch.Components) > 0 {
				name = arch.Components[0]
			}
			if t.DesyncAt >= 0 && t.DesyncAt < len(arch.Components) {
				dcomp = arch.Components[t.DesyncAt]
			}
		}
		if t.DesyncAt != -1 {
			fmt.Printf("  #%d NEW slot=%d ti=%d DÉSYNC comp[%d]=%q (ok=%d) body=%db — STOP\n",
				k, slot, t.TypeIndex, t.DesyncAt, dcomp, len(t.Comps), br.BitPos()-start)
			break
		}
		nOK++
		if k < 60 {
			fmt.Printf("  #%d NEW slot=%d ti=%-3d body=%-4db comps=%d %s\n",
				k, slot, t.TypeIndex, br.BitPos()-start, len(t.Comps), name)
		}
	}
	fmt.Printf("DÉROULÉ : %d records NEW propres avant blocage\n\n", nOK)
}

// scoreKeyframe déroule le keyframe (loop manuel) et rend le nombre de records NEW propres à
// slots croissants + le ti/slot du blocage. Le tail et les default-states par-ti sont réglés
// par l'appelant. Sert de métrique aux probes.
func scoreKeyframe(buf []byte, reg *filmdec.Registry) (nOK int, stopTI uint32, stopSlot int) {
	br := filmdec.NewBitReader(buf)
	br.Skip(2)
	prev := -1
	for k := 0; k < 3000; k++ {
		var typ int
		if br.ReadBit() {
			typ = 3
		} else {
			typ = int(br.ReadBits(2))
		}
		if typ != 1 {
			break
		}
		slot := int(br.ReadBits(13))
		br.ReadBits(2)
		if slot <= prev {
			break
		}
		t := filmdec.TraverseEntity(br, reg, 0)
		stopTI, stopSlot = t.TypeIndex, slot
		if t.DesyncAt != -1 {
			break
		}
		nOK++
		prev = slot
	}
	return
}

// probeDefaultState cherche empiriquement la largeur de default-state d'un archétype ti qui
// maximise la chaîne de records NEW croissants contre l'oracle. Un plateau net = la largeur
// fixe du default-state de ti ; aucun gain = default-state stub (0) et le blocage est ailleurs
// (composant/masque de ti).
func probeDefaultState(buf []byte, reg *filmdec.Registry, ti uint32) {
	fmt.Printf("--- PROBE default-state ti=%d (record suivant le blocage) ---\n", ti)
	filmdec.SetNewRecordTailBits(1)
	base, _, _ := scoreKeyframe(buf, reg)
	bestD, bestC := 0, base
	for d := 0; d <= 300; d++ {
		filmdec.SetDefaultStateBitsForTI(ti, d)
		if n, _, _ := scoreKeyframe(buf, reg); n > bestC {
			bestC, bestD = n, d
		}
	}
	filmdec.SetDefaultStateBitsForTI(ti, bestD)
	n, stopTI, stopSlot := scoreKeyframe(buf, reg)
	fmt.Printf("  base=%d ; meilleur ti=%d default-state=%d bits -> %d NEW croissants (prochain blocage ti=%d slot=%d)\n\n",
		base, ti, bestD, n, stopTI, stopSlot)
	filmdec.SetDefaultStateBitsForTI(ti, 0)
	filmdec.SetNewRecordTailBits(0)
}
