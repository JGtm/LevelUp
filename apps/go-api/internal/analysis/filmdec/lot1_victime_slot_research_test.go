package filmdec

// lot1_victime_slot_research_test.go — LOT 1 : RESOUDRE la reference domaine-1 de
// damage_aftermath en SLOT (donc en joueur, via le pont slot->xuid existant). Intuition
// utilisateur : « on a deja le slot-joueur ». Le maillon manquant n'est pas slot->joueur mais
// REF->slot : la reference est un INDEX de la table domaine-1, et la victime se reconstruit
// (workflow) comme (generation<<30)|(base + index). On CHERCHE la base : celle pour laquelle
// (base+index) tombe sur un bipede LIE dans le monde (archetype 35) est la bonne.
//
// Instrument : par chunk, monde = images-cles + passe de tick (comme killsource.timeline le
// ferait). Pour chaque evenement damage_aftermath, on lit l'index (et la generation) de chaque
// ref d'en-tete, et pour un BALAYAGE de base, on compte si le monde a un bipede a (base+index).
// La base qui maximise les touches bipedes RESOUT la reference -> slot -> joueur.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris.

import (
	"os"
	"testing"
)

func TestLot1VictimeSlot(t *testing.T) {
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
	// bases candidates (0, 256, 384, 448, 512 = debut plage bipede, etc.)
	bases := []int{0, 128, 256, 384, 448, 480, 500, 508, 510, 512, 514, 516, 520, 544, 576}
	// pour chaque (ref#, base) : nombre de fois ou (base+index) est un bipede lie.
	type key struct {
		ref  int
		base int
	}
	bipedeHits := map[key]int{}
	totalRef := [3]int{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
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
		cfg2 := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg2)
			}
		}
		// le monde a l'etat de fin de tick du chunk (approx : recyclage intra-chunk ignore)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xC0 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 0 {
				continue
			}
			// 3 refs d'en-tete : dom1 sonde, dom1 sonde, dom7. On capte l'index de chacune.
			idx := [3]int{-1, -1, -1}
			// ref0 dom1
			if br.ReadBit() {
				w := 13
				if br.ReadBit() {
					w = 9
				}
				idx[0] = int(br.ReadBits(uint(w)))
				br.Skip(2)
			}
			// ref1 dom1
			if br.ReadBit() {
				w := 13
				if br.ReadBit() {
					w = 9
				}
				idx[1] = int(br.ReadBits(uint(w)))
				br.Skip(2)
			}
			// ref2 dom7
			if br.ReadBit() {
				idx[2] = int(br.ReadBits(13))
				br.Skip(2)
			}
			for r := 0; r < 3; r++ {
				if idx[r] < 0 {
					continue
				}
				totalRef[r]++
				for _, b := range bases {
					slot := b + idx[r]
					if slot < 0 || slot >= 8192 {
						continue
					}
					if ti, ok := wBase.ArchetypeForSlot(uint32(slot)); ok && ti == BipedTypeIndex {
						bipedeHits[key{r, b}]++
					}
				}
			}
		}
	}
	for r := 0; r < 3; r++ {
		if totalRef[r] == 0 {
			continue
		}
		t.Logf("REF#%d (%d references presentes) : taux de (base+index) == bipede lie, par base :", r, totalRef[r])
		best, bestB := 0, -1
		for _, b := range bases {
			h := bipedeHits[key{r, b}]
			if h > 0 {
				t.Logf("    base=%d : %d / %d (%.1f %%)", b, h, totalRef[r], lot1Pct(h, totalRef[r]))
			}
			if h > best {
				best, bestB = h, b
			}
		}
		t.Logf("  -> MEILLEURE base ref#%d = %d (%d/%d = %.1f %% de bipedes lies)",
			r, bestB, best, totalRef[r], lot1Pct(best, totalRef[r]))
	}
}
