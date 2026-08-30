package filmdec

// lot1_carte_research_test.go — LOT 1 : LA CARTE DES COMPOSANTS DES TRAMES CADREES.
// Suite de lot1_cadrage_research_test.go (scinde pour le seuil de 500 lignes) : le
// decodage INTEGRE (un seul monde, chaque famille sous son cadrage) et le recensement
// des composants portes par les records aboutis — la carte L1-C3 du plan.

import (
	"fmt"
	"os"
	"testing"
)

// TestLot1CarteComposants — sous le cadrage ETABLI (0xC2 -> k=6, 0xD2 -> k=8, mesure sur
// 2 films), la CARTE des composants que portent les records aboutis de ces trames : c'est
// elle qui dit ou vivent l'arme, la visee et la victime. Recensement pur (L1-C3), sans seuil.
// Publie, par famille : (ti, composant) -> occurrences, sur les records DELTA aboutis.
func TestLot1CarteComposants(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	familleK := map[byte]int{0xC2: 6, 0xD2: 8}
	comps := map[byte]map[string]int{}
	recTIs := map[byte]map[string]int{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		snap := wBase.Snapshot()
		// UN SEUL MONDE, paquets decodes DANS L'ORDRE DU CHUNK, chaque famille sous son
		// cadrage : les transitoires vises par 0xC2/0xD2 sont crees par des records NEW a
		// l'interieur des trames de tick 0xA0 — sans les decoder aussi, leurs slots restent
		// non lies. Les familles au cadrage NON etabli sont sautees (les decoder mal
		// cadrees empoisonnerait les liaisons). C'est le prototype du decodeur integre.
		w := NewWorld(reg)
		w.Restore(snap)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			k, vise := familleK[pay[0]]
			switch {
			case vise:
				// famille cible : son cadrage etabli
			case pay[0]&0x40 == 0:
				k = DefaultPacketPreambleBits // bit 1 = 0 : cadrage historique (0xA0, 0x80, 0x89...)
			default:
				continue // bit 1 = 1 et cadrage non etabli : sauter
			}
			cfg := DefaultFrameConfig()
			cfg.PacketPreambleBits = k
			br := NewBitReader(pay)
			recs, _ := DecodeFrameRecords(br, w, cfg)
			if !vise {
				continue // les 0xA0 ne sont decodees que pour leurs liaisons
			}
			for i := range recs {
				r := &recs[i]
				if r.DesyncAt != -1 {
					continue
				}
				if recTIs[pay[0]] == nil {
					recTIs[pay[0]], comps[pay[0]] = map[string]int{}, map[string]int{}
				}
				genre := map[int]string{recNew: "new", recDel: "del", recDelta: "delta"}[r.Type]
				recTIs[pay[0]][fmt.Sprintf("%s ti=%d", genre, r.TypeIndex)]++
				for _, cp := range r.Trace.Comps {
					comps[pay[0]][fmt.Sprintf("ti=%d i%d %s", r.TypeIndex, cp.Index, cp.Name)]++
				}
			}
		}
	}
	for _, o := range []byte{0xC2, 0xD2} {
		t.Logf("  0x%02X (k=%d) — records aboutis par (genre, ti) :", o, familleK[o])
		for _, l := range lot1TopN(recTIs[o], 10) {
			t.Logf("      %s", l)
		}
		t.Logf("  0x%02X — composants presents (ti, index, nom) :", o)
		for _, l := range lot1TopN(comps[o], 25) {
			t.Logf("      %s", l)
		}
	}
}
